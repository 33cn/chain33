package util

import (
	"io"
	"os"
	"path/filepath"
)

// CheckPathExists 检查文件夹是否存在
func CheckPathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
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
