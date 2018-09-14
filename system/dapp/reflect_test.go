package dapp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	cty "gitlab.33.cn/chain33/chain33/system/dapp/coins/types"
)

func TestMethodCall(t *testing.T) {
	action := &cty.CoinsAction{Value: &cty.CoinsAction_Transfer{Transfer: &cty.CoinsTransfer{}}}
	_, _, _, f := action.XXX_OneofFuncs()
	funclist := ListMethod(action, f)
	name, ty, v := GetActionValue(action, funclist)
	assert.Equal(t, int32(0), ty)
	assert.Equal(t, "Transfer", name)
	assert.Equal(t, &cty.CoinsTransfer{}, v.Interface())
}

func TestListMethod(t *testing.T) {
	action := &cty.CoinsAction{Value: &cty.CoinsAction_Transfer{Transfer: &cty.CoinsTransfer{}}}
	_, _, _, f := action.XXX_OneofFuncs()
	funclist := ListMethod(action, f)
	excpect := []string{"GetWithdraw", "GetGenesis", "GetTransfer", "GetTransferToExec", "GetValue"}
	for _, v := range excpect {
		if _, ok := funclist[v]; !ok {
			t.Error(v + " is not in list")
		}
	}
}

func BenchmarkGetActionValue(b *testing.B) {
	action := &cty.CoinsAction{Value: &cty.CoinsAction_Transfer{Transfer: &cty.CoinsTransfer{}}}
	_, _, _, f := action.XXX_OneofFuncs()
	funclist := ListMethod(action, f)
	for i := 0; i < b.N; i++ {
		_, _, v := GetActionValue(action, funclist)
		assert.NotNil(b, v)
	}
}
