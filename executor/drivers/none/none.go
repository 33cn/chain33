package none

//package none execer for unknow execer
//all none transaction exec ok, execept nofee
//nofee transaction will not pack into block

import (
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/types"
)

func Init() {
	drivers.Register(newNone().GetName(), newNone, 0)
}

type None struct {
	drivers.DriverBase
}

func newNone() drivers.Driver {
	n := &None{}
	n.SetChild(n)
	return n
}

func (n *None) GetName() string {
	return types.ExecName("none")
}
