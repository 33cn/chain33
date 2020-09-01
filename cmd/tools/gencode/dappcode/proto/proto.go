// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package proto

import (
	"github.com/33cn/chain33/cmd/tools/gencode/base"
	"github.com/33cn/chain33/cmd/tools/types"
)

func init() {

	base.RegisterCodeFile(protoBase{})
	base.RegisterCodeFile(protoFile{})
}

type protoBase struct {
	base.DappCodeFile
}

func (protoBase) GetDirName() string {

	return "proto"
}

func (protoBase) GetFiles() map[string]string {

	return map[string]string{
		protoShellName: protoShellContent,
		makeName:       makeContent,
	}
}

func (protoBase) GetFileReplaceTags() []string {
	return []string{types.TagExecName}
}

type protoFile struct {
	protoBase
}

func (protoFile) GetFiles() map[string]string {
	return map[string]string{
		protoFileName: protoFileContent,
	}
}

func (protoFile) GetFileReplaceTags() []string {
	return []string{types.TagProtoFileContent, types.TagProtoFileAppend, types.TagExecName}
}

var (
	protoShellName    = "create_protobuf.sh"
	protoShellContent = `#!/bin/bash
# proto生成命令，将pb.go文件生成到types/目录下, chain33_path支持引用chain33框架的proto文件
chain33_path=$(go list -f '{{.Dir}}' "github.com/33cn/chain33")
protoc --go_out=plugins=grpc:../types ./*.proto --proto_path=. --proto_path="${chain33_path}/types/proto/"
`

	makeName    = "Makefile"
	makeContent = `all:
	bash ./create_protobuf.sh
`

	protoFileName    = "${EXECNAME}.proto"
	protoFileContent = `${PROTOFILECONTENT}
${PROTOFILEAPPEND}`
)
