package none

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
	execs.Register("none", newNone())
}

type None struct {
	db dbm.KVDB
}

func newNone() *None {
	return &None{}
}

func (n *None) Exec(tx *types.Transaction) *types.Receipt {
	acc, err := account.LoadAccount(n.db, tx.Account)
	if err != nil {
		//account not exist
		return errReceipt(err)
	}
	if acc.GetBalance()-tx.Fee >= 0 {
		acc.SetBalance(acc.GetBalance() - tx.Fee)
		account.SaveAccount(n.db, acc)
		return cutFeeReceipt(acc, tx)
	} else {
		return errReceipt(types.ErrNoBalance)
	}
}

func (n *None) SetDB(db dbm.KVDB) {
	n.db = db
}
