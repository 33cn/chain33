package relay

import (
	"gitlab.33.cn/chain33/chain33/plugin/dapp/relay/commands"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/relay/executor"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/relay/rpc"
	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/relay/types"
	"gitlab.33.cn/chain33/chain33/pluginmgr"
	"gitlab.33.cn/chain33/chain33/types"
)

func init() {
	pluginmgr.Register(&pluginmgr.PluginBase{
		Name:     "relay",
		ExecName: executor.GetName(),
		Exec:     executor.Init,
		Cmd:      commands.RelayCmd,
		RPC:      rpc.Init,
	})
	types.RegisterDappFork(ty.RelayX, "Enable", 570000)
}
