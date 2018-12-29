// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tasks

import (
	"github.com/33cn/chain33/cmd/tools/util"
)

// CheckFileExistedTask 检测文件是否存在
type CheckFileExistedTask struct {
	TaskBase
	FileName string
}

//GetName 获取name
func (c *CheckFileExistedTask) GetName() string {
	return "CheckFileExistedTask"
}

//Execute 执行
func (c *CheckFileExistedTask) Execute() error {
	mlog.Info("Execute file existed task.", "file", c.FileName)
	_, err := util.CheckFileExists(c.FileName)
	return err
}
