// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package utils

import (
	"encoding/pem"
	"fmt"
	"io/ioutil"
)

// ReadFile 读取文件
func ReadFile(file string) ([]byte, error) {
	fileCont, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("Could not read file %s, err %s", file, err)
	}

	return fileCont, nil
}

// ReadPemFile 读取pem文件
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
