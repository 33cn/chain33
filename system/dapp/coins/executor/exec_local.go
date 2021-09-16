// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"github.com/33cn/chain33/common"
	cty "github.com/33cn/chain33/system/dapp/coins/types"
	"github.com/33cn/chain33/types"
)

func (c *Coins) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (dbSet *types.LocalDBSet, err error) {
	dbSet, err = c.execLocal(tx, receipt, index)
	if err != nil || dbSet == nil { // 不能向上层返回LocalDBSet为nil, 以及error
		return &types.LocalDBSet{}, nil
	}
	return dbSet, nil
}

func (c *Coins) execLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {

	action := &cty.CoinsAction{}
	err := types.Decode(tx.Payload, action)
	if err != nil {
		clog.Error("execLocal", "txHash", common.ToHex(tx.Hash()), "decode action err", err)
		return nil, err
	}
	if action.GetTy() == cty.CoinsActionTransfer {
		return c.ExecLocal_Transfer(action.GetTransfer(), tx, receipt, index)
	} else if action.GetTy() == cty.CoinsActionTransferToExec {
		return c.ExecLocal_TransferToExec(action.GetTransferToExec(), tx, receipt, index)
	} else if action.GetTy() == cty.CoinsActionWithdraw {
		return c.ExecLocal_Withdraw(action.GetWithdraw(), tx, receipt, index)
	} else if action.GetTy() == cty.CoinsActionGenesis {
		return c.ExecLocal_Genesis(action.GetGenesis(), tx, receipt, index)
	} else {
		return nil, types.ErrActionNotSupport
	}
}

// ExecLocal_Transfer  transfer of local exec
func (c *Coins) ExecLocal_Transfer(transfer *types.AssetsTransfer, tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	kv, err := updateAddrReciver(c.GetLocalDB(), tx.GetRealToAddr(), transfer.Amount, true)
	if err != nil {
		return nil, err
	}
	return &types.LocalDBSet{KV: []*types.KeyValue{kv}}, nil
}

// ExecLocal_TransferToExec  transfer of local exec to exec
func (c *Coins) ExecLocal_TransferToExec(transfer *types.AssetsTransferToExec, tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	kv, err := updateAddrReciver(c.GetLocalDB(), tx.GetRealToAddr(), transfer.Amount, true)
	if err != nil {
		return nil, err
	}
	return &types.LocalDBSet{KV: []*types.KeyValue{kv}}, nil
}

// ExecLocal_Withdraw  withdraw local exec
func (c *Coins) ExecLocal_Withdraw(withdraw *types.AssetsWithdraw, tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	from := tx.From()
	kv, err := updateAddrReciver(c.GetLocalDB(), from, withdraw.Amount, true)
	if err != nil {
		return nil, err
	}
	return &types.LocalDBSet{KV: []*types.KeyValue{kv}}, nil
}

// ExecLocal_Genesis Genesis of local exec
func (c *Coins) ExecLocal_Genesis(gen *types.AssetsGenesis, tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	kv, err := updateAddrReciver(c.GetLocalDB(), tx.GetRealToAddr(), gen.Amount, true)
	if err != nil {
		return nil, err
	}
	return &types.LocalDBSet{KV: []*types.KeyValue{kv}}, nil
}
