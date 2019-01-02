// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tasks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/33cn/chain33/cmd/tools/types"
	"github.com/33cn/chain33/cmd/tools/util"
)

//CopyTemplateToOutputTask ...
type CopyTemplateToOutputTask struct {
	TaskBase
	TemplatePath string
	OutputPath   string
	ProjectName  string
	ClassName    string
}

//GetName 获取name
func (c *CopyTemplateToOutputTask) GetName() string {
	return "CopyTemplateToOutputTask"
}

//Execute 执行
func (c *CopyTemplateToOutputTask) Execute() error {
	mlog.Info("Execute copy template task.")
	err := filepath.Walk(c.TemplatePath, func(path string, info os.FileInfo, err error) error {
		if info == nil {
			return err
		}
		if c.TemplatePath == path {
			return nil
		}
		if info.IsDir() {
			outFolder := fmt.Sprintf("%s/%s/", c.OutputPath, info.Name())
			if err := util.MakeDir(outFolder); err != nil {
				mlog.Error("MakeDir failed", "error", err, "outFolder", outFolder)
				return err
			}
		} else {
			srcFile := path
			path = strings.Replace(path, types.TagClassName, c.ClassName, -1)
			path = strings.Replace(path, ".tmp", "", -1)
			dstFile := strings.Replace(path, c.TemplatePath, c.OutputPath, -1)
			if _, err := util.CopyFile(srcFile, dstFile); err != nil {
				mlog.Error("CopyFile failed", "error", err, "srcFile", srcFile, "dstFile", dstFile)
				return err
			}
		}
		return nil
	})
	return err
}
