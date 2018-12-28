// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package util

import (
	"testing"

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
	assert.Equal(t, "/TEMP/hello/datadir", cfg.BlockChain.DbPath)

	cfg, _ = types.InitCfg("../cmd/chain33/chain33.toml")
	datadir = ResetDatadir(cfg, "~/hello")
	assert.Equal(t, datadir+"/datadir", cfg.BlockChain.DbPath)
}
