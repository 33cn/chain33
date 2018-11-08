package tasks

import (
	"errors"
	"fmt"
	"os/exec"

	"gitlab.33.cn/chain33/chain33/util"
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

func (this *UpdateInitFileTask) GetName() string {
	return "UpdateInitFileTask"
}

func (this *UpdateInitFileTask) Execute() error {
	// 1. 检查目标文件夹是否存在，如果不存在则不扫描
	if !util.CheckPathExisted(this.Folder) {
		mlog.Error("UpdateInitFileTask Execute failed", "folder", this.Folder)
		return errors.New("NotExisted")
	}
	funcs := []func() error{
		this.init,
		this.genInitFile,
		this.formatInitFile,
	}
	for _, fn := range funcs {
		if err := fn(); err != nil {
			return err
		}
	}
	mlog.Info("Success generate init.go", "filename", this.initFile)
	return nil
}

func (this *UpdateInitFileTask) init() error {
	this.initFile = fmt.Sprintf("%sinit/init.go", this.Folder)
	this.itemDatas = make([]*itemData, 0)
	return nil
}

func (this *UpdateInitFileTask) genInitFile() error {
	if err := util.MakeDir(this.initFile); err != nil {
		return err
	}
	var importStr, content string
	for _, item := range this.itemDatas {
		importStr += fmt.Sprintf("_ \"%s\"\r\n", item.path)
	}
	content = fmt.Sprintf("package init \r\n\r\nimport (\r\n%s)", importStr)

	util.DeleteFile(this.initFile)
	_, err := util.WriteStringToFile(this.initFile, content)
	return err
}

func (this *UpdateInitFileTask) formatInitFile() error {
	cmd := exec.Command("gofmt", "-l", "-s", "-w", this.initFile)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("Format init.go failed. file %v", this.initFile)
	}
	return nil
}
