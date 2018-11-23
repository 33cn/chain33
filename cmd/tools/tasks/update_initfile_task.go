// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tasks

import (
	"errors"
	"fmt"
	"os/exec"

	"github.com/33cn/chain33/util"
)

type itemData struct {
	path string
}

// UpdateInitFileTask 通过扫描本地目录更新init.go文件
type UpdateInitFileTask struct {
	TaskBase
	Folder string

	initFile  string
	itemDatas []*itemData
}

//GetName 获取name
func (up *UpdateInitFileTask) GetName() string {
	return "UpdateInitFileTask"
}

//Execute 执行
func (up *UpdateInitFileTask) Execute() error {
	// 1. 检查目标文件夹是否存在，如果不存在则不扫描
	if !util.CheckPathExisted(up.Folder) {
		mlog.Error("UpdateInitFileTask Execute failed", "folder", up.Folder)
		return errors.New("NotExisted")
	}
	funcs := []func() error{
		up.init,
		up.genInitFile,
		up.formatInitFile,
	}
	for _, fn := range funcs {
		if err := fn(); err != nil {
			return err
		}
	}
	mlog.Info("Success generate init.go", "filename", up.initFile)
	return nil
}

func (up *UpdateInitFileTask) init() error {
	up.initFile = fmt.Sprintf("%sinit/init.go", up.Folder)
	up.itemDatas = make([]*itemData, 0)
	return nil
}

func (up *UpdateInitFileTask) genInitFile() error {
	if err := util.MakeDir(up.initFile); err != nil {
		return err
	}
	var importStr, content string
	for _, item := range up.itemDatas {
		importStr += fmt.Sprintf("_ \"%s\"\r\n", item.path)
	}
	content = fmt.Sprintf("package init \r\n\r\nimport (\r\n%s)", importStr)

	util.DeleteFile(up.initFile)
	_, err := util.WriteStringToFile(up.initFile, content)
	return err
}

func (up *UpdateInitFileTask) formatInitFile() error {
	cmd := exec.Command("gofmt", "-l", "-s", "-w", up.initFile)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("Format init.go failed. file %v", up.initFile)
	}
	return nil
}
