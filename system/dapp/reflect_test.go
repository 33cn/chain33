package dapp

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	cty "gitlab.33.cn/chain33/chain33/system/dapp/coins/types"
)

func TestMethodCall(t *testing.T) {
	action := &cty.CoinsAction{Value: &cty.CoinsAction_Transfer{Transfer: &cty.CoinsTransfer{}}}
	base := action.GetValue()
	//typ := reflect.TypeOf(base)
	rcvr := reflect.ValueOf(base)
	sname := reflect.Indirect(rcvr).Type().Name()
	//assert.Equal(t, "DriverBase2", sname)
	assert.Equal(t, "CoinsAction_Transfer", sname)
	datas := strings.Split(sname, "_")
	values := reflect.ValueOf(action).MethodByName("Get" + datas[1]).Call([]reflect.Value{})
	assert.Equal(t, 1, len(values))
	name, ty, v := GetActionValue(action)
	assert.Equal(t, int32(0), ty)
	assert.Equal(t, "Transfer", name)
	assert.Equal(t, &cty.CoinsTransfer{}, v.Interface())
}

func BenchmarkGetActionValue(b *testing.B) {
	action := &cty.CoinsAction{Value: &cty.CoinsAction_Transfer{Transfer: &cty.CoinsTransfer{}}}
	for i := 0; i < b.N; i++ {
		_, _, v := GetActionValue(action)
		assert.NotNil(b, v)
	}
}
