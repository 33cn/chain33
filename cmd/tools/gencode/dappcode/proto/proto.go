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

type protoFile struct {
	protoBase
}

func (protoFile) GetFiles() map[string]string {
	return map[string]string{
		protoFileName: protoFileContent,
	}
}

func (protoFile) GetReplaceTags() []string {
	return []string{types.TagProtoFileContent, types.TagProtoFileAppend, types.TagExecName}
}

var (
	protoShellName    = "create_protobuf.sh"
	protoShellContent = `#!/bin/sh
# proto生成命令，将pb.go文件生成到types目录下
protoc --go_out=plugins=grpc:../types ./*.proto
`

	makeName    = "Makefile"
	makeContent = `all:
	./create_protobuf.sh
`

	protoFileName    = "${EXECNAME}.proto"
	protoFileContent = `${PROTOFILECONTENT}
${PROTOFILEAPPEND}`
)
