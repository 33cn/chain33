package tasks

import (
	"github.com/33cn/chain33/util"
)

// CheckFileExistedTask 检测文件是否存在
type CheckFileExistedTask struct {
	TaskBase
	FileName string
}

func (this *CheckFileExistedTask) GetName() string {
	return "CheckFileExistedTask"
}

func (this *CheckFileExistedTask) Execute() error {
	mlog.Info("Execute file existed task.", "file", this.FileName)
	_, err := util.CheckFileExists(this.FileName)
	return err
}
