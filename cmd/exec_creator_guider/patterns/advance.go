package patterns

import "gitlab.33.cn/chain33/chain33/cmd/exec_creator_guider/tasks"

type advancePattern struct {
	projName     string
	clsName      string
	actionName   string
	propFile     string
	templateFile string
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
		// 检查系统配置的模板文件是否存在
		&tasks.CheckFileExistedTask{
			FileName: this.templateFile,
		},
		// 使用protoc编译用户编写的protobuf文件，生成 xxx.pb.go
		&tasks.CompileProtoFileTask{
			FileName: this.propFile,
		},
		// 结合系统的模板文件和生成的xxx.pb.go，生成项目生成脚本代码
		&tasks.CreateBuildAppSourceTask{
			FileName: this.propFile,
		},
		// 编译并运行
		&tasks.RunBuildAppTask{
			FileName: this.propFile,
		},
	)

	task := taskSlice[0]
	sliceLen := len(taskSlice)
	for n := 1; n < sliceLen; n++ {
		task.SetNext(taskSlice[n])
		task = taskSlice[n]
	}
	return task
}
