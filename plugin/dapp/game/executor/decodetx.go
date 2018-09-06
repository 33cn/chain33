package executor

import (
	gt "gitlab.33.cn/chain33/chain33/plugin/dapp/game/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func DecodeTx(tx *types.Transaction) interface{} {
	var pl interface{}
	action := &gt.GameAction{}
	err := types.Decode(tx.GetPayload(), action)
	if err != nil {
		pl = string(tx.GetPayload())
	} else {
		pl = action
	}
	return pl
}
