package manage

import (
	"gitlab.33.cn/chain33/chain33/pluginmgr"
	"gitlab.33.cn/chain33/chain33/system/dapp/manage/commands"
	"gitlab.33.cn/chain33/chain33/system/dapp/manage/executor"
)

func init() {
	pluginmgr.Register(&pluginmgr.PluginBase{
		Name:     "manage",
		ExecName: executor.GetName(),
		Exec:     executor.Init,
		Cmd:      commands.ConfigCmd,
		RPC:      nil,
	})
}
