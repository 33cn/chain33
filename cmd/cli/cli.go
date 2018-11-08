package main

import (

	// 这一步是必需的，目的时让插件源码有机会进行匿名注册
	"gitlab.33.cn/chain33/chain33/cmd/cli/buildflags"
	_ "gitlab.33.cn/chain33/chain33/system"
	"gitlab.33.cn/chain33/chain33/util/cli"
)

func main() {
	if buildflags.RPCAddr == "" {
		buildflags.RPCAddr = "http://localhost:8801"
	}
	cli.Run(buildflags.RPCAddr, buildflags.ParaName)
}
