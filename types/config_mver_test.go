// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigFlat(t *testing.T) {
	conf := map[string]interface{}{
		"key1": "value1",
		"key2": map[string]interface{}{
			"key21": "value21",
		},
	}
	flat := FlatConfig(conf)
	assert.Equal(t, "value1", flat["key1"])
	assert.Equal(t, "value21", flat["key2.key21"])
}

func TestConfigMverInit(t *testing.T) {
	RegisterDappFork("store-kvmvccmavl", "ForkKvmvccmavl", MaxHeight)
	cfg, _ := InitCfg("testdata/local.mvertest.toml")
	Init(cfg.Title, cfg)
	assert.Equal(t, MGStr("mver.consensus.name2", 0), "ticket-bityuan")
	assert.Equal(t, MGStr("mver.consensus.name2", 10), "ticket-bityuanv5")
	assert.Equal(t, MGStr("mver.consensus.name2", 25), "ticket-bityuanv2")
	assert.Equal(t, MGStr("mver.hello", 0), "world")
	assert.Equal(t, MGStr("mver.hello", 11), "forkworld")
	assert.Equal(t, MGStr("mver.nofork", 0), "nofork")
	assert.Equal(t, MGStr("mver.nofork", 9), "nofork")
	assert.Equal(t, MGStr("mver.nofork", 11), "nofork")

	assert.Equal(t, MGStr("mver.exec.sub.coins.name2", -1), "ticket-bityuan")
	assert.Equal(t, MGStr("mver.exec.sub.coins.name2", 0), "ticket-bityuanv5-enable")
	assert.Equal(t, MGStr("mver.exec.sub.coins.name2", 9), "ticket-bityuanv5-enable")
	assert.Equal(t, MGStr("mver.exec.sub.coins.name2", 10), "ticket-bityuanv5")
	assert.Equal(t, MGStr("mver.exec.sub.coins.name2", 11), "ticket-bityuanv5")
	assert.Equal(t, int64(1), GetDappFork("store-kvmvccmavl", "ForkKvmvccmavl"))

	assert.Equal(t, MGInt("mver.consensus.ticket.ticketPrice", 1), int64(10000))
	assert.Equal(t, MGInt("mver.consensus.ticket.ticketPrice", 11), int64(20000))
	assert.Equal(t, MGInt("mver.consensus.ticket.ticketPrice", 21), int64(30000))

	assert.Equal(t, MGInt("mver.consensus.ticket.ticketFrozenTime", -2), int64(5))
	assert.Equal(t, MGInt("mver.consensus.ticket.ticketFrozenTime", 1), int64(5))
	assert.Equal(t, MGInt("mver.consensus.ticket.ticketFrozenTime", 10), int64(5))
	assert.Equal(t, MGInt("mver.consensus.ticket.ticketFrozenTime", 20), int64(5))
}

var chainBaseParam *ChainParam
var chainV3Param *ChainParam

func initChainBase() {
	chainBaseParam = &ChainParam{}
	chainBaseParam.PowLimitBits = uint32(0x1f00ffff)
	chainBaseParam.MaxTxNumber = 1600 //160

}

func getP(height int64) *ChainParam {
	initChainBase()
	initChainBityuanV3()
	if IsFork(height, "ForkChainParamV1") {
		return chainV3Param
	}
	return chainBaseParam
}

func initChainBityuanV3() {
	chainV3Param = &ChainParam{}
	tmp := *chainBaseParam
	//copy base param
	chainV3Param = &tmp
	//修改的值

	chainV3Param.MaxTxNumber = 1500

}

func TestInitChainParam(t *testing.T) {
	cfg, _ := InitCfg("../cmd/chain33/chain33.toml")
	Init(cfg.Title, cfg)
	forkid := GetFork("ForkChainParamV1")
	assert.Equal(t, GetP(0), getP(0))
	assert.Equal(t, GetP(forkid-1), getP(forkid-1))
	assert.Equal(t, GetP(forkid), getP(forkid))
	assert.Equal(t, GetP(forkid+1), getP(forkid+1))
	assert.Equal(t, GetFundAddr(), "1BQXS6TxaYYG5mADaWij4AxhZZUTpw95a5")

	conf := ConfSub("manage")
	assert.Equal(t, GetFundAddr(), "1BQXS6TxaYYG5mADaWij4AxhZZUTpw95a5")
	assert.Equal(t, conf.GStrList("superManager"), []string{
		"1Bsg9j6gW83sShoee1fZAt9TkUjcrCgA9S",
		"12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv",
		"1Q8hGLfoGe63efeWa8fJ4Pnukhkngt6poK",
	})
}
