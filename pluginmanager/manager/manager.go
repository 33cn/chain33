package manager

import (
	"gitlab.33.cn/chain33/chain33/pluginmanager/plugin"
	"gitlab.33.cn/chain33/chain33/types"
)

func InitExecutor() {
	pluginMgr.initExecutor()
}

func DecodeTx(tx *types.Transaction) interface{} {
	return pluginMgr.decodeTx(tx)
}

func RegisterPlugin(p plugin.Plugin) bool {
	return pluginMgr.registerPlugin(p)
}
