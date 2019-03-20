// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package base

var (
	//CodeFileManager 类型文件
	CodeFileManager = map[string][]ICodeFile{}
)

// ICodeFile code file interface
type ICodeFile interface {
	GetCodeType() string
	GetDirName() string
	GetFiles() map[string]string //key:filename, val:file content
	GetDirReplaceTags() []string
	GetFileReplaceTags() []string
}

//RegisterCodeFile regeister code file
func RegisterCodeFile(filer ICodeFile) {

	codeType := filer.GetCodeType()
	fileArr := CodeFileManager[codeType]
	fileArr = append(fileArr, filer)
	CodeFileManager[codeType] = fileArr
}

// CodeFile 基础类
type CodeFile struct {
}

//GetCodeType get cody type
func (CodeFile) GetCodeType() string {
	return ""
}

//GetDirName get directory name
func (CodeFile) GetDirName() string {

	return ""
}

//GetFiles get files
func (CodeFile) GetFiles() map[string]string {

	return nil
}

//GetDirReplaceTags get directory replace tags
func (CodeFile) GetDirReplaceTags() []string {

	return nil
}

//GetFileReplaceTags get file replace tags
func (CodeFile) GetFileReplaceTags() []string {

	return nil
}
