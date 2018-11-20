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

//UpdateInitFileTask 通过扫描本地目录更新init.go文件
type UpdateInitFileTask struct {
	TaskBase
	Folder string

	initFile  string
	itemDatas []*itemData
}

//GetName 获取name
func (u *UpdateInitFileTask) GetName() string {
	return "UpdateInitFileTask"
}

//Execute 执行
func (u *UpdateInitFileTask) Execute() error {
	// 1. 检查目标文件夹是否存在，如果不存在则不扫描
	if !util.CheckPathExisted(u.Folder) {
		mlog.Error("UpdateInitFileTask Execute failed", "folder", u.Folder)
		return errors.New("NotExisted")
	}
	funcs := []func() error{
		u.init,
		u.genInitFile,
		u.formatInitFile,
	}
	for _, fn := range funcs {
		if err := fn(); err != nil {
			return err
		}
	}
	mlog.Info("Success generate init.go", "filename", u.initFile)
	return nil
}

func (u *UpdateInitFileTask) init() error {
	u.initFile = fmt.Sprintf("%sinit/init.go", u.Folder)
	u.itemDatas = make([]*itemData, 0)
	return nil
}

func (u *UpdateInitFileTask) genInitFile() error {
	if err := util.MakeDir(u.initFile); err != nil {
		return err
	}
	var importStr, content string
	for _, item := range u.itemDatas {
		importStr += fmt.Sprintf("_ \"%s\"\r\n", item.path)
	}
	content = fmt.Sprintf("package init \r\n\r\nimport (\r\n%s)", importStr)

	util.DeleteFile(u.initFile)
	_, err := util.WriteStringToFile(u.initFile, content)
	return err
}

func (u *UpdateInitFileTask) formatInitFile() error {
	cmd := exec.Command("gofmt", "-l", "-s", "-w", u.initFile)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("Format init.go failed. file %v", u.initFile)
	}
	return nil
}
