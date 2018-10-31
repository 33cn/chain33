package evm

import (
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/commands"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/types"
	"gitlab.33.cn/chain33/chain33/pluginmgr"
)

func init() {
	pluginmgr.Register(&pluginmgr.PluginBase{
		Name:     types.ExecutorName,
		ExecName: executor.GetName(),
		Exec:     executor.Init,
		Cmd:      commands.EvmCmd,
		RPC:      nil,
	})
}
