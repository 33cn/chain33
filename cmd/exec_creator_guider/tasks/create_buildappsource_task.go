package tasks

// CreateBuildAppSourceTask 通过生成好的pb.go和预先设计的模板，生成反射程序源码
type CreateBuildAppSourceTask struct {
	TaskBase
	FileName string
}

func (this *CreateBuildAppSourceTask) Execute() error {
	return nil
}
