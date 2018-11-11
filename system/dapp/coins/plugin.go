package coins

import (
	"github.com/33cn/chain33/pluginmgr"
	"github.com/33cn/chain33/system/dapp/coins/executor"
	"github.com/33cn/chain33/system/dapp/coins/types"
)

func init() {
	pluginmgr.Register(&pluginmgr.PluginBase{
		Name:     types.CoinsX,
		ExecName: executor.GetName(),
		Exec:     executor.Init,
		Cmd:      nil,
		RPC:      nil,
	})
}
