// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"github.com/33cn/chain33/common/address"
	drivers "github.com/33cn/chain33/system/dapp"
	"github.com/33cn/chain33/types"
)

// Exec_Transfer transfer of exec
func (c *Coins) Exec_Transfer(transfer *types.AssetsTransfer, tx *types.Transaction, index int) (*types.Receipt, error) {
	from := tx.From()
	//to 是 execs 合约地址
	if drivers.IsDriverAddress(tx.GetRealToAddr(), c.GetHeight()) {
		return c.GetCoinsAccount().TransferToExec(from, tx.GetRealToAddr(), transfer.Amount)
	}
	return c.GetCoinsAccount().Transfer(from, tx.GetRealToAddr(), transfer.Amount)
}

// Exec_TransferToExec the transfer to exec address
func (c *Coins) Exec_TransferToExec(transfer *types.AssetsTransferToExec, tx *types.Transaction, index int) (*types.Receipt, error) {
	if !types.IsFork(c.GetHeight(), "ForkTransferExec") {
		return nil, types.ErrActionNotSupport
	}
	from := tx.From()
	//to 是 execs 合约地址
	if !isExecAddrMatch(transfer.ExecName, tx.GetRealToAddr()) {
		return nil, types.ErrToAddrNotSameToExecAddr
	}
	return c.GetCoinsAccount().TransferToExec(from, tx.GetRealToAddr(), transfer.Amount)
}

// Exec_Withdraw withdraw exec
func (c *Coins) Exec_Withdraw(withdraw *types.AssetsWithdraw, tx *types.Transaction, index int) (*types.Receipt, error) {
	if !types.IsFork(c.GetHeight(), "ForkWithdraw") {
		withdraw.ExecName = ""
	}
	from := tx.From()
	//to 是 execs 合约地址
	if drivers.IsDriverAddress(tx.GetRealToAddr(), c.GetHeight()) || isExecAddrMatch(withdraw.ExecName, tx.GetRealToAddr()) {
		return c.GetCoinsAccount().TransferWithdraw(from, tx.GetRealToAddr(), withdraw.Amount)
	}
	return nil, types.ErrActionNotSupport
}

// Exec_Genesis genesis of exec
func (c *Coins) Exec_Genesis(genesis *types.AssetsGenesis, tx *types.Transaction, index int) (*types.Receipt, error) {
	if c.GetHeight() == 0 {
		if drivers.IsDriverAddress(tx.GetRealToAddr(), c.GetHeight()) {
			return c.GetCoinsAccount().GenesisInitExec(genesis.ReturnAddress, genesis.Amount, tx.GetRealToAddr())
		}
		return c.GetCoinsAccount().GenesisInit(tx.GetRealToAddr(), genesis.Amount)
	}
	return nil, types.ErrReRunGenesis
}

func isExecAddrMatch(name string, to string) bool {
	toaddr := address.ExecAddress(name)
	return toaddr == to
}
