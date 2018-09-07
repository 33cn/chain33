package blackwhite

import (
	"gitlab.33.cn/chain33/chain33/plugin/dapp/blackwhite/commands"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/blackwhite/executor"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/blackwhite/rpc"
	"gitlab.33.cn/chain33/chain33/pluginmgr"
)

func init() {
	pluginmgr.Register(&pluginmgr.PluginBase{
		Name:     "chain33.blackwhite",
		ExecName: executor.GetName(),
		Exec:     executor.Init,
		Cmd:      commands.BlackwhiteCmd,
		RPC:      rpc.Init,
	})
}
