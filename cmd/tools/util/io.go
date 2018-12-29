// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package util

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

//ReadFile : read file
func ReadFile(file string) ([]byte, error) {
	fileCont, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("Could not read file %s, err %s", file, err)
	}

	return fileCont, nil
}

//CheckFileIsExist : check whether the file exists or not
func CheckFileIsExist(filename string) bool {
	var exist = true
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		exist = false
	}
	return exist
}

//WriteStringToFile : write content to file
func WriteStringToFile(file, content string) (writeLen int, err error) {
	var f *os.File
	if err = MakeDir(file); err != nil {
		return
	}
	DeleteFile(file)
	if CheckFileIsExist(file) {
		f, err = os.OpenFile(file, os.O_APPEND, 0666)
	} else {
		f, err = os.Create(file)
	}
	if err != nil {
		return
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	writeLen, err = w.WriteString(content)
	w.Flush()
	return
}

//CopyFile : copy file
func CopyFile(srcFile, dstFile string) (written int64, err error) {
	src, err := os.Open(srcFile)
	if err != nil {
		return
	}
	defer src.Close()
	dst, err := os.OpenFile(dstFile, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return
	}
	defer dst.Close()
	return io.Copy(dst, src)
}
