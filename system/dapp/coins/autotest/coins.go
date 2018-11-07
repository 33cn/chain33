package autotest

import (
	"reflect"

	. "gitlab.33.cn/chain33/chain33/cmd/autotest/types"
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
