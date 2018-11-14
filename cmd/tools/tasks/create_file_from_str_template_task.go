/*
 * Copyright Fuzamei Corp. 2018 All Rights Reserved.
 * Use of this source code is governed by a BSD-style
 * license that can be found in the LICENSE file.
 */

package tasks

import (
	"fmt"
	"strings"

	"github.com/33cn/chain33/util"
)

// CreateFileFromStrTemplateTask 从指定的模板字符串中创建目标文件的任务
type CreateFileFromStrTemplateTask struct {
	TaskBase
	SourceStr       string
	OutputFile      string
	BlockStrBegin   string // Block会将SourceStr包含在内部
	BlockStrEnd     string
	ReplaceKeyPairs map[string]string
	fileContent     string
}

func (this *CreateFileFromStrTemplateTask) GetName() string {
	return "CreateFileFromStrTemplateTask"
}

func (this *CreateFileFromStrTemplateTask) Execute() error {
	if len(this.BlockStrBegin) > 0 && len(this.BlockStrEnd) > 0 {
		this.SourceStr = fmt.Sprintf("%s%s%s", this.BlockStrBegin, this.SourceStr, this.BlockStrEnd)
	}
	this.fileContent = this.SourceStr
	for key, value := range this.ReplaceKeyPairs {
		this.fileContent = strings.Replace(this.fileContent, key, value, -1)
	}
	if err := util.MakeDir(this.OutputFile); err != nil {
		return err
	}
	util.DeleteFile(this.OutputFile)
	_, err := util.WriteStringToFile(this.OutputFile, this.fileContent)
	return err
}
