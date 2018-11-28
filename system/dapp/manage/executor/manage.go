// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package executor 管理插件执行器
package executor

import (
	log "github.com/33cn/chain33/common/log/log15"
	drivers "github.com/33cn/chain33/system/dapp"
	"github.com/33cn/chain33/types"
)

var (
	clog       = log.New("module", "execs.manage")
	driverName = "manage"
	conf       = types.ConfSub(driverName)
)

func init() {
	ety := types.LoadExecutorType(driverName)
	ety.InitFuncList(types.ListMethod(&Manage{}))
}

// Init resister a dirver
func Init(name string, sub []byte) {
	drivers.Register(GetName(), newManage, types.GetDappFork(driverName, "Enable"))
}

// GetName return manage name
func GetName() string {
	return newManage().GetName()
}

// Manage defines Manage object
type Manage struct {
	drivers.DriverBase
}

func newManage() drivers.Driver {
	c := &Manage{}
	c.SetChild(c)
	c.SetExecutorType(types.LoadExecutorType(driverName))
	return c
}

// GetDriverName return a drivername
func (c *Manage) GetDriverName() string {
	return driverName
}

// CheckTx checkout transaction
func (c *Manage) CheckTx(tx *types.Transaction, index int) error {
	return nil
}

// IsSuperManager is supper manager or not
func IsSuperManager(addr string) bool {
	for _, m := range conf.GStrList("superManager") {
		if addr == m {
			return true
		}
	}
	return false
}

// CheckReceiptExecOk return true to check if receipt ty is ok
func (c *Manage) CheckReceiptExecOk() bool {
	return true
}
