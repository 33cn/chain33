package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.33.cn/chain33/chain33/types"
)

func TestMethodCall(t *testing.T) {
	action := &CoinsAction{Value: &CoinsAction_Transfer{Transfer: &types.AssetsTransfer{}}}
	funclist := ListMethod(action)
	name, ty, v := GetActionValue(action, funclist)
	assert.Equal(t, int32(0), ty)
	assert.Equal(t, "Transfer", name)
	assert.Equal(t, &types.AssetsTransfer{}, v.Interface())
}

func TestListMethod(t *testing.T) {
	action := &CoinsAction{Value: &CoinsAction_Transfer{Transfer: &types.AssetsTransfer{}}}
	funclist := ListMethod(action)
	excpect := []string{"GetWithdraw", "GetGenesis", "GetTransfer", "GetTransferToExec", "GetValue"}
	for _, v := range excpect {
		if _, ok := funclist[v]; !ok {
			t.Error(v + " is not in list")
		}
	}
}

func BenchmarkGetActionValue(b *testing.B) {
	action := &CoinsAction{Value: &CoinsAction_Transfer{Transfer: &types.AssetsTransfer{}}}
	funclist := ListMethod(action)
	for i := 0; i < b.N; i++ {
		_, _, v := GetActionValue(action, funclist)
		assert.NotNil(b, v)
	}
}
