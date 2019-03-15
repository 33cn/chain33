// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

func TestMethodCall(t *testing.T) {
	action := &CoinsAction{Value: &CoinsAction_Transfer{Transfer: &types.AssetsTransfer{}}}
	funclist := types.ListMethod(action)
	name, ty, v, err := types.GetActionValue(action, funclist)
	assert.Nil(t, err)
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
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		action, ty, _, _ := types.GetActionValue(action, funclist)
		if action != "Transfer" || ty != 0 {
			b.Fatal(action)
		}
	}
}
func BenchmarkDecodePayload(b *testing.B) {
	action := &CoinsAction{Value: &CoinsAction_Transfer{Transfer: &types.AssetsTransfer{}}}
	payload := types.Encode(action)
	tx := &types.Transaction{Payload: payload}
	ty := NewType()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ty.DecodePayload(tx)
	}
}

func BenchmarkDecodePayloadValue(b *testing.B) {
	b.ReportAllocs()
	action := &CoinsAction{Value: &CoinsAction_Transfer{Transfer: &types.AssetsTransfer{}}, Ty: CoinsActionTransfer}
	payload := types.Encode(action)
	tx := &types.Transaction{Payload: payload}
	ty := NewType()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ty.DecodePayloadValue(tx)
	}
}
