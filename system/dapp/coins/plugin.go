package coins

import (
	"gitlab.33.cn/chain33/chain33/pluginmgr"
	"gitlab.33.cn/chain33/chain33/system/dapp/coins/executor"
)

func init() {
	pluginmgr.Register(&pluginmgr.PluginBase{
		Name:     "coins",
		ExecName: executor.GetName(),
		Exec:     executor.Init,
		Cmd:      nil,
		RPC:      nil,
	})
}
