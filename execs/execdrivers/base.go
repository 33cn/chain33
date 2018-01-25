package execdrivers

//package none execer for unknow execer
//all none transaction exec ok, execept nofee
//nofee transaction will not pack into block

import (
	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/types"
)

type ExecBase struct {
	db        dbm.KVDB
	localdb   dbm.KVDB
	querydb   dbm.DB
	height    int64
	blocktime int64
	child     Executer
}

func NewExecBase() *ExecBase {
	return &ExecBase{}
}

func (n *ExecBase) SetEnv(height, blocktime int64) {
	n.height = height
	n.blocktime = blocktime
}

func (n *ExecBase) SetChild(e Executer) {
	n.child = e
}

func (n *ExecBase) GetAddr() string {
	return ExecAddress(n.child.GetName()).String()
}

func (n *ExecBase) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return nil, types.ErrActionNotSupport
}

func (n *ExecBase) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return nil, types.ErrActionNotSupport
}

func (n *ExecBase) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	return nil, types.ErrActionNotSupport
}

func (n *ExecBase) Query(funcname string, params types.Message) (types.Message, error) {
	return nil, types.ErrActionNotSupport
}

func (n *ExecBase) SetDB(db dbm.KVDB) {
	n.db = db
}

func (n *ExecBase) GetDB() dbm.KVDB {
	return n.db
}

func (n *ExecBase) SetLocalDB(db dbm.KVDB) {
	n.localdb = db
}

func (n *ExecBase) GetLocalDB() dbm.KVDB {
	return n.localdb
}

func (n *ExecBase) SetQueryDB(db dbm.DB) {
	n.querydb = db
}

func (n *ExecBase) GetQueryDB() dbm.DB {
	return n.querydb
}

func (n *ExecBase) GetHeight() int64 {
	return n.height
}

func (n *ExecBase) GetBlockTime() int64 {
	return n.blocktime
}
