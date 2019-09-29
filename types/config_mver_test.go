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
	RegFork("store-kvmvccmavl", func(cfg *Chain33Config) {
		cfg.RegisterDappFork("store-kvmvccmavl", "ForkKvmvccmavl", MaxHeight)
	})
	cfg := NewChain33ConfigNoInit(ReadFile("testdata/local.mvertest.toml"))
	cfg.EnableCheckFork(false)
	cfg.chain33CfgInit(cfg.GetModuleConfig())
	assert.Equal(t, cfg.MGStr("mver.consensus.name2", 0), "ticket-bityuan")
	assert.Equal(t, cfg.MGStr("mver.consensus.name2", 10), "ticket-bityuanv5")
	assert.Equal(t, cfg.MGStr("mver.consensus.name2", 25), "ticket-bityuanv2")
	assert.Equal(t, cfg.MGStr("mver.hello", 0), "world")
	assert.Equal(t, cfg.MGStr("mver.hello", 11), "forkworld")
	assert.Equal(t, cfg.MGStr("mver.nofork", 0), "nofork")
	assert.Equal(t, cfg.MGStr("mver.nofork", 9), "nofork")
	assert.Equal(t, cfg.MGStr("mver.nofork", 11), "nofork")

	assert.Equal(t, cfg.MGStr("mver.exec.sub.coins.name2", -1), "ticket-bityuan")
	assert.Equal(t, cfg.MGStr("mver.exec.sub.coins.name2", 0), "ticket-bityuanv5-enable")
	assert.Equal(t, cfg.MGStr("mver.exec.sub.coins.name2", 9), "ticket-bityuanv5-enable")
	assert.Equal(t, cfg.MGStr("mver.exec.sub.coins.name2", 10), "ticket-bityuanv5")
	assert.Equal(t, cfg.MGStr("mver.exec.sub.coins.name2", 11), "ticket-bityuanv5")
	assert.Equal(t, int64(1), cfg.GetDappFork("store-kvmvccmavl", "ForkKvmvccmavl"))

	assert.Equal(t, cfg.MGInt("mver.consensus.ticket.ticketPrice", 1), int64(10000))
	assert.Equal(t, cfg.MGInt("mver.consensus.ticket.ticketPrice", 11), int64(20000))
	assert.Equal(t, cfg.MGInt("mver.consensus.ticket.ticketPrice", 21), int64(30000))

	assert.Equal(t, cfg.MGInt("mver.consensus.ticket.ticketFrozenTime", -2), int64(5))
	assert.Equal(t, cfg.MGInt("mver.consensus.ticket.ticketFrozenTime", 1), int64(5))
	assert.Equal(t, cfg.MGInt("mver.consensus.ticket.ticketFrozenTime", 10), int64(5))
	assert.Equal(t, cfg.MGInt("mver.consensus.ticket.ticketFrozenTime", 20), int64(5))
}

var chainBaseParam *ChainParam
var chainV3Param *ChainParam

func initChainBase() {
	chainBaseParam = &ChainParam{}
	chainBaseParam.PowLimitBits = uint32(0x1f00ffff)
	chainBaseParam.MaxTxNumber = 1600 //160

}

func getP(cfg *Chain33Config, height int64) *ChainParam {
	initChainBase()
	initChainBityuanV3()
	if cfg.IsFork(height, "ForkChainParamV1") {
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
	cfg := NewChain33Config(ReadFile("../cmd/chain33/chain33.toml"))
	forkid := cfg.GetFork("ForkChainParamV1")
	assert.Equal(t, cfg.GetP(0), getP(cfg, 0))
	assert.Equal(t, cfg.GetP(forkid-1), getP(cfg, forkid-1))
	assert.Equal(t, cfg.GetP(forkid), getP(cfg, forkid))
	assert.Equal(t, cfg.GetP(forkid+1), getP(cfg, forkid+1))
	assert.Equal(t, cfg.GetFundAddr(), "1BQXS6TxaYYG5mADaWij4AxhZZUTpw95a5")

	conf := ConfSub(cfg, "manage")
	assert.Equal(t, cfg.GetFundAddr(), "1BQXS6TxaYYG5mADaWij4AxhZZUTpw95a5")
	assert.Equal(t, conf.GStrList("superManager"), []string{
		"1Bsg9j6gW83sShoee1fZAt9TkUjcrCgA9S",
		"12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv",
		"1Q8hGLfoGe63efeWa8fJ4Pnukhkngt6poK",
	})
}
