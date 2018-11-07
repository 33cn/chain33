package autotest

import (
	. "gitlab.33.cn/chain33/chain33/cmd/autotest/types"
	"reflect"

	. "gitlab.33.cn/chain33/chain33/plugin/dapp/token/autotest"
	. "gitlab.33.cn/chain33/chain33/system/dapp/coins/autotest"
)

type privacyAutoTest struct {
	SimpleCaseArr            []SimpleCase            `toml:"SimpleCase,omitempty"`
	TokenPreCreateCaseArr    []TokenPreCreateCase    `toml:"TokenPreCreateCase,omitempty"`
	TokenFinishCreateCaseArr []TokenFinishCreateCase `toml:"TokenFinishCreateCase,omitempty"`
	TransferCaseArr          []TransferCase          `toml:"TransferCase,omitempty"`
	PubToPrivCaseArr         []PubToPrivCase         `toml:"PubToPrivCase,omitempty"`
	PrivToPrivCaseArr        []PrivToPrivCase        `toml:"PrivToPrivCase,omitempty"`
	PrivToPubCaseArr         []PrivToPubCase         `toml:"PrivToPubCase,omitempty"`
	CreateUtxosCaseArr   	 []CreateUtxosCase       `toml:"CreateUtxosCase,omitempty"`
}


func init() {

	RegisterAutoTest(privacyAutoTest{})

}



func (config privacyAutoTest) GetName() string {

	return "privacy"
}


func (config privacyAutoTest) GetTestConfigType() reflect.Type {

	return reflect.TypeOf(config)
}