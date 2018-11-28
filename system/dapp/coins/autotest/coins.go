// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package autotest 系统级coins dapp自动测试包
package autotest

import (
	"reflect"

	"github.com/33cn/chain33/cmd/autotest/types"
)

type coinsAutoTest struct {
	SimpleCaseArr   []types.SimpleCase `toml:"SimpleCase,omitempty"`
	TransferCaseArr []TransferCase     `toml:"TransferCase,omitempty"`
	WithdrawCaseArr []WithdrawCase     `toml:"WithdrawCase,omitempty"`
}

func init() {

	types.RegisterAutoTest(coinsAutoTest{})

}

// GetName return name = "coins"
func (config coinsAutoTest) GetName() string {

	return "coins"
}

// GetTestConfigType return type of config
func (config coinsAutoTest) GetTestConfigType() reflect.Type {

	return reflect.TypeOf(config)
}
