package tasks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gitlab.33.cn/chain33/chain33/cmd/tools/types"
	"gitlab.33.cn/chain33/chain33/util"
)

type CopyTemplateToOutputTask struct {
	TaskBase
	TemplatePath string
	OutputPath   string
	ProjectName  string
	ClassName    string
}

func (this *CopyTemplateToOutputTask) GetName() string {
	return "CopyTemplateToOutputTask"
}

func (this *CopyTemplateToOutputTask) Execute() error {
	mlog.Info("Execute copy template task.")
	err := filepath.Walk(this.TemplatePath, func(path string, info os.FileInfo, err error) error {
		if info == nil {
			return err
		}
		if this.TemplatePath == path {
			return nil
		}
		if info.IsDir() {
			outFolder := fmt.Sprintf("%s/%s/", this.OutputPath, info.Name())
			if err := util.MakeDir(outFolder); err != nil {
				mlog.Error("MakeDir failed", "error", err, "outFolder", outFolder)
				return err
			}
		} else {
			srcFile := path
			path = strings.Replace(path, types.TagClassName, this.ClassName, -1)
			path = strings.Replace(path, ".tmp", "", -1)
			dstFile := strings.Replace(path, this.TemplatePath, this.OutputPath, -1)
			if _, err := util.CopyFile(srcFile, dstFile); err != nil {
				mlog.Error("CopyFile failed", "error", err, "srcFile", srcFile, "dstFile", dstFile)
				return err
			}
		}
		return nil
	})
	return err
}
