package none

//package none execer for unknow execer
//all none transaction exec ok, execept nofee
//nofee transaction will not pack into block

import (
	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/execs"
	"code.aliyun.com/chain33/chain33/types"
)

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
	return nil
}

func (n *None) SetDB(db dbm.KVDB) {
	n.db = db
}
