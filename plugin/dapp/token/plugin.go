package token

import (
	"gitlab.33.cn/chain33/chain33/plugin/dapp/token/commands"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/token/executor"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/token/rpc"
	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/token/types"
	"gitlab.33.cn/chain33/chain33/pluginmgr"
	"gitlab.33.cn/chain33/chain33/types"
)

func init() {
	pluginmgr.Register(&pluginmgr.PluginBase{
		Name:     "token",
		ExecName: executor.GetName(),
		Exec:     executor.Init,
		Cmd:      commands.TokenCmd,
		RPC:      rpc.Init,
	})
	types.RegisterDappFork(ty.TokenX, "Enable", 100899)
	types.RegisterDappFork(ty.TokenX, "ForkTokenBlackList", 190000)
	types.RegisterDappFork(ty.TokenX, "ForkBadTokenSymbol", 184000)
	types.RegisterDappFork(ty.TokenX, "ForkTokenPrice", 560000)
}
