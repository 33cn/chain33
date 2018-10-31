package ticket

import (
	"gitlab.33.cn/chain33/chain33/plugin/dapp/ticket/commands"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/ticket/executor"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/ticket/rpc"
	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/ticket/types"
	_ "gitlab.33.cn/chain33/chain33/plugin/dapp/ticket/wallet"
	"gitlab.33.cn/chain33/chain33/pluginmgr"
	"gitlab.33.cn/chain33/chain33/types"
)

func init() {
	pluginmgr.Register(&pluginmgr.PluginBase{
		Name:     "ticket",
		ExecName: executor.GetName(),
		Exec:     executor.Init,
		Cmd:      commands.TicketCmd,
		RPC:      rpc.Init,
	})
	types.RegisterDappFork(ty.TicketX, "Enable", 0)
}
