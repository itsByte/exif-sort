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
	in := flag.String("in", "in", "input directory")
	out := flag.String("out", "out", "output directory")
	flag.Parse()
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
	err = iterateFolder(*in, *et, *out)
	if err != nil {
		log.Println("Error while iterating folder: ", err)
		return
	}
}

func checkExif(path string, et exiftool.Exiftool) (string, error) {
	fileInfos := et.ExtractMetadata(path)
	if len(fileInfos) > 1 {
		return "", errors.New("more than one file has been scanned")
	}
	fileInfo := fileInfos[0]
	if fileInfo.Err != nil {
		return "", fileInfo.Err
	}
	model, err := fileInfo.GetString("Model")
	if err == exiftool.ErrKeyNotFound {
		return "Unknown", nil
	}
	if err != nil {
		return "", err
	}

	return model, nil
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

func iterateFolder(in string, et exiftool.Exiftool, out string) error {
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
		model, err := checkExif(path, et)
		if err != nil {
			return err
		}
		checkFolder(out, model)
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
