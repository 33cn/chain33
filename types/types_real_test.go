// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types_test

import (
	"testing"

	"github.com/33cn/chain33/common/address"
	_ "github.com/33cn/chain33/system"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

//how to create transafer for para
func TestCallCreateTxPara(t *testing.T) {
	ti := types.GetTitle()
	defer types.SetTitleOnlyForTest(ti)
	types.SetTitleOnlyForTest("user.p.sto.")
	req := &types.CreateTx{
		To:          "184wj4nsgVxKyz2NhM3Yb5RK5Ap6AFRFq2",
		Amount:      10,
		Fee:         1,
		Note:        []byte("12312"),
		IsWithdraw:  false,
		IsToken:     false,
		TokenSymbol: "",
		ExecName:    types.ExecName("coins"),
	}
	assert.True(t, types.IsPara())
	tx, err := types.CallCreateTransaction("coins", "", req)
	assert.Nil(t, err)
	tx, err = types.FormatTx("coins", tx)
	assert.Nil(t, err)
	assert.Equal(t, "coins", string(tx.Execer))
	assert.Equal(t, address.ExecAddress("coins"), tx.To)
	tx, err = types.FormatTx(types.ExecName("coins"), tx)
	assert.Nil(t, err)
	assert.Equal(t, "user.p.sto.coins", string(tx.Execer))
	assert.Equal(t, address.ExecAddress("user.p.sto.coins"), tx.To)
}

func TestExecName(t *testing.T) {
	assert.Equal(t, types.ExecName("coins"), "coins")
	ti := types.GetTitle()
	defer types.SetTitleOnlyForTest(ti)
	types.SetTitleOnlyForTest("user.p.sto.")
	assert.Equal(t, types.ExecName("coins"), "user.p.sto.coins")
	//#在exec前面加一个 # 表示不重写执行器
	assert.Equal(t, types.ExecName("#coins"), "coins")
}
