package game

import (
	"gitlab.33.cn/chain33/chain33/plugin/dapp/game/commands"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/game/executor"
	gt "gitlab.33.cn/chain33/chain33/plugin/dapp/game/types"
	"gitlab.33.cn/chain33/chain33/pluginmgr"
	"gitlab.33.cn/chain33/chain33/types"
)

func init() {
	pluginmgr.Register(&pluginmgr.PluginBase{
		Name:     gt.GameX,
		ExecName: executor.GetName(),
		Exec:     executor.Init,
		Cmd:      commands.Cmd,
		RPC:      nil,
	})
	types.RegisterDappFork(gt.GameX, "Enable", 0)
}
