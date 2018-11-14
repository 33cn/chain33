//Copyright Fuzamei Corp. 2018 All Rights Reserved.
//Use of this source code is governed by a BSD-style
//license that can be found in the LICENSE file.
package strategy

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/33cn/chain33/cmd/tools/tasks"
	"github.com/33cn/chain33/cmd/tools/types"
	"github.com/pkg/errors"
)

// createPluginStrategy 根據模板配置文件，創建一個完整的項目工程
type createPluginStrategy struct {
	strategyBasic

	projName    string // 项目名称
	outRootPath string // 项目生成的根目录

}

func (this *createPluginStrategy) Run() error {
	fmt.Println("Begin run chain33 create plugin project mode.")
	defer fmt.Println("Run chain33 create plugin project mode finish.")
	if err := this.initMember(); err != nil {
		return err
	}
	return this.rumImpl()
}

func (this *createPluginStrategy) initMember() error {
	gopath := os.Getenv("GOPATH")
	if len(gopath) <= 0 {
		return errors.New("Can't find GOPATH")
	}
	this.outRootPath = filepath.Join(gopath, "/src/github.com/33cn")
	this.projName, _ = this.getParam(types.KeyProjectName)
	return nil
}

func (this *createPluginStrategy) rumImpl() error {
	var err error
	task := this.buildTask()
	for {
		if task == nil {
			break
		}
		err = task.Execute()
		if err != nil {
			mlog.Error("Execute command failed.", "error", err, "taskname", task.GetName())
			break
		}
		task = task.Next()
	}
	return err
}

func (this *createPluginStrategy) buildTask() tasks.Task {
	taskSlice := make([]tasks.Task, 0)
	taskSlice = append(taskSlice,
		&tasks.CreateFileFromStrTemplateTask{
			SourceStr:  CPFT_MAIN_GO,
			OutputFile: fmt.Sprintf("%s/%s/main.go", this.outRootPath, this.projName),
			ReplaceKeyPairs: map[string]string{
				types.TagProjectName: this.projName,
			},
		},
		&tasks.CreateFileFromStrTemplateTask{
			SourceStr:  CPFT_CFG_TOML,
			OutputFile: fmt.Sprintf("%s/%s/%s.toml", this.outRootPath, this.projName, this.projName),
			ReplaceKeyPairs: map[string]string{
				types.TagProjectName: this.projName,
			},
		},
		&tasks.CreateFileFromStrTemplateTask{
			SourceStr:     CPFT_RUNMAIN,
			BlockStrBegin: CPFT_RUNMAIN_BLOCK + "`",
			BlockStrEnd:   "`",
			OutputFile:    fmt.Sprintf("%s/%s/%s.go", this.outRootPath, this.projName, this.projName),
			ReplaceKeyPairs: map[string]string{
				types.TagProjectName: this.projName,
			},
		},
	)

	task := taskSlice[0]
	sliceLen := len(taskSlice)
	for n := 1; n < sliceLen; n++ {
		task.SetNext(taskSlice[n])
		task = taskSlice[n]
	}
	return taskSlice[0]
}
