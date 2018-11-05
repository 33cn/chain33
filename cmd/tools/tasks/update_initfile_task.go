package tasks

import (
	"fmt"
	"github.com/pkg/errors"
	"gitlab.33.cn/chain33/chain33/util"
)

type UpdateInitFileTask struct {
	TaskBase
	Folder string
}

func (this *UpdateInitFileTask) GetName() string {
	return "UpdateInitFileTask"
}

func (this *UpdateInitFileTask) Execute() error {
	// 1. 检查目标文件夹是否存在，如果不存在则不扫描
	if !util.CheckPathExisted(this.Folder) {
		return errors.New(fmt.Sprintf("%s Not Existed.", this.Folder))
	}
	// 2. 扫描目标文件夹内一级文件夹名称，记录到一个数组中
	// 3. 在文件夹内新增init文件夹
	// 4. 在init文件夹内新增init.go文件
	// 5. 在init.go中，按照格式将数组的内容填充到文件内
	// 6. 格式化init.go文件
	return nil
}