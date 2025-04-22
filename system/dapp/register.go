// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dapp

//store package store the world - state data
import (
	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/common/address"
	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/types"
)

var elog = log.New("module", "execs")

// DriverCreate defines a drivercreate function
type DriverCreate func() Driver

// KVChecker checks kv stored in db
type KVChecker func(key, value []byte) bool

type driverWithHeight struct {
	create DriverCreate
	height int64
}

var (
	execDrivers          = make(map[string]*driverWithHeight)
	execAddressNameMap   = make(map[string]string)
	registedExecDriver   = make(map[string]*driverWithHeight)
	mvccKVExpiredChecker = make(map[string]KVChecker)
)

// Register register dcriver height in name
func Register(cfg *types.Chain33Config, name string, create DriverCreate, height int64) {
	if cfg == nil {
		panic("Execute: GetConfig is nil")
	}
	if create == nil {
		panic("Execute: Register driver is nil")
	}
	if _, dup := registedExecDriver[name]; dup {
		panic("Execute: Register called twice for driver " + name)
	}
	driverHeight := &driverWithHeight{
		create: create,
		height: height,
	}
	registedExecDriver[name] = driverHeight
	//考虑到前期平行链兼容性和防止误操作(平行链下转账到一个主链合约)，也会注册主链合约(不带前缀)的地址
	registerAddress(name)
	execDrivers[ExecAddress(name)] = driverHeight
	if cfg.IsPara() {
		paraHeight := cfg.GetFork("ForkEnableParaRegExec")
		if paraHeight < height {
			paraHeight = height
		}
		//平行链的合约地址是通过user.p.x.name计算的
		paraDriverName := cfg.ExecName(name)
		registerAddress(paraDriverName)
		execDrivers[ExecAddress(paraDriverName)] = &driverWithHeight{
			create: create,
			height: paraHeight,
		}
	}
}

// LoadDriver load driver
func LoadDriver(name string, height int64) (driver Driver, err error) {
	// user.evm.xxxx 的交易，使用evm执行器
	//   user.p.evm
	name = string(types.GetRealExecName([]byte(name)))
	c, ok := registedExecDriver[name]
	if !ok {
		elog.Debug("LoadDriver", "driver", name)
		return nil, types.ErrUnRegistedDriver
	}
	if height >= c.height || height == -1 {
		return c.create(), nil
	}
	return nil, types.ErrUnknowDriver
}

// LoadDriverWithClient load
func LoadDriverWithClient(qclent client.QueueProtocolAPI, name string, height int64) (driver Driver, err error) {
	driver, err = LoadDriver(name, height)
	if err != nil {
		return nil, err
	}
	driver.SetAPI(qclent)
	return driver, nil
}

// LoadDriverAllow load driver allow
func LoadDriverAllow(qclent client.QueueProtocolAPI, tx *types.Transaction, index int, height int64) (driver Driver) {
	exec, err := LoadDriverWithClient(qclent, string(tx.Execer), height)
	if err == nil {
		exec.SetEnv(height, 0, 0)
		err = exec.Allow(tx, index)
	}
	if err != nil {
		exec, err = LoadDriverWithClient(qclent, "none", height)
		if err != nil {
			panic(err)
		}
	} else {
		exec.SetName(string(types.GetRealExecName(tx.Execer)))
		exec.SetCurrentExecName(string(tx.Execer))
	}
	return exec
}

// IsDriverAddress whether or not execdrivers by address
func IsDriverAddress(addr string, height int64) bool {
	c, ok := execDrivers[addr]
	if !ok {
		return false
	}
	if height >= c.height || height == -1 {
		return true
	}
	return false
}

func registerAddress(name string) {
	if len(name) == 0 {
		panic("empty name string")
	}
	addr := ExecAddress(name)
	execAddressNameMap[name] = addr
}

// ExecAddress return exec address
func ExecAddress(name string) string {
	if addr, ok := execAddressNameMap[name]; ok {
		return addr
	}
	return address.ExecAddress(name)
}

// RegisterKVExpiredChecker registers dapp kv checker
func RegisterKVExpiredChecker(name string, f KVChecker) {
	// 目前只有 ticket 合约用到，如果合约里有需要精简的状态数据，可以自定义 KVChecker 校验规则
	// KVChecker 返回 true 表示状态数据不会再使用，精简数据库时可以删除
	if f == nil {
		panic("Execute: KVExpiredChecker is nil")
	}
	if _, dup := mvccKVExpiredChecker[name]; dup {
		panic("Execute: RegisterKVExpiredChecker called twice for " + name)
	}
	mvccKVExpiredChecker[name] = f
}

// LoadKVExpiredChecker loads dapp kv checker by dapp name
func LoadKVExpiredChecker(name string) (KVChecker, bool) {
	f, ok := mvccKVExpiredChecker[name]
	return f, ok
}

// KVExpiredCheckerList gets names of dapp which has registered kv checker
func KVExpiredCheckerList() []string {
	var checkerNames []string
	for name := range mvccKVExpiredChecker {
		checkerNames = append(checkerNames, name)
	}
	return checkerNames
}
