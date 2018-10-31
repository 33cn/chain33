package valnode

import (
	"gitlab.33.cn/chain33/chain33/plugin/dapp/valnode/executor"
	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/valnode/types"
	"gitlab.33.cn/chain33/chain33/pluginmgr"
	"gitlab.33.cn/chain33/chain33/types"
)

func init() {
	pluginmgr.Register(&pluginmgr.PluginBase{
		Name:     "valnode",
		ExecName: executor.GetName(),
		Exec:     executor.Init,
		Cmd:      nil,
		RPC:      nil,
	})
	types.RegisterDappFork(ty.ValNodeX, "Enable", 0)
}
