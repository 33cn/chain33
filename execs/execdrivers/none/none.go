package none

//package none execer for unknow execer
//all none transaction exec ok, execept nofee
//nofee transaction will not pack into block

import (
	"code.aliyun.com/chain33/chain33/execs/execdrivers"
)

func init() {
	execdrivers.Register("none", newNone())
}

type None struct {
	execdrivers.ExecBase
}

func newNone() *None {
	n := &None{}
	n.SetChild(n)
	return n
}

func (n *None) GetName() string {
	return "none"
}
