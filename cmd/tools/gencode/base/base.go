// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package base

var (
	CodeFileManager = map[string][]ICodeFile{}
)

type ICodeFile interface {
	GetCodeType() string
	GetDirName() string
	GetFiles() map[string]string //key:filename, val:file content
	GetReplaceTags() []string
}

func RegisterCodeFile(filer ICodeFile) {

	codeType := filer.GetCodeType()
	fileArr := CodeFileManager[codeType]
	fileArr = append(fileArr, filer)
	CodeFileManager[codeType] = fileArr
}

type BaseCodeFile struct {
}

func (BaseCodeFile) GetCodeType() string {
	return ""
}

func (BaseCodeFile) GetDirName() string {

	return ""
}

func (BaseCodeFile) GetFiles() map[string]string {

	return nil
}

func (BaseCodeFile) GetReplaceTags() []string {

	return nil
}
