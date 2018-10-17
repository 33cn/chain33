package tasks

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

// CompileProtoFileTask 将用户编写的proto文件通过工具编译成pb.go
type CompileProtoFileTask struct {
	TaskBase
	FileName string
}

func (this *CompileProtoFileTask) Execute() error {
	mlog.Info("Execute compile protobuf file task.", "file", this.FileName)
	cmd := exec.Command("protoc", "--proto_path=.", "--go_out=.", this.FileName)
	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	if errBuf.String() != "" {
		mlog.Error("CompileProtoFileTask Command error", "error", errBuf.String())
	} else {
		mlog.Info("CompileProtoFileTask success")
	}
	return err
}
