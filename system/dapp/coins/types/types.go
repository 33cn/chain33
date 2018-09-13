package types

const (
	CoinsX                    = "coins"
	CoinsActionTransfer       = 1
	CoinsActionGenesis        = 2
	CoinsActionWithdraw       = 3
	CoinsActionTransferToExec = 10
)

var ExecerCoins = []byte(CoinsX)
