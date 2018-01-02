package none

//package none execer for unknow execer
//all none transaction exec ok, execept nofee
//nofee transaction will not pack into block

import (
	"code.aliyun.com/chain33/chain33/account"
	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/execs/execdrivers"
	"code.aliyun.com/chain33/chain33/types"
)

var keyBuf [200]byte

func init() {
	execdrivers.Register("none", newNone())
}

type None struct {
	db        dbm.KVDB
	height    int64
	blocktime int64
}

func newNone() *None {
	return &None{}
}

func (n *None) SetEnv(height, blocktime int64) {
	n.height = height
	n.blocktime = blocktime
}

func (n *None) Exec(tx *types.Transaction) (*types.Receipt, error) {
	return nil, types.ErrActionNotSupport
}

func (n *None) SetDB(db dbm.KVDB) {
	n.db = db
}

func cutFeeReceipt(acc *types.Account, receiptBalance *types.ReceiptBalance) *types.Receipt {
	feelog := &types.ReceiptLog{types.TyLogFee, types.Encode(receiptBalance)}
	return &types.Receipt{types.ExecErr, account.GetKVSet(acc), []*types.ReceiptLog{feelog}}
}
