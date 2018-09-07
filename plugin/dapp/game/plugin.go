package game

import (
	"gitlab.33.cn/chain33/chain33/plugin/dapp/game/commands"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/game/executor"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/game/rpc"
	gt "gitlab.33.cn/chain33/chain33/plugin/dapp/game/types"
	"gitlab.33.cn/chain33/chain33/pluginmgr"
)

func init() {
	pluginmgr.RegisterPlugin(&pluginmgr.PluginBase{
		Name:     gt.PackageName,
		ExecName: executor.GetName(),
		Exec:     executor.Init,
		Cmd:      commands.Cmd,
		RPC:      rpc.Init,
	})
}
