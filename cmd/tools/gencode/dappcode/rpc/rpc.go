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


func (c rpcCodeFile)GetDirName() string {

	return "rpc"
}


func (c rpcCodeFile)GetFiles() map[string]string {

	return map[string]string{
		rpcName: rpcContent,
		typesName: typesContent,
	}
}


func (c rpcCodeFile) GetReplaceTags() []string {

	return []string{types.TagExecName, types.TagClassName}
}


var (

	rpcName = "rpc.go"
	rpcContent =
`package rpc`


	typesName = "types.go"
	typesContent =
`package rpc

import (
	pcode "github.com/33cn/plugin/plugin/dapp/${EXECNAME}/code"
	rpccode "github.com/33cn/chain33/rpc/code"
	"github.com/33cn/chain33/code"
)

type channelClient struct {
	rpccode.ChannelClient
}

type Jrpc struct {
	cli *channelClient
}

type Grpc struct {
	*channelClient
}

func Init(name string, s rpccode.RPCServer) {
	cli := &channelClient{}
	grpc := &Grpc{channelClient: cli}
	cli.Init(name, s, &Jrpc{cli: cli}, grpc)
	//存在grpc service时注册grpc server，需要生成对应的pb.go文件
	pcode.Register${CLASSNAME}Server(s.GRPC(), grpc)
}`
)