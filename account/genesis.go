package account

import (
	"gitlab.33.cn/chain33/chain33/types"
)

func (acc *DB) GenesisInit(addr string, amount int64) (*types.Receipt, error) {
	accTo := acc.LoadAccount(addr)
	copyto := *accTo
	accTo.Balance = accTo.GetBalance() + amount
	receiptBalanceTo := &types.ReceiptAccountTransfer{
		Prev:    &copyto,
		Current: accTo,
	}
	acc.SaveAccount(accTo)
	receipt := acc.genesisReceipt(accTo, receiptBalanceTo)
	return receipt, nil
}

func (acc *DB) GenesisInitExec(addr string, amount int64, execaddr string) (*types.Receipt, error) {
	accTo := acc.LoadAccount(execaddr)
	copyto := *accTo
	accTo.Balance = accTo.GetBalance() + amount
	receiptBalanceTo := &types.ReceiptAccountTransfer{
		Prev:    &copyto,
		Current: accTo,
	}
	acc.SaveAccount(accTo)
	receipt := acc.genesisReceipt(accTo, receiptBalanceTo)
	receipt2, err := acc.execDeposit(addr, execaddr, amount)
	if err != nil {
		panic(err)
	}
	ty := int32(types.TyLogGenesisDeposit)
	if acc.IsTokenAccount() {
		ty = int32(types.TyLogTokenGenesisDeposit)
	}
	receipt2.Ty = ty
	receipt = acc.mergeReceipt(receipt, receipt2)
	return receipt, nil
}

func (acc *DB) genesisReceipt(accTo *types.Account, receiptTo *types.ReceiptAccountTransfer) *types.Receipt {
	ty := int32(types.TyLogGenesisTransfer)
	if acc.IsTokenAccount() {
		ty = int32(types.TyLogTokenGenesisTransfer)
	}
	log2 := &types.ReceiptLog{
		Ty:  ty,
		Log: types.Encode(receiptTo),
	}
	kv := acc.GetKVSet(accTo)
	return &types.Receipt{
		Ty:   types.ExecOk,
		KV:   kv,
		Logs: []*types.ReceiptLog{log2},
	}
}
