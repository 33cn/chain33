// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tasks

import (
	"os"
	"os/exec"
	"path"
	"path/filepath"
)

// FormatDappSourceTask 利用Go工具，对生成出来的Go源码进行格式化
type FormatDappSourceTask struct {
	TaskBase
	OutputFolder string
}

//GetName 获取name
func (f *FormatDappSourceTask) GetName() string {
	return "FormatDappSourceTask"
}

//Execute 执行
func (f *FormatDappSourceTask) Execute() error {
	mlog.Info("Execute format dapp source task.")
	err := filepath.Walk(f.OutputFolder, func(fpath string, info os.FileInfo, err error) error {
		if info == nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		ext := path.Ext(fpath)
		if ext != ".go" { // 仅对go的源码文件进行格式化
			return nil
		}
		cmd := exec.Command("gofmt", "-l", "-s", "-w", fpath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	})
	return err
}
