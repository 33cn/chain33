// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package util

import (
	"io"
	"os"
	"path/filepath"
)

// CheckPathExists 检查文件夹是否存在
func DirExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func CheckPathExisted(path string) bool {
	existed, _ := DirExists(path)
	return existed
}

func CheckFileExists(fileName string) (bool, error) {
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		return false, err
	}
	return true, nil
}

func DeleteFile(fileName string) error {
	if existed, _ := CheckFileExists(fileName); existed {
		return os.Remove(fileName)
	}
	return nil
}

func OpenFile(fileName string) (*os.File, error) {
	var file *os.File
	var err error
	if existed, _ := CheckFileExists(fileName); !existed {
		file, err = os.Create(fileName)
		if err != nil {
			return nil, err
		}
	} else {
		file, err = os.OpenFile(fileName, os.O_RDWR, os.ModePerm)
		if err != nil {
			return nil, err
		}
	}
	return file, nil
}

func MakeDir(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, os.ModePerm)
}

func Pwd() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(err)
	}
	return dir
}

func DirEmpty(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}
