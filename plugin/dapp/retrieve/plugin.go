package retrieve

import (
	"gitlab.33.cn/chain33/chain33/plugin/dapp/retrieve/commands"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/retrieve/executor"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/retrieve/rpc"
	"gitlab.33.cn/chain33/chain33/pluginmgr"
)

func init() {
	pluginmgr.Register(&pluginmgr.PluginBase{
		Name:     "retrieve",
		ExecName: executor.GetName(),
		Exec:     executor.Init,
		Cmd:      commands.RetrieveCmd,
		RPC:      rpc.Init,
	})
}
