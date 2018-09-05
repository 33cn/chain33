package pluginmanager

import (
	"gitlab.33.cn/chain33/chain33/pluginmanager/manager"
	"gitlab.33.cn/chain33/chain33/types"
)

func InitExecutor() {
	manager.InitExecutor()
}

func DecodeTx(tx *types.Transaction) interface{} {
	return manager.DecodeTx(tx)
}
