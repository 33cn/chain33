// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"github.com/33cn/chain33/cmd/tools/gencode/base"
	"github.com/33cn/chain33/cmd/tools/types"
)

func init() {

	base.RegisterCodeFile(rpcCodeFile{})
}

type rpcCodeFile struct {
	base.DappCodeFile
}

func (c rpcCodeFile) GetDirName() string {

	return "rpc"
}

func (c rpcCodeFile) GetFiles() map[string]string {

	return map[string]string{
		rpcName:   rpcContent,
		typesName: typesContent,
	}
}

func (c rpcCodeFile) GetFileReplaceTags() []string {

	return []string{types.TagExecName, types.TagImportPath, types.TagClassName}
}

var (
	rpcName    = "rpc.go"
	rpcContent = `package rpc


/* 
 * 实现json rpc和grpc service接口
 * json rpc用Jrpc结构作为接收实例
 * grpc使用channelClient结构作为接收实例
*/

`

	typesName    = "types.go"
	typesContent = `package rpc

import (
	ptypes "${IMPORTPATH}/${EXECNAME}/types/${EXECNAME}"
	rpctypes "github.com/33cn/chain33/rpc/types"
)

/* 
 * rpc相关结构定义和初始化
*/

// 实现grpc的service接口
type channelClient struct {
	rpctypes.ChannelClient
}

// Jrpc 实现json rpc调用实例
type Jrpc struct {
	cli *channelClient
}

// Grpc grpc
type Grpc struct {
	*channelClient
}

// Init init rpc
func Init(name string, s rpctypes.RPCServer) {
	cli := &channelClient{}
	grpc := &Grpc{channelClient: cli}
	cli.Init(name, s, &Jrpc{cli: cli}, grpc)
	//存在grpc service时注册grpc server，需要生成对应的pb.go文件
	ptypes.Register${CLASSNAME}Server(s.GRPC(), grpc)
}`
)
