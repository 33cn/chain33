package types

import "gitlab.33.cn/chain33/chain33/types"

const (
	CoinsX                    = "coins"
	CoinsActionTransfer       = 1
	CoinsActionGenesis        = 2
	CoinsActionWithdraw       = 3
	CoinsActionTransferToExec = 10
)

var ExecerCoins = []byte(CoinsX)

func init() {
	types.AllowUserExec = append(types.AllowUserExec, ExecerCoins)
}
