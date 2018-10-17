package tasks

import "gitlab.33.cn/chain33/chain33/util"

type CreateFolderTask struct {
	TaskBase
	Path string
}

func (this *CreateFolderTask) Execute() error {
	mlog.Info("Execute create folder task.", "path", this.Path)
	return util.MakeDir(this.Path)
}

