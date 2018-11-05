package strategy

import (
	"fmt"

	"github.com/pkg/errors"
	"gitlab.33.cn/chain33/chain33/cmd/tools/tasks"
	"gitlab.33.cn/chain33/chain33/cmd/tools/types"
	"gitlab.33.cn/chain33/chain33/util"
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

func (this *advanceCreateExecProjStrategy) Run() error {
	fmt.Println("Begin run chain33 create executor project advance mode.")
	defer fmt.Println("Run chain33 create executor project advance modefinish.")
	this.initMember()
	if !this.checkParamValid() {
		return errors.New("InvalidParams")
	}
	return this.runImpl()
}

func (this *advanceCreateExecProjStrategy) checkParamValid() bool {
	return true
}

func (this *advanceCreateExecProjStrategy) initMember() {
	if v, err := this.getParam(types.KeyConfigFolder); err == nil {
		this.configFolder = v
	}
	if v, err := this.getParam(types.KeyProjectName); err == nil {
		this.projName = v
	}
	if v, err := this.getParam(types.KeyClassName); err == nil {
		this.clsName = v
	}
	if v, err := this.getParam(types.KeyExecutorName); err == nil {
		this.execName = v
	}
	if v, err := this.getParam(types.KeyActionName); err == nil {
		this.actionName, _ = util.MakeStringToUpper(v, 0, 1)
	}
	if v, err := this.getParam(types.KeyProtobufFile); err == nil {
		this.propFile = v
	}
	if v, err := this.getParam(types.KeyTemplateFilePath); err == nil {
		this.templateFile = v
	}
	this.outputFolder = fmt.Sprintf("output/%s/", this.projName)
	util.MakeDir(this.outputFolder)
}

func (this *advanceCreateExecProjStrategy) runImpl() error {
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

func (this *advanceCreateExecProjStrategy) buildTask() tasks.Task {
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
			ExecName:    this.execName,
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
