package none

//package none execer for unknow execer
//all none transaction exec ok, execept nofee
//nofee transaction will not pack into block

import (
	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/execs"
	"code.aliyun.com/chain33/chain33/types"
)

var keyBuf [200]byte

var perfix = []byte("merkle-tree-")

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
	buf := append(keyBuf[:], perfix...)
	buf = append(buf, tx.Account...)
	data, err := n.db.Get(buf)
	if err != nil {
		//account not exist
	}
	return nil
}

func (n *None) SetDB(db dbm.KVDB) {
	n.db = db
}
