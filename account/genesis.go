package account

import (
	"gitlab.33.cn/chain33/chain33/types"
)

func (acc *AccountDB) GenesisInit(addr string, amount int64) (*types.Receipt, error) {
	accTo := acc.LoadAccount(addr)
	copyto := *accTo
	accTo.Balance = accTo.GetBalance() + amount
	receiptBalanceTo := &types.ReceiptAccountTransfer{&copyto, accTo}
	acc.SaveAccount(accTo)
	receipt := acc.genesisReceipt(accTo, receiptBalanceTo)
	return receipt, nil
}

func (acc *AccountDB) GenesisInitExec(addr string, amount int64, execaddr string) (*types.Receipt, error) {
	accTo := acc.LoadAccount(execaddr)
	copyto := *accTo
	accTo.Balance = accTo.GetBalance() + amount
	receiptBalanceTo := &types.ReceiptAccountTransfer{&copyto, accTo}
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

func (acc *AccountDB) genesisReceipt(accTo *types.Account, receiptTo *types.ReceiptAccountTransfer) *types.Receipt {
	ty := int32(types.TyLogGenesisTransfer)
	if acc.IsTokenAccount() {
		ty = int32(types.TyLogTokenGenesisTransfer)
	}
	log2 := &types.ReceiptLog{ty, types.Encode(receiptTo)}
	kv := acc.GetKVSet(accTo)
	return &types.Receipt{types.ExecOk, kv, []*types.ReceiptLog{log2}}
}
