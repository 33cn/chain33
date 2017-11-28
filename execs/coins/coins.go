package coins

/*
coins 是一个货币的exec。内置货币的执行器。

主要提供两种操作：

EventFee -> 扣除手续费
EventTransfer -> 转移资产
*/

//package none execer for unknow execer
//all none transaction exec ok, execept nofee
//nofee transaction will not pack into block

import (
	"code.aliyun.com/chain33/chain33/account"
	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/execs"
	"code.aliyun.com/chain33/chain33/types"
)

var keyBuf [200]byte

func init() {
	execs.Register("coins", newCoins())
}

type Coins struct {
	db dbm.KVDB
}

func newCoins() *Coins {
	return &Coins{}
}

func (n *None) Exec(tx *types.Transaction) *types.Receipt {
	acc, err := account.LoadAccount(n.db, tx.Account)
	if err != nil {
		//account not exist
		return errReceipt(err)
	}
	if acc.GetBalance()-tx.Fee >= 0 {
		receiptBalance := &types.ReceiptBalance{acc.GetBalance(), acc.GetBalance() - tx.Fee, tx.Fee}
		acc.Balance = acc.GetBalance() - tx.Fee
		account.SaveAccount(n.db, acc)
		return cutFeeReceipt(acc, receiptBalance)
	} else {
		return errReceipt(types.ErrNoBalance)
	}
}

func (n *None) SetDB(db dbm.KVDB) {
	n.db = db
}

func errReceipt(err error) *types.Receipt {
	berr := err.Error()
	errlog := &types.ReceiptLog{types.TyLogErr, []byte(berr)}
	return &types.Receipt{types.ExecErr, nil, []*types.ReceiptLog{errlog}}
}

func cutFeeReceipt(acc *types.Account, receiptBalance *types.ReceiptBalance) *types.Receipt {
	feelog := &types.ReceiptLog{types.TyLogFee, types.Encode(receiptBalance)}
	return &types.Receipt{types.ExecErr, account.GetKVSet(acc), []*types.ReceiptLog{feelog}}
}
