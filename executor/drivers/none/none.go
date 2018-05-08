package none

//package none execer for unknow execer
//all none transaction exec ok, execept nofee
//nofee transaction will not pack into block

import (
	"gitlab.33.cn/chain33/chain33/executor/drivers"
)

func Init() {
	n := newNone()
	drivers.Register(n.GetName(), n, 0)
}

type None struct {
	drivers.DriverBase
}

func newNone() *None {
	n := &None{}
	n.SetChild(n)
	return n
}

func (n *None) GetName() string {
	return "none"
}

func (n *None) Clone() drivers.Driver {
	clone := &None{}
	clone.DriverBase = *(n.DriverBase.Clone().(*drivers.DriverBase))
	clone.SetChild(clone)
	return clone
}
