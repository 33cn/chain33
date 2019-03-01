// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package executor coins执行器
package executor

/*
coins 是一个货币的exec。内置货币的执行器。

主要提供两种操作：
EventTransfer -> 转移资产
*/

// package none execer for unknow execer
// all none transaction exec ok, execept nofee
// nofee transaction will not pack into block

import (
	drivers "github.com/33cn/chain33/system/dapp"
	"github.com/33cn/chain33/types"
)

// var clog = log.New("module", "execs.coins")
var driverName = "coins"

// Init defines a register function
func Init(name string, sub []byte) {
	if name != driverName {
		panic("system dapp can't be rename")
	}
	drivers.Register(driverName, newCoins, types.GetDappFork(driverName, "Enable"))
}

// the initialization process is relatively heavyweight, lots of reflact, so it's global

func init() {
	ety := types.LoadExecutorType(driverName)
	ety.InitFuncList(types.ListMethod(&Coins{}))
}

// GetName return name string
func GetName() string {
	return newCoins().GetName()
}

// Coins defines coins
type Coins struct {
	drivers.DriverBase
}

func newCoins() drivers.Driver {
	c := &Coins{}
	c.SetChild(c)
	c.SetExecutorType(types.LoadExecutorType(driverName))
	return c
}

// GetDriverName get drive name
func (c *Coins) GetDriverName() string {
	return driverName
}

// CheckTx check transaction amount 必须不能为负数
func (c *Coins) CheckTx(tx *types.Transaction, index int) error {
	ety := c.GetExecutorType()
	amount, err := ety.Amount(tx)
	if err != nil {
		return err
	}
	if amount < 0 {
		return types.ErrAmount
	}
	return nil
}

// IsFriend coins contract  the mining transaction that runs the ticket contract
func (c *Coins) IsFriend(myexec, writekey []byte, othertx *types.Transaction) bool {
	//step1 先判定自己合约的权限
	if !c.AllowIsSame(myexec) {
		return false
	}
	//step2 判定 othertx 的 执行器名称(只允许主链，并且是挖矿的行为)
	if string(othertx.Execer) == "ticket" && othertx.ActionName() == "miner" {
		return true
	}
	return false
}

// CheckReceiptExecOk return true to check if receipt ty is ok
func (c *Coins) CheckReceiptExecOk() bool {
	return true
}
