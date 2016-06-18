package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

type FileInfos []os.FileInfo
type ByName struct{ FileInfos }

func (fi ByName) Len() int {
	return len(fi.FileInfos)
}
func (fi ByName) Swap(i, j int) {
	fi.FileInfos[i], fi.FileInfos[j] = fi.FileInfos[j], fi.FileInfos[i]
}
func (fi ByName) Less(i, j int) bool {
	return fi.FileInfos[j].ModTime().Unix() < fi.FileInfos[i].ModTime().Unix()
}

func IsDirectory(name string) (isDir bool, err error) {
	fInfo, err := os.Stat(name)
	if err != nil {
		return false, err
	}
	return fInfo.IsDir(), nil
}

func Exists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

func ListFiles(arg string) (fileList []string, err error) {
	var curDir, _ = os.Getwd()
	curDir += "/"

	if arg == "" {
		arg = curDir
	}

	var dirName, filePattern = path.Split(arg)
	if dirName == "" {
		dirName = curDir
	}

	var isDir, _ = IsDirectory(dirName + filePattern)
	if isDir == true {
		dirName = dirName + filePattern
		filePattern = ""
	}

	fileInfos, err := ioutil.ReadDir(dirName)
	if err != nil {
		fmt.Errorf("Directory cannot read %s\n", err)
		os.Exit(1)
	}

	var filePaths []string = make([]string, 0)
	for _, fileInfo := range fileInfos {
		var findName = (fileInfo).Name()
		path := filepath.Join(dirName, findName)
		isDir, _ = IsDirectory(path)
		if isDir {
			continue
		}
		filePaths = append(filePaths, path)
	}

	return filePaths, nil
}

func Unzip(src, dest string) error {
	reader, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer reader.Close()

	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	for _, file := range reader.File {
		rc, err := file.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		// TODO: judge encoding for windows-jp
		filename, err := FromShiftJIS(file.Name)
		if err != nil {
			return err
		}
		path := filepath.Join(dest, filename)
		// fmt.Printf("%s\n", path)
		dirPath := filepath.Dir(path)
		if Exists(dirPath) == false {
			os.MkdirAll(dirPath, file.Mode())
		}

		if file.FileInfo().IsDir() {
			os.MkdirAll(path, file.Mode())
			continue
		}

		f, err := os.OpenFile(
			path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(f, rc)
		if err != nil {
			return err
		}
	}

	return nil
}

func transformEncoding(rawReader io.Reader, trans transform.Transformer) (string, error) {
	ret, err := ioutil.ReadAll(transform.NewReader(rawReader, trans))
	if err == nil {
		return string(ret), nil
	} else {
		return "", err
	}
}

// Convert a string encoding from ShiftJIS to UTF-8
func FromShiftJIS(str string) (string, error) {
	return transformEncoding(strings.NewReader(str), japanese.ShiftJIS.NewDecoder())
}

func main() {
	var arg string

	flag.StringVar(&arg, "f", "", "SearchPattern")
	flag.Parse()
	//
	var fileList, _ = ListFiles(arg)
	//
	for _, filePath := range fileList {
		fmt.Printf("%s\n", filePath)
		pos := strings.LastIndex(filePath, ".")
		dirName := filePath[:pos]
		ext := filepath.Ext(filePath)
		if ext != ".zip" {
			continue
		}
		err := Unzip(filePath, dirName)
		if err != nil {
			log.Fatal(err)
		}
	}
}
