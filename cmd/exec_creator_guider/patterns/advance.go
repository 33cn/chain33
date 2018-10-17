package patterns

import (
	"fmt"

	"gitlab.33.cn/chain33/chain33/cmd/exec_creator_guider/tasks"
)

type advancePattern struct {
	projName     string
	clsName      string
	actionName   string
	propFile     string
	templateFile string
	outputFolder string
}

func (this *advancePattern) Run(projName, clsName, actionName, propFile, templateFile string) {
	mlog.Info("Advance project create guid start.")
	defer mlog.Info("Advance project create guid finish.")

	if len(projName) == 0 || len(clsName) == 0 || len(actionName) == 0 ||
		len(propFile) == 0 || len(templateFile) == 0 {
		mlog.Error("Run param invalid.",
			"projName", projName,
			"clsName", clsName,
			"actionName", actionName,
			"propFile", propFile,
			"templateFile", templateFile)
		return
	}
	this.projName = projName
	this.clsName = clsName
	this.actionName = actionName
	this.propFile = propFile
	this.templateFile = templateFile

	this.outputFolder = fmt.Sprintf("output/%s", projName)

	var err error
	task := this.buildTask()
	for {
		if task == nil {
			break
		}
		err = task.Execute()
		if err != nil {
			break
		}
		task = task.Next()
	}
	if err != nil {
		mlog.Error("Execute command failed.", "error", err)
	}
}

func (this *advancePattern) buildTask() tasks.Task {
	taskSlice := make([]tasks.Task, 0)
	taskSlice = append(taskSlice,
		// 检查用户编写的protobuf文件是否存在
		&tasks.CheckFileExistedTask{
			FileName: this.propFile,
		},
		// 将文件复制到输出目录下
		&tasks.CopyTemplateToOutputTask{
			TemplatePath: this.templateFile,
			OutputPath: this.outputFolder,
			ProjectName:this.projName,
			ClassName:this.clsName,
		},
		&tasks.ReplaceTargetTask{
			OutputPath: this.outputFolder,
			ProjectName:this.projName,
			ClassName: this.clsName,
			ActionName:this.actionName,
		},
		// 结合系统的模板文件和生成的xxx.pb.go，生成项目生成脚本代码
		&tasks.CreateBuildAppSourceTask{
			TemplatePath: this.templateFile,
			ClsName:      this.clsName,
			ActionName:   this.actionName,
			ProtoFile:    this.propFile,
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
