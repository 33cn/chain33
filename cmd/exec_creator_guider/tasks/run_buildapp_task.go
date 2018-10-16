package tasks

// RunBuildAppTask 过go命令将其编译出来运行
type RunBuildAppTask struct {
	TaskBase
	FileName string
}

func (this *RunBuildAppTask) Execute() error {
	mlog.Info("Execute run build app task.")
	return nil
}
