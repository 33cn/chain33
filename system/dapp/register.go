// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dapp

//store package store the world - state data
import (
	"github.com/33cn/chain33/common/address"
	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/types"
)

var elog = log.New("module", "execs")

// DriverCreate defines a drivercreate function
type DriverCreate func() Driver

type driverWithHeight struct {
	create DriverCreate
	height int64
}

var (
	execDrivers        = make(map[string]*driverWithHeight)
	execAddressNameMap = make(map[string]string)
	registedExecDriver = make(map[string]*driverWithHeight)
)

// Register register dcriver height in name
func Register(name string, create DriverCreate, height int64) {
	if create == nil {
		panic("Execute: Register driver is nil")
	}
	if _, dup := registedExecDriver[name]; dup {
		panic("Execute: Register called twice for driver " + name)
	}
	driverWithHeight := &driverWithHeight{
		create: create,
		height: height,
	}
	registedExecDriver[name] = driverWithHeight
	registerAddress(name)
	execDrivers[ExecAddress(name)] = driverWithHeight
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

// LoadDriverAllow load driver allow
func LoadDriverAllow(tx *types.Transaction, index int, height int64) (driver Driver) {
	exec, err := LoadDriver(string(tx.Execer), height)
	if err == nil {
		exec.SetEnv(height, 0, 0)
		err = exec.Allow(tx, index)
	}
	if err != nil {
		exec, err = LoadDriver("none", height)
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
