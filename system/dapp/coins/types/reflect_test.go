package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/33cn/chain33/types"
)

func TestMethodCall(t *testing.T) {
	action := &CoinsAction{Value: &CoinsAction_Transfer{Transfer: &types.AssetsTransfer{}}}
	funclist := types.ListMethod(action)
	name, ty, v := types.GetActionValue(action, funclist)
	assert.Equal(t, int32(0), ty)
	assert.Equal(t, "Transfer", name)
	assert.Equal(t, &types.AssetsTransfer{}, v.Interface())
}

func TestListMethod(t *testing.T) {
	action := &CoinsAction{Value: &CoinsAction_Transfer{Transfer: &types.AssetsTransfer{}}}
	funclist := types.ListMethod(action)
	excpect := []string{"GetWithdraw", "GetGenesis", "GetTransfer", "GetTransferToExec", "GetValue"}
	for _, v := range excpect {
		if _, ok := funclist[v]; !ok {
			t.Error(v + " is not in list")
		}
	}
}

func TestListType(t *testing.T) {
	excpect := []string{"Value_Withdraw", "Withdraw", "Value_Transfer", "Value_Genesis", "Value_TransferToExec"}
	for _, v := range excpect {
		if _, ok := NewType().GetValueTypeMap()[v]; !ok {
			t.Error(v + " is not in list")
		}
	}
}
func BenchmarkGetActionValue(b *testing.B) {
	action := &CoinsAction{Value: &CoinsAction_Transfer{Transfer: &types.AssetsTransfer{}}}
	funclist := types.ListMethod(action)
	for i := 0; i < b.N; i++ {
		_, _, v := types.GetActionValue(action, funclist)
		assert.NotNil(b, v)
	}
}
