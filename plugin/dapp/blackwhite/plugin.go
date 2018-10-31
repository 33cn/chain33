package blackwhite

import (
	"gitlab.33.cn/chain33/chain33/plugin/dapp/blackwhite/commands"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/blackwhite/executor"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/blackwhite/rpc"
	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/blackwhite/types"
	"gitlab.33.cn/chain33/chain33/pluginmgr"
	"gitlab.33.cn/chain33/chain33/types"
)

func init() {
	pluginmgr.Register(&pluginmgr.PluginBase{
		Name:     "blackwhite",
		ExecName: executor.GetName(),
		Exec:     executor.Init,
		Cmd:      commands.BlackwhiteCmd,
		RPC:      rpc.Init,
	})
	types.RegisterDappFork(ty.BlackwhiteX, "ForkBlackWhiteV2", 900000)
	types.RegisterDappFork(ty.BlackwhiteX, "Enable", 850000)
}
