package autotest

import (
	. "gitlab.33.cn/chain33/chain33/cmd/autotest/types"
	"reflect"
)

type tradeAutoTest struct {
	TokenPreCreateCaseArr    []TokenPreCreateCase    `toml:"TokenPreCreateCase,omitempty"`
	TokenFinishCreateCaseArr []TokenFinishCreateCase `toml:"TokenFinishCreateCase,omitempty"`
	TransferCaseArr          []TransferCase          `toml:"TransferCase,omitempty"`
	SellCaseArr              []SellCase              `toml:"SellCase,omitempty"`
	DependBuyCaseArr         []DependBuyCase         `toml:"DependBuyCase,omitempty"`
}

func init() {

	RegisterAutoTest(&tradeAutoTest{})

}



func (config *tradeAutoTest) GetName() string {

	return "trade"
}


func (config *tradeAutoTest) GetTestConfigType() reflect.Type {

	return reflect.TypeOf(config)
}
