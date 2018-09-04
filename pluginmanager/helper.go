package pluginmanager

import (
	"gitlab.33.cn/chain33/chain33/pluginmanager/manager"
	"gitlab.33.cn/chain33/chain33/types"
)

func InitExecutor() {
	manager.InitExecutor()
}

func InitConsensus() {
	manager.InitConsensus()
}

func InitStore() {
	manager.InitStore()
}

func DecodeTx(tx *types.Transaction) interface{} {
	return manager.DecodeTx(tx)

	//else if types.GameX == realExec(string(tx.Execer)) {
	//	var action types.GameAction
	//	err := types.Decode(tx.GetPayload(), &action)
	//	if err != nil {
	//		unkownPl["unkownpayload"] = string(tx.GetPayload())
	//		pl = unkownPl
	//	} else {
	//		pl = &action
	//	}
	//}

	return nil
}