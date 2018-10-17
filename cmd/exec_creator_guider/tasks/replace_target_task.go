package tasks

// ReplaceTargetTask 替换指定目录下所有文件的标志性文字
// 可替换的名字列表如下：
// ${PROJECTNAME}
// ${CLASSNAME}
// ${ACTIONNAME}
type ReplaceTargetTask struct {
	TaskBase
	OutputPath  string
	ProjectName string
	ClassName   string
	ActionName  string
}

// Execute 执行具体的替换动作
// 1. 扫描指定的output路径
// 2. 打开每一个文件，根据替换规则替换内部的所有标签
// 3. 保存时查看文件名是否要替换，如果要则替换后保存，否则直接保存
// 4. 一直到所有的文件都替换完毕
func (this *ReplaceTargetTask) Execute() error {
	mlog.Info("Execute replace target task.")
	return nil
}
