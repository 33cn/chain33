package cert

import (
	"gitlab.33.cn/chain33/chain33/plugin/dapp/cert/executor"
	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/cert/types"
	"gitlab.33.cn/chain33/chain33/pluginmgr"
	"gitlab.33.cn/chain33/chain33/types"
)

func init() {
	pluginmgr.Register(&pluginmgr.PluginBase{
		Name:     "cert",
		ExecName: executor.GetName(),
		Exec:     executor.Init,
		Cmd:      nil,
		RPC:      nil,
	})
	types.RegisterDappFork(ty.CertX, "Enable", 0)
}
