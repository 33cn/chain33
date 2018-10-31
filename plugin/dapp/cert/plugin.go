package cert

import (
	"gitlab.33.cn/chain33/chain33/plugin/dapp/cert/executor"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/cert/types"
	"gitlab.33.cn/chain33/chain33/pluginmgr"
)

func init() {
	pluginmgr.Register(&pluginmgr.PluginBase{
		Name:     types.CertX,
		ExecName: executor.GetName(),
		Exec:     executor.Init,
		Cmd:      nil,
		RPC:      nil,
	})
}
