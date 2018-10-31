package privacy

import (
	"gitlab.33.cn/chain33/chain33/plugin/dapp/privacy/commands"
	_ "gitlab.33.cn/chain33/chain33/plugin/dapp/privacy/crypto"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/privacy/executor"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/privacy/rpc"
	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/privacy/types"
	_ "gitlab.33.cn/chain33/chain33/plugin/dapp/privacy/wallet"
	"gitlab.33.cn/chain33/chain33/pluginmgr"
	"gitlab.33.cn/chain33/chain33/types"
)

func init() {
	pluginmgr.Register(&pluginmgr.PluginBase{
		Name:     "privacy",
		ExecName: executor.GetName(),
		Exec:     executor.Init,
		Cmd:      commands.PrivacyCmd,
		RPC:      rpc.Init,
	})
	types.RegisterDappFork(ty.PrivacyX, "Enable", 980000)
}
