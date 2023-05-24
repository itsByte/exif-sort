package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/barasher/go-exiftool"
)

func main() {
	et, err := exiftool.NewExiftool()
	if err != nil {
		log.Println("Error when intializing ExifTool: ", err)
		return
	}
	defer et.Close()
	in := flag.String("in", "indir", "Input directory")
	out := flag.String("out", "outdir", "Output directory")
	parsesize := flag.Bool("parsesize", false, "Sort by size when the make model is Unknown. Useful for screenshots")
	flag.Usage = func() {
		fmt.Println("Usage: exif-sort --in {input_dir} --out {output_dir}")
	}
	flag.Parse()
	if *in == "indir" || *out == "outdir" {
		flag.Usage()
		return
	}
	*in, err = filepath.Abs(*in)
	if err != nil {
		log.Println("Error obtaining absolute input path: ", err)
		return
	}
	*out, err = filepath.Abs(*out)
	if err != nil {
		log.Println("Error obtaining absolute output path: ", err)
		return
	}
	fmt.Println("Directory to scan: ", *in)
	fmt.Println("Output directory: ", *out)
	fmt.Println("Are you sure? Enter to continue")
	fmt.Scanln()
	if _, err := os.Stat(*out); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(*out, os.ModePerm)
		if err != nil {
			log.Println("Error creating output directory: ", err)
			return
		}
	}
	err = iterateFolder(*in, *et, *out, *parsesize)
	if err != nil {
		log.Println("Error while iterating folder: ", err)
		return
	}
}

func getExif(src string, et exiftool.Exiftool) (exiftool.FileMetadata, error) {
	fileInfos := et.ExtractMetadata(src)
	if len(fileInfos) > 1 {
		return exiftool.EmptyFileMetadata(), errors.New("more than one file has been scanned")
	}
	fileInfo := fileInfos[0]
	if fileInfo.Err != nil {
		return exiftool.EmptyFileMetadata(), fileInfo.Err
	}
	return fileInfo, nil
}

func getField(fileInfo exiftool.FileMetadata, field string) (string, error) {
	value, err := fileInfo.GetString(field)
	if err == exiftool.ErrKeyNotFound {
		return "Unknown", nil
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

func checkFolder(folder string) error {
	if _, err := os.Stat(folder); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(folder, os.ModePerm)
		if err != nil {
			return err
		}
	}
	return nil
}

func copyImage(src string, dest string) error {
	srcStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !srcStat.Mode().IsRegular() {
		return err
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dest)
	if err != nil {
		return err
	}

	defer destination.Close()
	_, err = io.Copy(destination, source)
	if err != nil {
		return err
	}
	return nil
}

func iterateFolder(in string, et exiftool.Exiftool, out string, parsesize bool) error {
	err := filepath.Walk(in, func(src string, f fs.FileInfo, err error) error {
		if err != nil {
			log.Println("Failure accessing ", src, ": ", err)
			return err
		}
		if f.IsDir() {
			if src == out {
				return filepath.SkipDir
			}
			return nil
		}
		fileInfo, err := getExif(src, et)
		if err != nil {
			return err
		}
		model, err := getField(fileInfo, "Model")
		if err != nil {
			return err
		}
		dest := filepath.Join(out, model)
		if parsesize && model == "Unknown" {
			size, err := getField(fileInfo, "ImageSize")
			if err != nil {
				return err
			}
			if size != "Unknown" {
				checkFolder(dest)
				if err != nil {
					return err
				}
				dest = filepath.Join(dest, size)
			}
		}
		err = checkFolder(dest)
		if err != nil {
			return err
		}
		err = copyImage(src, filepath.Join(dest, f.Name()))
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return err
	}
	return nil
}
