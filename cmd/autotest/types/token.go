package types

import (
	"reflect"

	. "gitlab.33.cn/chain33/chain33/cmd/autotest/types"
)

type tokenAutoTest struct {
	TokenPreCreateCaseArr    []TokenPreCreateCase    `toml:"TokenPreCreateCase,omitempty"`
	TokenFinishCreateCaseArr []TokenFinishCreateCase `toml:"TokenFinishCreateCase,omitempty"`
	TransferCaseArr          []TransferCase          `toml:"TransferCase,omitempty"`
	WithdrawCaseArr          []WithdrawCase          `toml:"WithdrawCase,omitempty"`
	TokenRevokeCaseArr       []TokenRevokeCase       `toml:"TokenRevokeCase,omitempty"`
}

func init() {

	RegisterAutoTest(&tokenAutoTest{})

}



func (config *tokenAutoTest) GetName() string {

	return "token"
}


func (config *tokenAutoTest) GetTestConfigType() reflect.Type {

	return reflect.TypeOf(config)
}