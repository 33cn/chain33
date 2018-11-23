// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package strategy

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/33cn/chain33/cmd/tools/tasks"
	"github.com/33cn/chain33/cmd/tools/types"
	"github.com/33cn/chain33/util"
	"github.com/pkg/errors"
)

type advanceCreateExecProjStrategy struct {
	strategyBasic

	projName     string // 创建的执行器包名
	execName     string
	clsName      string // 执行器主体类名
	actionName   string // 执行器处理过程中的Action类名
	propFile     string // protobuf 源文件路径
	templateFile string // 生成执行器的模板文件路径
	outputFolder string // 生成执行器的输出目录
	configFolder string // 应用运行的配置目录
}

func (ad *advanceCreateExecProjStrategy) Run() error {
	fmt.Println("Begin run chain33 create executor project advance mode.")
	defer fmt.Println("Run chain33 create executor project advance mode finish.")
	ad.initMember()
	if !ad.checkParamValid() {
		return errors.New("InvalidParams")
	}
	return ad.runImpl()
}

func (ad *advanceCreateExecProjStrategy) checkParamValid() bool {
	return true
}

func (ad *advanceCreateExecProjStrategy) initMember() {
	if v, err := ad.getParam(types.KeyConfigFolder); err == nil {
		ad.configFolder = v
	}
	if v, err := ad.getParam(types.KeyProjectName); err == nil {
		ad.projName = v
	}
	if v, err := ad.getParam(types.KeyClassName); err == nil {
		ad.clsName = v
	}
	if v, err := ad.getParam(types.KeyExecutorName); err == nil {
		ad.execName = v
	}
	if v, err := ad.getParam(types.KeyActionName); err == nil {
		ad.actionName, _ = util.MakeStringToUpper(v, 0, 1)
	}
	if v, err := ad.getParam(types.KeyProtobufFile); err == nil {
		ad.propFile = v
	}
	if v, err := ad.getParam(types.KeyTemplateFilePath); err == nil {
		ad.templateFile = v
	}
	// 默认输出到chain33项目的plugin/dapp/目录下
	var outputPath string
	gopath := os.Getenv("GOPATH")
	if len(gopath) > 0 {
		outputPath = filepath.Join(gopath, "/src/github.com/33cn/chain33/plugin/dapp/")
	}
	if len(outputPath) > 0 && util.CheckPathExisted(outputPath) {
		ad.outputFolder = fmt.Sprintf("%s/%s/", outputPath, ad.projName)
	} else {
		// 默认就在当前目录下
		ad.outputFolder = fmt.Sprintf("output/%s/", ad.projName)
	}
	util.MakeDir(ad.outputFolder)
}

func (ad *advanceCreateExecProjStrategy) runImpl() error {
	var err error
	task := ad.buildTask()
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

func (ad *advanceCreateExecProjStrategy) buildTask() tasks.Task {
	taskSlice := make([]tasks.Task, 0)
	taskSlice = append(taskSlice,
		// 检查用户编写的protobuf文件是否存在
		&tasks.CheckFileExistedTask{
			FileName: ad.propFile,
		},
		// 将文件复制到输出目录下
		&tasks.CopyTemplateToOutputTask{
			TemplatePath: ad.templateFile,
			OutputPath:   ad.outputFolder,
			ProjectName:  ad.projName,
			ClassName:    ad.clsName,
		},
		&tasks.ReplaceTargetTask{
			OutputPath:  ad.outputFolder,
			ProjectName: ad.projName,
			ClassName:   ad.clsName,
			ActionName:  ad.actionName,
			ExecName:    ad.execName,
		},
		&tasks.CreateDappSourceTask{
			TemplatePath:       ad.templateFile,
			OutputPath:         ad.outputFolder,
			ProjectName:        ad.projName,
			ClsName:            ad.clsName,
			ActionName:         ad.actionName,
			TypeName:           ad.clsName + "Type",
			ExecuteName:        ad.execName,
			ProtoFile:          ad.propFile,
			ExecHeaderTempFile: ad.configFolder + "/exec_header.template",
			TypeTempFile:       ad.configFolder + "/types_content.template",
			TypeOutputFile:     ad.outputFolder + "ptypes/",
		},
		&tasks.FormatDappSourceTask{
			OutputFolder: ad.outputFolder,
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
