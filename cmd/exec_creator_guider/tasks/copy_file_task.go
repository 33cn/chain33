package tasks

// CopyFileTask 检测文件是否存在
type CopyFileTask struct {
	TaskBase
	FileName string
}

func (this *CopyFileTask) Execute() error {
	mlog.Info("Execute copy file task.", "origin file", this.FileName, "target file", this.FileName)

	return nil
}
