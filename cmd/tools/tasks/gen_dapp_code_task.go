// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tasks

import (
	"github.com/33cn/chain33/cmd/tools/gencode"
	"path/filepath"
)

// GenDappCodeTask 通过生成好的pb.go和预先设计的模板，生成反射程序源码
type GenDappCodeTask struct {
	TaskBase
	DappName string
	DappDir string
	ProtoFile string
	replacePairs map[string]string
}

//GetName 获取name
func (c *GenDappCodeTask) GetName() string {
	return "GenDappCodeTask"
}

//Execute 执行
func (c *GenDappCodeTask) Execute() error {
	mlog.Info("Execute generate dapp code task.")
	var err error

	return err
}


func (c *GenDappCodeTask) formatDappReplacePairs() {

}


func (c *GenDappCodeTask) genDappCode() {

	codeTypes := gencode.GetCodeFilesWithType("dapp")

	for _, code := range codeTypes {

		dirPath := filepath.Join(c.DappDir, code.GetDirName())

	}
}
