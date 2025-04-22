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
	"bytes"

	"github.com/33cn/chain33/common/log"
	drivers "github.com/33cn/chain33/system/dapp"
	dtypes "github.com/33cn/chain33/system/dapp/coins/types"
	"github.com/33cn/chain33/types"
)

var clog = log.New("module", "execs.coins")
var driverName = "coins"

type subConfig struct {
	DisableAddrReceiver  bool     `json:"disableAddrReceiver"`
	DisableCheckTxAmount bool     `json:"disableCheckTxAmount"`
	FriendExecer         []string `json:"friendExecer,omitempty"`
}

var subCfg subConfig

// Init defines a register function
func Init(name string, cfg *types.Chain33Config, sub []byte) {
	if name != driverName {
		panic("system dapp can't be rename")
	}
	if sub != nil {
		types.MustDecode(sub, &subCfg)
	}
	// 需要先 RegisterDappFork才可以Register dapp
	drivers.Register(cfg, driverName, newCoins, cfg.GetDappFork(driverName, "Enable"))
	InitExecType()
}

// InitExecType the initialization process is relatively heavyweight, lots of reflect, so it's global
func InitExecType() {
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

	//TODO 数额在账户操作中会做检测, 此处不需要(提升性能), 需要确认在现有链中是否有分叉
	if !subCfg.DisableCheckTxAmount {
		ety := c.GetExecutorType()
		amount, err := ety.Amount(tx)
		if err != nil {
			return err
		}
		if amount < 0 {
			return types.ErrAmount
		}
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
	types.AssertConfig(c.GetAPI())
	cfg := c.GetAPI().GetConfig()
	if othertx.ActionName() == "miner" {
		for _, exec := range cfg.GetMinerExecs() {
			if cfg.ExecName(exec) == string(othertx.Execer) {
				return true
			}
		}
	}

	if !cfg.IsDappFork(c.GetHeight(), c.GetDriverName(), dtypes.ForkFriendExecerKey) {
		return false
	}
	//step3 从配置文件读取允许具体的执行器修改的本地合约的状态
	if types.IsEthSignID(othertx.GetSignature().GetTy()) {
		for _, friendExec := range subCfg.FriendExecer { //evm执行器操作coins 账户
			//myexec=coins 这种情况出现在evm 交易调用coins 转账的情况 ===> msg.Value!=0 && msg.Data!=nil && msg.To!=nil
			//1.tx.Exec=evm,user.p.xxx.evm
			//2.tx.amount !=0
			//3.writekey=mavl-coins-xxxx-xxxx
			if cfg.ExecName(friendExec) == string(othertx.GetExecer()) {
				//writekey: mavl-coins-para-0x93f200342d4154a0e025bd3a12128e8eb73b43a5
				//writekey: mavl-coins-bty-0xa42431da868c58877a627cc71dc95f01bf40c196
				if bytes.HasPrefix(writekey, []byte("mavl-coins-")) {
					return true
				}
			}
		}
	}

	return false
}

// CheckReceiptExecOk return true to check if receipt ty is ok
func (c *Coins) CheckReceiptExecOk() bool {
	return true
}
