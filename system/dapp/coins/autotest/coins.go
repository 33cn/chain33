// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package autotest

import (
	"reflect"

	. "github.com/33cn/chain33/cmd/autotest/types"
)

type coinsAutoTest struct {
	SimpleCaseArr   []SimpleCase   `toml:"SimpleCase,omitempty"`
	TransferCaseArr []TransferCase `toml:"TransferCase,omitempty"`
	WithdrawCaseArr []WithdrawCase `toml:"WithdrawCase,omitempty"`
}

func init() {

	RegisterAutoTest(coinsAutoTest{})

}

func (config coinsAutoTest) GetName() string {

	return "coins"
}

func (config coinsAutoTest) GetTestConfigType() reflect.Type {

	return reflect.TypeOf(config)
}
