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
		fmt.Printf("Error when intializing ExifTool: %v\n", err)
		return
	}
	defer et.Close()
	in := flag.String("in", "in", "input directory")
	out := flag.String("out", "out", "output directory")
	flag.Parse()
	*in, err = filepath.Abs(*in)
	if err != nil {
		fmt.Println("Error in input path")
		return
	}
	*out, err = filepath.Abs(*out)
	if err != nil {
		fmt.Printf("Error in output path")
		return
	}
	fmt.Println("Directory to scan: ", *in)
	fmt.Println("Output directory: ", *out)
	fmt.Println("Are you sure? Enter to continue")
	fmt.Scanln()
	if _, err := os.Stat(*out); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(*out, os.ModePerm)
		if err != nil {
			log.Println(err)
		}
	}
	iterateFolder(*in, *et, *out)
}

func checkExif(path string, et exiftool.Exiftool) (string, error) {
	fileInfos := et.ExtractMetadata(path)
	if len(fileInfos) > 1 {
		return "", errors.New("more than one file has been scanned")
	}
	fileInfo := fileInfos[0]
	if fileInfo.Err != nil {
		fmt.Printf("Error concerning %v: %v\n", fileInfo.File, fileInfo.Err)
		return "", fileInfo.Err
	}
	model, err := fileInfo.GetString("Model")
	if err != nil {
		return "", err
	}
	return model, nil
}

func checkFolder(outdir string, model string) {
	modelpath := filepath.Join(outdir, model)
	if _, err := os.Stat(modelpath); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(modelpath, os.ModePerm)
		if err != nil {
			log.Println(err)
		}
	}
}

func copyImage(src string, dst string) {
	srcStat, err := os.Stat(src)
	if err != nil {
		return
	}

	if !srcStat.Mode().IsRegular() {
		return
	}

	source, err := os.Open(src)
	if err != nil {
		return
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return
	}
	defer destination.Close()
	_, err = io.Copy(destination, source)
	if err != nil {
		println("Error")
	}
}

func iterateFolder(in string, et exiftool.Exiftool, out string) {
	err := filepath.Walk(in, func(path string, f fs.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("Prevent panic by handling failure accessing a path %q: %v\n", path, err)
			return err
		}
		if err != nil {
			return err
		}
		if f.IsDir() {
			if path == out {
				return filepath.SkipDir
			}
			fmt.Println("Scanning Directory:", f.Name())
			return nil
		}
		model, err := checkExif(path, et)
		if err != nil {
			return err
		}
		fmt.Println(model)
		checkFolder(out, model)
		copyImage(path, filepath.Join(out, model, f.Name()))
		return nil
	})

	if err != nil {
		fmt.Printf("Error walking the path")
		return
	}
}
