package tasks

import (
	"gitlab.33.cn/chain33/chain33/util"
)

// CheckFileExistedTask 检测文件是否存在
type CheckFileExistedTask struct {
	TaskBase
	FileName string
}

func (this *CheckFileExistedTask) Execute() error {
	mlog.Info("Execute file existed task.", "file", this.FileName)
	_, err := util.CheckFileExists(this.FileName)
	if err != nil {
		return err
	}
	return nil
}
