// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types_test

import (
	"testing"

	"strings"

	"github.com/33cn/chain33/common/address"
	_ "github.com/33cn/chain33/system"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

// how to create transafer for para
func TestCallCreateTxPara(t *testing.T) {
	str := types.ReadFile("testdata/guodun2.toml")
	new := strings.Replace(str, "Title=\"user.p.guodun2.\"", "Title=\"user.p.sto.\"", 1)
	cfg := types.NewChain33Config(new)

	req := &types.CreateTx{
		To:          "184wj4nsgVxKyz2NhM3Yb5RK5Ap6AFRFq2",
		Amount:      10,
		Fee:         1,
		Note:        []byte("12312"),
		IsWithdraw:  false,
		IsToken:     false,
		TokenSymbol: "",
		ExecName:    cfg.ExecName("coins"),
	}
	assert.True(t, cfg.IsPara())
	tx, err := types.CallCreateTransaction("coins", "", req)
	assert.Nil(t, err)
	tx, err = types.FormatTx(cfg, "coins", tx)
	assert.Nil(t, err)
	assert.Equal(t, "coins", string(tx.Execer))
	assert.Equal(t, address.ExecAddress("coins"), tx.To)
	tx, err = types.FormatTx(cfg, cfg.ExecName("coins"), tx)
	assert.Nil(t, err)
	assert.Equal(t, "user.p.sto.coins", string(tx.Execer))
	assert.Equal(t, address.ExecAddress("user.p.sto.coins"), tx.To)
}

func TestExecName(t *testing.T) {
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	assert.Equal(t, cfg.ExecName("coins"), "coins")
	cfg.SetTitleOnlyForTest("user.p.sto.")
	assert.Equal(t, cfg.ExecName("coins"), "user.p.sto.coins")
	//#在exec前面加一个 # 表示不重写执行器
	assert.Equal(t, cfg.ExecName("#coins"), "coins")
}
