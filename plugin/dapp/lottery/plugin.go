package lottery

import (
	"gitlab.33.cn/chain33/chain33/plugin/dapp/lottery/executor"
	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/lottery/types"
	"gitlab.33.cn/chain33/chain33/pluginmgr"
	"gitlab.33.cn/chain33/chain33/types"
)

func init() {
	pluginmgr.Register(&pluginmgr.PluginBase{
		Name:     "lottery",
		ExecName: executor.GetName(),
		Exec:     executor.Init,
		Cmd:      nil,
		RPC:      nil,
	})
	types.RegisterDappFork(ty.LotteryX, "Enable", 0)
}
