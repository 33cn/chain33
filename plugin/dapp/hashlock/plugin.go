package hashlock

import (
	"gitlab.33.cn/chain33/chain33/plugin/dapp/hashlock/commands"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/hashlock/executor"
	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/hashlock/types"
	"gitlab.33.cn/chain33/chain33/pluginmgr"
	"gitlab.33.cn/chain33/chain33/types"
)

func init() {
	pluginmgr.Register(&pluginmgr.PluginBase{
		Name:     "hashlock",
		ExecName: executor.GetName(),
		Exec:     executor.Init,
		Cmd:      commands.HashlockCmd,
		RPC:      nil,
	})
	types.RegisterDappFork(ty.HashlockX, "Enable", 0)
}
