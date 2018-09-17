package executor

//package none execer for unknow execer
//all none transaction exec ok, execept nofee
//nofee transaction will not pack into block

import (
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
)

func Init() {
	drivers.Register(GetName(), newNone, 0)
}

func GetName() string {
	return newNone().GetName()
}

type None struct {
	drivers.DriverBase
}

func newNone() drivers.Driver {
	n := &None{}
	n.SetChild(n)
	return n
}

func (n *None) GetDriverName() string {
	return "none"
}
