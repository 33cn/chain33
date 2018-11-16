//Copyright Fuzamei Corp. 2018 All Rights Reserved.
//Use of this source code is governed by a BSD-style
//license that can be found in the LICENSE file.
package strategy

import (
	"fmt"
	"github.com/33cn/chain33/util"
	"os"
	"path/filepath"
	"strings"

	"github.com/33cn/chain33/cmd/tools/tasks"
	"github.com/33cn/chain33/cmd/tools/types"
	"github.com/pkg/errors"
)

// createPluginStrategy 根據模板配置文件，創建一個完整的項目工程
type createPluginStrategy struct {
	strategyBasic

	gopath          string
	projName        string // 项目名称
	execName        string // 项目中实现的执行器名称
	execNameFB      string
	className       string
	classTypeName   string
	classActionName string
	outRootPath     string // 项目生成的根目录
	projectPath     string // 生成的项目路径,是绝对路径

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
	this.gopath = gopath
	this.outRootPath = filepath.Join(gopath, "/src/github.com/33cn")
	this.projName, _ = this.getParam(types.KeyProjectName)
	this.execName, _ = this.getParam(types.KeyExecutorName)
	this.className, _ = this.getParam(types.KeyClassName)
	this.projectPath = fmt.Sprintf("%s/%s", this.outRootPath, this.projName)
	this.execNameFB, _ = util.MakeStringToUpper(this.execName, 0, 1)
	this.classTypeName = this.execNameFB + "Type"
	this.classActionName = this.execNameFB + "Action"
	return nil
}

func (this *createPluginStrategy) rumImpl() error {
	var err error
	tasks := this.buildTask()
	for _, task := range tasks {
		err = task.Execute()
		if err != nil {
			mlog.Error("Execute command failed.", "error", err, "taskname", task.GetName())
			break
		}
	}
	return err
}

func (this *createPluginStrategy) buildTask() []tasks.Task {
	// 获取项目相对于gopath/src中的目录路径
	goprojpath := strings.Replace(this.projectPath, this.gopath+"/src/", "", -1)
	taskSlice := make([]tasks.Task, 0)
	taskSlice = append(taskSlice,
		&tasks.CreateFileFromStrTemplateTask{
			SourceStr:  CPFT_MAIN_GO,
			OutputFile: fmt.Sprintf("%s/main.go", this.projectPath),
			ReplaceKeyPairs: map[string]string{
				types.TagProjectName: this.projName,
				types.TagProjectPath: goprojpath,
			},
		},
		&tasks.CreateFileFromStrTemplateTask{
			SourceStr:  CPFT_CFG_TOML,
			OutputFile: fmt.Sprintf("%s/%s.toml", this.projectPath, this.projName),
			ReplaceKeyPairs: map[string]string{
				types.TagProjectName: this.projName,
			},
		},
		&tasks.CreateFileFromStrTemplateTask{
			SourceStr:     CPFT_RUNMAIN,
			BlockStrBegin: CPFT_RUNMAIN_BLOCK + "`",
			BlockStrEnd:   "`",
			OutputFile:    fmt.Sprintf("%s/%s.go", this.projectPath, this.projName),
			ReplaceKeyPairs: map[string]string{
				types.TagProjectName: this.projName,
			},
		},
		&tasks.CreateFileFromStrTemplateTask{
			SourceStr:  CPFT_MAKEFILE,
			OutputFile: fmt.Sprintf("%s/Makefile", this.projectPath),
			ReplaceKeyPairs: map[string]string{
				types.TagProjectName: this.projName,
				types.TagProjectPath: goprojpath,
			},
		},
		&tasks.CreateFileFromStrTemplateTask{
			SourceStr:  CPFT_TRAVIS_YML,
			OutputFile: fmt.Sprintf("%s/.travis.yml", this.projectPath),
			ReplaceKeyPairs: map[string]string{
				types.TagProjectName: this.projName,
			},
		},
		&tasks.CreateFileFromStrTemplateTask{
			SourceStr:  CPFT_PLUGIN_TOML,
			OutputFile: fmt.Sprintf("%s/plugin/plugin.toml", this.projectPath),
			ReplaceKeyPairs: map[string]string{
				types.TagProjectName: this.projName,
			},
		},
		&tasks.CreateFileFromStrTemplateTask{
			SourceStr:  CPFT_CLI_MAIN,
			OutputFile: fmt.Sprintf("%s/cli/main.go", this.projectPath),
			ReplaceKeyPairs: map[string]string{
				types.TagProjectName: this.projName,
				types.TagProjectPath: goprojpath,
			},
		},
		&tasks.CreateFileFromStrTemplateTask{
			SourceStr:  CPFT_DAPP_COMMANDS,
			OutputFile: fmt.Sprintf("%s/plugin/dapp/%s/commands/cmd.go", this.projectPath, this.execName),
			ReplaceKeyPairs: map[string]string{
				types.TagProjectName: this.projName,
			},
		},
		&tasks.CreateFileFromStrTemplateTask{
			SourceStr:  CPFT_DAPP_PLUGIN,
			OutputFile: fmt.Sprintf("%s/plugin/dapp/%s/plugin.go", this.projectPath, this.projName),
			ReplaceKeyPairs: map[string]string{
				types.TagProjectName: this.projName,
				types.TagExecNameFB:  this.execNameFB,
				types.TagProjectPath: goprojpath,
			},
		},
		&tasks.CreateFileFromStrTemplateTask{
			SourceStr:  CPFT_DAPP_EXEC,
			OutputFile: fmt.Sprintf("%s/plugin/dapp/%s/executor/%s.go", this.projectPath, this.projName, this.execName),
			ReplaceKeyPairs: map[string]string{
				types.TagProjectName: this.projName,
				types.TagExecName:    this.execName,
				types.TagClassName:   this.className,
			},
		},
		&tasks.CreateFileFromStrTemplateTask{
			SourceStr:       CPFT_DAPP_CREATEPB,
			OutputFile:      fmt.Sprintf("%s/plugin/dapp/%s/proto/create_protobuf.sh", this.projectPath, this.projName),
			ReplaceKeyPairs: map[string]string{},
		},
		&tasks.CreateFileFromStrTemplateTask{
			SourceStr:       CPFT_DAPP_MAKEFILE,
			OutputFile:      fmt.Sprintf("%s/plugin/dapp/%s/proto/Makefile", this.projectPath, this.projName),
			ReplaceKeyPairs: map[string]string{},
		},
		&tasks.CreateFileFromStrTemplateTask{
			SourceStr:  CPFT_DAPP_PROTO,
			OutputFile: fmt.Sprintf("%s/plugin/dapp/%s/proto/%s.proto", this.projectPath, this.projName, this.execName),
			ReplaceKeyPairs: map[string]string{
				types.TagActionName: this.classActionName,
			},
		},
		&tasks.CreateFileFromStrTemplateTask{
			SourceStr:  CPFT_DAPP_TYPEFILE,
			OutputFile: fmt.Sprintf("%s/plugin/dapp/%s/types/types.go", this.projectPath, this.projName),
			ReplaceKeyPairs: map[string]string{
				types.TagExecNameFB:    this.execNameFB,
				types.TagExecName:      this.execName,
				types.TagClassTypeName: this.classTypeName,
				types.TagActionName:    this.classActionName,
			},
		},
		// 需要将所有的go文件格式化以下
		&tasks.FormatDappSourceTask{
			OutputFolder: fmt.Sprintf("%s/%s/", this.outRootPath, this.projName),
		},
	)
	return taskSlice
}
