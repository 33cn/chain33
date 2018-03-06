package none

//package none execer for unknow execer
//all none transaction exec ok, execept nofee
//nofee transaction will not pack into block

import (
	"code.aliyun.com/chain33/chain33/executor/drivers"
	"code.aliyun.com/chain33/chain33/types"
)

func init() {
	drivers.Register("none", newNone())
}

type None struct {
	drivers.ExecBase
}

func newNone() *None {
	n := &None{}
	n.SetChild(n)
	return n
}

func (n *None) GetName() string {
	return "none"
}

func (n *None) GetActionName(tx *types.Transaction) string {
	return tx.ActionName()
}

func (n *None) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	return n.ExecCommon(tx, index)
}

func (n *None) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return n.ExecLocalCommon(tx, receipt, index)
}

func (n *None) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return n.ExecDelLocalCommon(tx, receipt, index)
}
