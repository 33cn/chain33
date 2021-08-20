// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"testing"

	rpctypes "github.com/33cn/chain33/rpc/types"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

func TestCheckExpireOpt(t *testing.T) {
	expire := "0s"
	str, err := CheckExpireOpt(expire)
	assert.NoError(t, err)
	assert.Equal(t, "0s", str)

	expire = "14s"
	str, err = CheckExpireOpt(expire)
	assert.NoError(t, err)
	assert.Equal(t, "120s", str)

	expire = "14"
	str, err = CheckExpireOpt(expire)
	assert.NoError(t, err)
	assert.Equal(t, "14", str)

	expire = ""
	_, err = CheckExpireOpt(expire)
	assert.Error(t, err)

	expire = "H:-123"
	_, err = CheckExpireOpt(expire)
	assert.Error(t, err)

	expire = "-123"
	_, err = CheckExpireOpt(expire)
	assert.Error(t, err)

	expire = "H:123"
	str, err = CheckExpireOpt(expire)
	assert.NoError(t, err)
	assert.Equal(t, "H:123", str)

}

func TestDecodeTransaction(t *testing.T) {
	tx := &rpctypes.Transaction{
		Execer: "coins",
	}
	result := DecodeTransaction(tx)
	assert.Equal(t, result.Execer, "coins")
}

func TestDecodeAccount(t *testing.T) {
	precision := types.DefaultCoinPrecision
	acc := &types.Account{
		Currency: 2,
		Balance:  3 * precision,
		Frozen:   4 * precision,
		Addr:     "0x123",
	}
	accResult := DecodeAccount(acc, precision)
	assert.Equal(t, &AccountResult{2, "3.0000", "4.0000", "0x123"}, accResult)
}

func TestCreateRawTx(t *testing.T) {
	paraName := ""
	cfg := &rpctypes.ChainConfigInfo{
		Title:         "chain33",
		CoinExec:      types.DefaultCoinsExec,
		CoinSymbol:    types.DefaultCoinsSymbol,
		CoinPrecision: types.DefaultCoinPrecision,
		IsPara:        false,
	}

	var err error
	_, err = CreateRawTx(paraName, "", 0, "", false, "", "", cfg)
	assert.Nil(t, err)
	_, err = CreateRawTx(paraName, "", 0, "", false, "", "coins", cfg)
	assert.Nil(t, err)
	_, err = CreateRawTx(paraName, "", 0, "", true, "", "", cfg)
	assert.Nil(t, err)
	_, err = CreateRawTx(paraName, "", -1, "", false, "", "", cfg)
	assert.Equal(t, types.ErrAmount, err)
	_, err = CreateRawTx(paraName, "", 1e10, "", false, "", "", cfg)
	assert.Equal(t, types.ErrAmount, err)
	_, err = CreateRawTx(paraName, "", 0, "", false, "", "coins-", cfg)
	assert.Equal(t, types.ErrExecNameNotMatch, err)
}

func TestGetExecAddr(t *testing.T) {
	_, err := GetExecAddr("coins")
	assert.Nil(t, err)
}

func TestGetRealExecName(t *testing.T) {
	s := getRealExecName("user.p.fzmtest.", "user.p.game")
	assert.Equal(t, "user.p.game", s)
	s = getRealExecName("user.p.fzmtest.", "game")
	assert.Equal(t, "user.p.fzmtest.game", s)
}
