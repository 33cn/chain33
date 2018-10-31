package hashlock

import (
	"gitlab.33.cn/chain33/chain33/plugin/dapp/hashlock/commands"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/hashlock/executor"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/hashlock/types"
	"gitlab.33.cn/chain33/chain33/pluginmgr"
)

func init() {
	pluginmgr.Register(&pluginmgr.PluginBase{
		Name:     types.HashlockX,
		ExecName: executor.GetName(),
		Exec:     executor.Init,
		Cmd:      commands.HashlockCmd,
		RPC:      nil,
	})
}
