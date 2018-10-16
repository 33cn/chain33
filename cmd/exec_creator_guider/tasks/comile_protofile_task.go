package tasks

// CompileProtoFileTask 将用户编写的proto文件通过工具编译成pb.go
type CompileProtoFileTask struct {
	TaskBase
	FileName string
}

func (this *CompileProtoFileTask) Execute() error {
	return nil
}
