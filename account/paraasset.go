package account

import (
	"github.com/golang/protobuf/proto"
	"gitlab.33.cn/chain33/chain33/types"
)

/*
func (acc *DB) ParaAssetTransfer(addr string, amount int64, execaddr string) (*types.Receipt, error) {
	accTo := acc.LoadAccount(execaddr)
	copyto := *accTo
	accTo.Balance = accTo.GetBalance() + amount
	receiptBalanceTo := &types.ReceiptAccountTransfer{
		Prev:    &copyto,
		Current: accTo,
	}
	acc.SaveAccount(accTo)
	receipt := acc.ParaAssetTransferReceipt(accTo, receiptBalanceTo)

	receipt2, err := acc.ExecDeposit(addr, execaddr, amount)
	if err != nil {
		panic(err)
	}

	receipt = acc.mergeReceipt(receipt, receipt2)
	return receipt, nil
}
*/
func (acc *DB) ParaAssetTransfer(addr string, amount int64, execaddr string) (*types.Receipt, error) {
	receipt2, err := acc.ExecDeposit(addr, execaddr, amount)
	return receipt2, err
}

func (acc *DB) ParaAssetTransferReceipt(accTo *types.Account, receiptTo proto.Message) *types.Receipt {
	ty := int32(types.TyLogParaAssetTransfer)
	if acc.execer == types.ExecName(types.TokenX) {
		ty = int32(types.TyLogParaTokenAssetTransfer)
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

/*
func (acc *DB) ParaAssetWithdraw(addr string, amount int64, execaddr string) (*types.Receipt, error) {
	receipt, err := acc.ExecWithdraw(execaddr, addr, amount)
	if err != nil {
		return receipt, err
	}

	accFrom := acc.LoadAccount(execaddr)
	copyFrom := *accFrom
	accFrom.Balance = accFrom.GetBalance() - amount
	receiptBalanceTo := &types.ReceiptAccountTransfer{
		Prev:    &copyFrom,
		Current: accFrom,
	}
	acc.SaveAccount(accFrom)
	receipt2 := acc.ParaAssetWithdrawReceipt(accFrom, receiptBalanceTo)

	receipt = acc.mergeReceipt(receipt, receipt2)
	return receipt, nil
}
*/

func (acc *DB) ParaAssetWithdraw(addr string, amount int64, execaddr string) (*types.Receipt, error) {
	receipt, err := acc.ExecWithdraw(execaddr, addr, amount)
	return receipt, err
}

func (acc *DB) ParaAssetWithdrawReceipt(accFrom *types.Account, receiptTo proto.Message) *types.Receipt {
	ty := int32(types.TyLogParaAssetWithdraw)
	if acc.execer == types.ExecName(types.TokenX) {
		ty = int32(types.TyLogParaTokenAssetWithdraw)
	}
	log2 := &types.ReceiptLog{
		Ty:  ty,
		Log: types.Encode(receiptTo),
	}
	kv := acc.GetKVSet(accFrom)
	return &types.Receipt{
		Ty:   types.ExecOk,
		KV:   kv,
		Logs: []*types.ReceiptLog{log2},
	}
}
