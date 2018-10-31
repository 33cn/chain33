package trade

import (
	"gitlab.33.cn/chain33/chain33/plugin/dapp/trade/commands"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/trade/executor"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/trade/rpc"
	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/trade/types"
	"gitlab.33.cn/chain33/chain33/pluginmgr"
	"gitlab.33.cn/chain33/chain33/types"
)

func init() {
	pluginmgr.Register(&pluginmgr.PluginBase{
		Name:     "trade",
		ExecName: executor.GetName(),
		Exec:     executor.Init,
		Cmd:      commands.TradeCmd,
		RPC:      rpc.Init,
	})
	types.RegisterDappFork(ty.TradeX, "Enable", 100899)
	types.RegisterDappFork(ty.TradeX, "ForkTradeBuyLimit", 301000)
	types.RegisterDappFork(ty.TradeX, "ForkTradeAsset", 1010000)
}
