package executor

import (
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/types"
)

func ParaAssetTransfer(acc *account.DB, addr string, amount int64, execaddr string) (*types.Receipt, error) {
	receipt2, err := acc.ExecDeposit(addr, execaddr, amount)
	return receipt2, err
}

func ParaAssetWithdraw(acc *account.DB, addr string, amount int64, execaddr string) (*types.Receipt, error) {
	receipt, err := acc.ExecWithdraw(execaddr, addr, amount)
	return receipt, err
}
