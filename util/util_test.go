// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package util

import (
	"testing"

	"github.com/33cn/chain33/common/address"

	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

func TestMakeStringUpper(t *testing.T) {
	originStr := "abcdefg"
	destStr, err := MakeStringToUpper(originStr, 0, 1)
	assert.NoError(t, err)
	assert.Equal(t, "Abcdefg", destStr)

	destStr, err = MakeStringToUpper(originStr, 2, 2)
	assert.NoError(t, err)
	assert.Equal(t, "abCDefg", destStr)

	_, err = MakeStringToUpper(originStr, -1, 2)
	assert.Error(t, err)
}

func TestMakeStringLower(t *testing.T) {
	originStr := "ABCDEFG"
	destStr, err := MakeStringToLower(originStr, 0, 1)
	assert.NoError(t, err)
	assert.Equal(t, "aBCDEFG", destStr)

	destStr, err = MakeStringToLower(originStr, 2, 2)
	assert.NoError(t, err)
	assert.Equal(t, "ABcdEFG", destStr)

	_, err = MakeStringToLower(originStr, -1, 2)
	assert.Error(t, err)
}

func TestResetDatadir(t *testing.T) {
	cfg, _ := types.InitCfg("../cmd/chain33/chain33.toml")
	datadir := ResetDatadir(cfg, "$TEMP/hello")
	assert.Equal(t, datadir+"/datadir", cfg.BlockChain.DbPath)

	cfg, _ = types.InitCfg("../cmd/chain33/chain33.toml")
	datadir = ResetDatadir(cfg, "/TEMP/hello")
	assert.Equal(t, datadir+"/datadir", cfg.BlockChain.DbPath)

	cfg, _ = types.InitCfg("../cmd/chain33/chain33.toml")
	datadir = ResetDatadir(cfg, "~/hello")
	assert.Equal(t, datadir+"/datadir", cfg.BlockChain.DbPath)
}

func TestHexToPrivkey(t *testing.T) {
	key := HexToPrivkey("4257D8692EF7FE13C68B65D6A52F03933DB2FA5CE8FAF210B5B8B80C721CED01")
	addr := address.PubKeyToAddress(key.PubKey().Bytes()).String()
	assert.Equal(t, addr, "12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv")
}

func TestGetParaExecName(t *testing.T) {
	s := GetParaExecName("user.p.hello.", "world")
	assert.Equal(t, "user.p.hello.world", s)
	s = GetParaExecName("user.p.hello.", "user.p.2.world")
	assert.Equal(t, "user.p.2.world", s)
}

func TestUpperLower(t *testing.T) {
	out, err := MakeStringToUpper("hello", 0, 1)
	assert.Nil(t, err)
	assert.Equal(t, "Hello", out)

	out, err = MakeStringToUpper("Hello", 0, 1)
	assert.Nil(t, err)
	assert.Equal(t, "Hello", out)

	out, err = MakeStringToUpper("Hello", -1, 1)
	assert.NotNil(t, err)

	out, err = MakeStringToUpper("Hello", 1, -1)
	assert.NotNil(t, err)

	out, err = MakeStringToLower("hello", 0, 1)
	assert.Nil(t, err)
	assert.Equal(t, "hello", out)

	out, err = MakeStringToLower("Hello", 0, 1)
	assert.Nil(t, err)
	assert.Equal(t, "hello", out)

	out, err = MakeStringToLower("Hello", -1, 1)
	assert.NotNil(t, err)

	out, err = MakeStringToLower("Hello", 1, -1)
	assert.NotNil(t, err)
}
