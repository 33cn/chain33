package privacy

import (
	"gitlab.33.cn/chain33/chain33/plugin/dapp/privacy/executor"
	_ "gitlab.33.cn/chain33/chain33/plugin/dapp/privacy/wallet"
	"gitlab.33.cn/chain33/chain33/pluginmgr"
)

func init() {
	pluginmgr.Register(&pluginmgr.PluginBase{
		Name:     "privacy",
		ExecName: executor.GetName(),
		Exec:     executor.Init,
		Cmd:      nil,
		RPC:      nil,
	})
}
