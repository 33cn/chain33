package executor

import (
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	cty "gitlab.33.cn/chain33/chain33/system/dapp/coins/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (c *Coins) Exec_Transfer(transfer *cty.CoinsTransfer, tx *types.Transaction, index int) (*types.Receipt, error) {
	from := tx.From()
	//to 是 execs 合约地址
	if drivers.IsDriverAddress(tx.GetRealToAddr(), c.GetHeight()) {
		return c.GetCoinsAccount().TransferToExec(from, tx.GetRealToAddr(), transfer.Amount)
	}
	return c.GetCoinsAccount().Transfer(from, tx.GetRealToAddr(), transfer.Amount)
}

func (c *Coins) Exec_TransferToExec(transfer *cty.CoinsTransferToExec, tx *types.Transaction, index int) (*types.Receipt, error) {
	if c.GetHeight() < types.ForkV12TransferExec {
		return nil, types.ErrActionNotSupport
	}
	from := tx.From()
	//to 是 execs 合约地址
	if !isExecAddrMatch(transfer.ExecName, tx.GetRealToAddr()) {
		return nil, types.ErrToAddrNotSameToExecAddr
	}
	return c.GetCoinsAccount().TransferToExec(from, tx.GetRealToAddr(), transfer.Amount)
}

func (c *Coins) Exec_Withdraw(withdraw *cty.CoinsWithdraw, tx *types.Transaction, index int) (*types.Receipt, error) {
	if !types.IsMatchFork(c.GetHeight(), types.ForkV16Withdraw) {
		withdraw.ExecName = ""
	}
	from := tx.From()
	//to 是 execs 合约地址
	if drivers.IsDriverAddress(tx.GetRealToAddr(), c.GetHeight()) || isExecAddrMatch(withdraw.ExecName, tx.GetRealToAddr()) {
		return c.GetCoinsAccount().TransferWithdraw(from, tx.GetRealToAddr(), withdraw.Amount)
	}
	return nil, types.ErrActionNotSupport
}

func (c *Coins) Exec_Genesis(genesis *cty.CoinsGenesis, tx *types.Transaction, index int) (*types.Receipt, error) {
	if c.GetHeight() == 0 {
		if drivers.IsDriverAddress(tx.GetRealToAddr(), c.GetHeight()) {
			return c.GetCoinsAccount().GenesisInitExec(genesis.ReturnAddress, genesis.Amount, tx.GetRealToAddr())
		}
		return c.GetCoinsAccount().GenesisInit(tx.GetRealToAddr(), genesis.Amount)
	} else {
		return nil, types.ErrReRunGenesis
	}
}
