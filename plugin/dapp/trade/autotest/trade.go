package autotest

import (
	. "gitlab.33.cn/chain33/chain33/cmd/autotest/types"
	. "gitlab.33.cn/chain33/chain33/plugin/dapp/token/autotest"
	. "gitlab.33.cn/chain33/chain33/system/dapp/coins/autotest"
	"reflect"
)

type tradeAutoTest struct {
	SimpleCaseArr            []SimpleCase            `toml:"SimpleCase,omitempty"`
	TokenPreCreateCaseArr    []TokenPreCreateCase    `toml:"TokenPreCreateCase,omitempty"`
	TokenFinishCreateCaseArr []TokenFinishCreateCase `toml:"TokenFinishCreateCase,omitempty"`
	TransferCaseArr          []TransferCase          `toml:"TransferCase,omitempty"`
	SellCaseArr              []SellCase              `toml:"SellCase,omitempty"`
	DependBuyCaseArr         []DependBuyCase         `toml:"DependBuyCase,omitempty"`
}

func init() {

	RegisterAutoTest(tradeAutoTest{})

}



func (config tradeAutoTest) GetName() string {

	return "trade"
}


func (config tradeAutoTest) GetTestConfigType() reflect.Type {

	return reflect.TypeOf(config)
}
