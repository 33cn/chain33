package retrieve

import (
	"gitlab.33.cn/chain33/chain33/plugin/dapp/retrieve/commands"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/retrieve/executor"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/retrieve/rpc"
	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/retrieve/types"
	"gitlab.33.cn/chain33/chain33/pluginmgr"
	"gitlab.33.cn/chain33/chain33/types"
)

func init() {
	pluginmgr.Register(&pluginmgr.PluginBase{
		Name:     "retrieve",
		ExecName: executor.GetName(),
		Exec:     executor.Init,
		Cmd:      commands.RetrieveCmd,
		RPC:      rpc.Init,
	})
	types.RegisterDappFork(ty.RetrieveX, "Enable", 0)
	types.RegisterDappFork(ty.RetrieveX, "ForkRetrive", 180000)
}
