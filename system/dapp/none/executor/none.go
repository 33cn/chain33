// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package executor none执行器
package executor

// package none execer for unknow execer
// all none transaction exec ok, execept nofee
// nofee transaction will not pack into block

import (
	"github.com/33cn/chain33/common/log"
	drivers "github.com/33cn/chain33/system/dapp"
	ntypes "github.com/33cn/chain33/system/dapp/none/types"
	"github.com/33cn/chain33/types"
)

var (
	eLog       = log.New("module", "none.exec")
	driverName = ntypes.NoneX
)

// Init register newnone
func Init(name string, cfg *types.Chain33Config, sub []byte) {
	if name != driverName {
		panic("system dapp can't be rename")
	}
	driverName = name
	drivers.Register(cfg, name, newNone, 0)
	InitExecType()
}

// GetName return name at execution time
func GetName() string {
	return newNone().GetName()
}

// None defines a none type
type None struct {
	drivers.DriverBase
}

func newNone() drivers.Driver {
	n := &None{}
	n.SetChild(n)
	n.SetExecutorType(types.LoadExecutorType(ntypes.NoneX))
	return n
}

// InitExecType the initialization process is relatively heavyweight, lots of reflect, so it's global
func InitExecType() {
	ety := types.LoadExecutorType(driverName)
	ety.InitFuncList(types.ListMethod(&None{}))
}

// GetDriverName return dcrivername at register
func (n *None) GetDriverName() string {
	return driverName
}

// IsFriend 控制none合约db key修改权限
func (n *None) IsFriend(self, dbKey []byte, checkTx *types.Transaction) bool {

	execName := types.GetParaExecName(checkTx.Execer)
	// 延时存证交易需要在主链和平行链同时执行(#1262)
	if string(execName) == ntypes.NoneX &&
		ntypes.ActionName(checkTx) == ntypes.NameCommitDelayTxAction {
		return true
	}

	return false
}
