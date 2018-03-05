package account

import (
	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/types"
)

func GenesisInit(db dbm.KVDB, addr string, amount int64) (*types.Receipt, error) {
	accTo := LoadAccount(db, addr)
	copyto := *accTo
	accTo.Balance = accTo.GetBalance() + amount
	receiptBalanceTo := &types.ReceiptAccountTransfer{&copyto, accTo}
	SaveAccount(db, accTo)
	receipt := genesisReceipt(accTo, receiptBalanceTo)
	return receipt, nil
}

func GenesisInitExec(db dbm.KVDB, addr string, amount int64, execaddr string) (*types.Receipt, error) {
	accTo := LoadAccount(db, execaddr)
	copyto := *accTo
	accTo.Balance = accTo.GetBalance() + amount
	receiptBalanceTo := &types.ReceiptAccountTransfer{&copyto, accTo}
	SaveAccount(db, accTo)
	receipt := genesisReceipt(accTo, receiptBalanceTo)
	receipt2, err := execDeposit(db, addr, execaddr, amount)
	if err != nil {
		panic(err)
	}
	receipt2.Ty = types.TyLogGenesisDeposit
	receipt = mergeReceipt(receipt, receipt2)
	return receipt, nil
}

func genesisReceipt(accTo *types.Account, receiptTo *types.ReceiptAccountTransfer) *types.Receipt {
	log2 := &types.ReceiptLog{types.TyLogGenesisTransfer, types.Encode(receiptTo)}
	kv := GetKVSet(accTo)
	return &types.Receipt{types.ExecOk, kv, []*types.ReceiptLog{log2}}
}
