package manager

import (
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/types"
)

func InitExecutor() {
	pluginMgr.initExecutor()
}

func InitConsensus() {

}

func InitStore() {

}

func DecodeTx(tx *types.Transaction) interface{} {
	return pluginMgr.decodeTx(tx)
}

func RegisterExecutor(name string, creator drivers.DriverCreate, height int64) bool {
	return pluginMgr.registerExecutor(name, creator, height)
}

func RegisterConsensus() {

}

func RegisterStore() {

}
