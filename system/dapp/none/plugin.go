package none

import (
	"github.com/33cn/chain33/pluginmgr"
	"github.com/33cn/chain33/system/dapp/none/executor"
)

func init() {
	pluginmgr.Register(&pluginmgr.PluginBase{
		Name:     "none",
		ExecName: executor.GetName(),
		Exec:     executor.Init,
		Cmd:      nil,
		RPC:      nil,
	})
}
