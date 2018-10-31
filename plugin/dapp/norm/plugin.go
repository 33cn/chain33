package norm

import (
	"gitlab.33.cn/chain33/chain33/plugin/dapp/norm/executor"
	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/norm/types"
	"gitlab.33.cn/chain33/chain33/pluginmgr"
	"gitlab.33.cn/chain33/chain33/types"
)

func init() {
	pluginmgr.Register(&pluginmgr.PluginBase{
		Name:     "norm",
		ExecName: executor.GetName(),
		Exec:     executor.Init,
		Cmd:      nil,
		RPC:      nil,
	})
	types.RegisterDappFork(ty.NormX, "Enable", 0)
}
