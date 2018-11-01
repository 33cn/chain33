package privacy

import (
	"gitlab.33.cn/chain33/chain33/plugin/dapp/privacy/commands"
	_ "gitlab.33.cn/chain33/chain33/plugin/dapp/privacy/crypto"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/privacy/executor"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/privacy/rpc"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/privacy/types"
	_ "gitlab.33.cn/chain33/chain33/plugin/dapp/privacy/wallet"
	"gitlab.33.cn/chain33/chain33/pluginmgr"
)

func init() {
	pluginmgr.Register(&pluginmgr.PluginBase{
		Name:     types.PrivacyX,
		ExecName: executor.GetName(),
		Exec:     executor.Init,
		Cmd:      commands.PrivacyCmd,
		RPC:      rpc.Init,
	})
}
