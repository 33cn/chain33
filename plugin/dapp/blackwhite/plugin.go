package blackwhite

import (
	"github.com/spf13/cobra"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/blackwhite/commands"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/blackwhite/executor"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/blackwhite/rpc"
	gt "gitlab.33.cn/chain33/chain33/plugin/dapp/blackwhite/types"
	"gitlab.33.cn/chain33/chain33/pluginmgr"
	"gitlab.33.cn/chain33/chain33/types"
)

var gblackwhitePlugin *blackwhitePlugin

func init() {
	types.AllowUserExec = append(types.AllowUserExec, gt.ExecerBlackwhite)
	gblackwhitePlugin = &blackwhitePlugin{}
	pluginmgr.RegisterPlugin(gblackwhitePlugin)
}

type blackwhitePlugin struct {
	pluginmgr.PluginBase
}

func (p *blackwhitePlugin) GetName() string {
	return "chain33.blackwhite"
}

func (p *blackwhitePlugin) GetExecutorName() string {
	return executor.GetName()
}

func (p *blackwhitePlugin) Init() {
	// TODO: 这里应该将初始化的内容统一放在一个初始化的地方
	executor.SetReciptPrefix()
	drivers.Register(executor.GetName(), executor.NewBlackwhite, types.ForkV25BlackWhite)
}

func (p *blackwhitePlugin) AddCmd(rootCmd *cobra.Command) {
	rootCmd.AddCommand(commands.BlackwhiteCmd())
}

func (p *blackwhitePlugin) AddRPC(s pluginmgr.RPCServer) {
	rpc.Init(s)
}
