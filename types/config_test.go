// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChainConfig(t *testing.T) {
	cfg := NewChain33Config(GetDefaultCfgstring())
	cfg.S("a", true)
	_, err := cfg.G("b")
	assert.Equal(t, err, ErrNotFound)
	assert.True(t, cfg.IsEnable("TxHeight"))
	adata, err := cfg.G("a")
	assert.Equal(t, err, nil)
	assert.Equal(t, adata.(bool), true)

	// tx fee config
	assert.Equal(t, cfg.GetMaxTxFee(), int64(1e9))
	assert.Equal(t, cfg.GetMaxTxFeeRate(), int64(1e7))
	assert.Equal(t, cfg.GetMinTxFeeRate(), int64(1e5))
}

//测试实际的配置文件
func TestSubConfig(t *testing.T) {
	cfg, err := initSubModuleString(readFile("testdata/chain33.toml"))
	assert.Equal(t, 0, len(cfg.Consensus))
	assert.Equal(t, 2, len(cfg.Store))
	assert.Equal(t, 1, len(cfg.Exec))
	assert.Equal(t, 1, len(cfg.Wallet))
	assert.Nil(t, err)
}

func TestConfInit(t *testing.T) {
	cfg := NewChain33Config(ReadFile("testdata/chain33.toml"))
	assert.True(t, cfg.IsEnable("TxHeight"))
}

func TestConfigNoInit(t *testing.T) {
	cfg := NewChain33ConfigNoInit(ReadFile("testdata/chain33.toml"))
	assert.False(t, cfg.IsEnable("TxHeight"))
	cfg.EnableCheckFork(false)
	cfg.chain33CfgInit(cfg.GetModuleConfig())
	mcfg := cfg.GetModuleConfig()
	assert.Equal(t, cfg.forks.forks["ForkV16Withdraw"], int64(480000))
	assert.Equal(t, mcfg.Fork.Sub["token"]["Enable"], int64(100899))
	confsystem := Conf(cfg, "config.fork.system")
	assert.Equal(t, confsystem.GInt("ForkV16Withdraw"), int64(480000))
	confsubtoken := Conf(cfg, "config.fork.sub.token")
	assert.Equal(t, confsubtoken.GInt("Enable"), int64(100899))
	// tx fee config
	assert.Equal(t, int64(1e8), cfg.GetMaxTxFee())
	assert.Equal(t, int64(1e6), cfg.GetMaxTxFeeRate())
	assert.Equal(t, int64(1e4), cfg.GetMinTxFeeRate())
	cfg.SetTxFeeConfig(1e9, 1e9, 1e9)
	assert.True(t, int64(1e9) == cfg.GetMinTxFeeRate() && cfg.GetMaxTxFeeRate() == cfg.GetMaxTxFee())
}

func TestBityuanInit(t *testing.T) {
	cfg, err := initCfgString(MergeCfg(ReadFile("testdata/bityuan.toml"), ""))
	assert.Nil(t, err)
	assert.Equal(t, int64(200000), cfg.Fork.System["ForkWithdraw"])
	assert.Equal(t, int64(0), cfg.Fork.Sub["token"]["Enable"])
}

func TestGetParaExecTitleName(t *testing.T) {
	_, exist := GetParaExecTitleName("token")
	assert.Equal(t, false, exist)

	_, exist = GetParaExecTitleName("user.p.para")
	assert.Equal(t, false, exist)

	title, exist := GetParaExecTitleName("user.p.para.")
	assert.Equal(t, true, exist)
	assert.Equal(t, "user.p.para.", title)

	title, exist = GetParaExecTitleName("user.p.guodux.token")
	assert.Equal(t, true, exist)
	assert.Equal(t, "user.p.guodux.", title)
}
