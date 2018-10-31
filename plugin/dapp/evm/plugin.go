package evm

import (
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/commands"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor"
	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/evm/types"
	"gitlab.33.cn/chain33/chain33/pluginmgr"
	"gitlab.33.cn/chain33/chain33/types"
)

func init() {
	pluginmgr.Register(&pluginmgr.PluginBase{
		Name:     "evm",
		ExecName: executor.GetName(),
		Exec:     executor.Init,
		Cmd:      commands.EvmCmd,
		RPC:      nil,
	})
	types.RegisterDappFork(ty.ExecutorName, "ForkEVMState", 650000)
	types.RegisterDappFork(ty.ExecutorName, "ForkEVMKVHash", 1000000)
	types.RegisterDappFork(ty.ExecutorName, "Enable", 500000)
}
