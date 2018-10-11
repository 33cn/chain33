package token

import (
	"gitlab.33.cn/chain33/chain33/plugin/dapp/token/commands"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/token/executor"
	"gitlab.33.cn/chain33/chain33/pluginmgr"
)

func init() {
	pluginmgr.Register(&pluginmgr.PluginBase{
		Name:     "token",
		ExecName: executor.GetName(),
		Exec:     executor.Init,
		Cmd:      commands.TokenCmd,
		RPC:      nil,
	})
}
