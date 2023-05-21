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

func getExif(path string, et exiftool.Exiftool) (exiftool.FileMetadata, error) {
	fileInfos := et.ExtractMetadata(path)
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

func checkFolder(outdir string, model string) error {
	modelpath := filepath.Join(outdir, model)
	if _, err := os.Stat(modelpath); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(modelpath, os.ModePerm)
		if err != nil {
			return err
		}
	}
	return nil
}

func copyImage(src string, dst string) error {
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

	destination, err := os.Create(dst)
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
	err := filepath.Walk(in, func(path string, f fs.FileInfo, err error) error {
		if err != nil {
			log.Println("Failure accessing ", path, ": ", err)
			return err
		}
		if f.IsDir() {
			if path == out {
				return filepath.SkipDir
			}
			return nil
		}
		fileInfo, err := getExif(path, et)
		if err != nil {
			return err
		}
		model, err := getField(fileInfo, "Model")
		if err != nil {
			return err
		}
		if parsesize && model == "Unknown" {
			size, err := getField(fileInfo, "ImageSize")
			if err != nil {
				return err
			}
			if size != "Unknown" {
				model = filepath.Join(model, size)
			}
			checkFolder(out, "Unknown")
			if err != nil {
				return err
			}
		}
		err = checkFolder(out, model)
		if err != nil {
			return err
		}
		err = copyImage(path, filepath.Join(out, model, f.Name()))
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
