// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

//package none execer for unknow execer
//all none transaction exec ok, execept nofee
//nofee transaction will not pack into block

import (
	drivers "github.com/33cn/chain33/system/dapp"
)

var driverName = "none"

func Init(name string, sub []byte) {
	if name != driverName {
		panic("system dapp can't be rename")
	}
	driverName = name
	drivers.Register(name, newNone, 0)
}

//执行时候的名称
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

//驱动注册时候的名称
func (n *None) GetDriverName() string {
	return driverName
}
