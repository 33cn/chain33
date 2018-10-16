package tasks

// CheckFileExistedTask 检测文件是否存在
type CheckFileExistedTask struct {
	TaskBase
	FileName string
}

func (this *CheckFileExistedTask) Execute() error {
	mlog.Info("Execute file existed task.", "file", this.FileName)

	return nil
}
