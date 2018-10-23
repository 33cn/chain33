package utils

import (
	"bufio"
	"encoding/pem"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"gitlab.33.cn/chain33/chain33/util"
)

func DirMissingOrEmpty(path string) (bool, error) {
	dirExists, err := DirExists(path)
	if err != nil {
		return false, err
	}
	if !dirExists {
		return true, nil
	}

	dirEmpty, err := DirEmpty(path)
	if err != nil {
		return false, err
	}
	if dirEmpty {
		return true, nil
	}
	return false, nil
}

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

func ReadFile(file string) ([]byte, error) {
	fileCont, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("Could not read file %s, err %s", file, err)
	}

	return fileCont, nil
}

func ReadPemFile(file string) ([]byte, error) {
	bytes, err := ReadFile(file)
	if err != nil {
		return nil, err
	}

	b, _ := pem.Decode(bytes)
	if b == nil {
		return nil, fmt.Errorf("No pem content for file %s", file)
	}

	return bytes, nil
}

func CheckFileIsExist(filename string) bool {
	var exist = true
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		exist = false
	}
	return exist
}

func DeleteFile(file string) error {
	return os.Remove(file)
}

func WriteStringToFile(file, content string) (writeLen int, err error) {
	var f *os.File
	if err = util.MakeDir(file); err != nil {
		return
	}
	util.DeleteFile(file)
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
