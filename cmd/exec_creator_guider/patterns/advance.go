package patterns

import (
	"fmt"

	"gitlab.33.cn/chain33/chain33/cmd/exec_creator_guider/tasks"
	"gitlab.33.cn/chain33/chain33/util"
)

type advancePattern struct {
	projName     string // 创建的执行器包名
	execName     string
	clsName      string // 执行器主体类名
	actionName   string // 执行器处理过程中的Action类名
	propFile     string // protobuf 源文件路径
	templateFile string // 生成执行器的模板文件路径
	outputFolder string // 生成执行器的输出目录
	configFolder string // 应用运行的配置目录
}

func (this *advancePattern) Init(configFolder string) {
	this.configFolder = configFolder
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

	this.outputFolder = fmt.Sprintf("output/%s/", projName)
	this.execName, _ = util.MakeStringToUpper(projName, 0, 1)

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
			OutputPath:   this.outputFolder,
			ProjectName:  this.projName,
			ClassName:    this.clsName,
		},
		&tasks.ReplaceTargetTask{
			OutputPath:  this.outputFolder,
			ProjectName: this.projName,
			ClassName:   this.clsName,
			ActionName:  this.actionName,
		},
		&tasks.CreateDappSourceTask{
			TemplatePath:       this.templateFile,
			OutputPath:         this.outputFolder,
			ProjectName:        this.projName,
			ClsName:            this.clsName,
			ActionName:         this.actionName,
			TypeName:           this.clsName + "Type",
			ExecuteName:        this.execName,
			ProtoFile:          this.propFile,
			ExecHeaderTempFile: this.configFolder + "/exec_header.template",
			TypeTempFile:       this.configFolder + "/types_content.template",
			TypeOutputFile:     this.outputFolder + "ptypes/",
		},
		&tasks.FormatDappSourceTask{
			OutputFolder: this.outputFolder,
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
