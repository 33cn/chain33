// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/33cn/chain33/system/crypto/secp256k1"

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
	assert.Equal(t, int64(1e9), cfg.GetMaxTxFee(0))
	assert.Equal(t, cfg.GetMaxTxFeeRate(), int64(1e7))
	assert.Equal(t, cfg.GetMinTxFeeRate(), int64(1e5))
	height, ok := cfg.GetModuleConfig().Crypto.EnableHeight[secp256k1.Name]
	assert.True(t, ok)
	_, ok = cfg.GetSubConfig().Crypto[secp256k1.Name]
	assert.True(t, ok)
	assert.Equal(t, int64(0), height)
}

// 测试实际的配置文件
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
	cfg.DisableCheckFork(true)
	cfg.chain33CfgInit(cfg.GetModuleConfig())
	mcfg := cfg.GetModuleConfig()
	assert.Equal(t, cfg.forks.forks["ForkV16Withdraw"], int64(480000))
	assert.Equal(t, mcfg.Fork.Sub["token"]["Enable"], int64(100899))
	confsystem := Conf(cfg, "config.fork.system")
	assert.Equal(t, confsystem.GInt("ForkV16Withdraw"), int64(480000))
	confsubtoken := Conf(cfg, "config.fork.sub.token")
	assert.Equal(t, confsubtoken.GInt("Enable"), int64(100899))
	// tx fee config
	assert.Equal(t, int64(1e8), cfg.GetMaxTxFee(0))
	assert.Equal(t, int64(1e6), cfg.GetMaxTxFeeRate())
	assert.Equal(t, int64(1e5), cfg.GetMinTxFeeRate())
	cfg.SetTxFeeConfig(1e9, 1e9, 1e9)
	assert.True(t, int64(1e9) == cfg.GetMinTxFeeRate() && cfg.GetMaxTxFeeRate() == cfg.GetMaxTxFee(0))
	//分叉高度30839600
	assert.True(t, 5000000000 == cfg.GetMaxTxFee(30839600))
	//分叉高度30939600
	assert.True(t, 2000000000 == cfg.GetMaxTxFee(30939600))
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

func TestCheckPrecision(t *testing.T) {
	a := int64(11)
	r := checkPrecision(a)
	assert.Equal(t, false, r)

	a = 10
	r = checkPrecision(a)
	assert.Equal(t, true, r)

	a = 1
	r = checkPrecision(a)
	assert.Equal(t, true, r)

	a = 1e8
	r = checkPrecision(a)
	assert.Equal(t, true, r)

	//大于1e8也允许
	a = 1000000000
	r = checkPrecision(a)
	assert.Equal(t, true, r)

	a = 110000000
	r = checkPrecision(a)
	assert.Equal(t, false, r)

}

func TestAddressConfig(t *testing.T) {

	cfg := NewChain33Config(GetDefaultCfgstring())
	addrConfig := cfg.GetModuleConfig().Address
	assert.Equal(t, "btc", addrConfig.DefaultDriver)
	assert.Equal(t, int64(-2), addrConfig.EnableHeight["eth"])
}

func TestIsForward2MainChainTx(t *testing.T) {

	cfg := NewChain33Config(GetDefaultCfgstring())
	tx := &Transaction{Execer: []byte("none")}

	val := IsForward2MainChainTx(cfg, tx)
	require.False(t, val)
	cfg.SetTitleOnlyForTest("user.p.test.")
	val = IsForward2MainChainTx(cfg, tx)
	require.True(t, val)
	tx.Execer = []byte("user.p.test1.none")
	val = IsForward2MainChainTx(cfg, tx)
	require.True(t, val)
	tx.Execer = []byte("user.p.test.none")
	val = IsForward2MainChainTx(cfg, tx)
	require.False(t, val)
	cfg.GetModuleConfig().RPC.ParaChain.ForwardExecs = []string{"none"}
	val = IsForward2MainChainTx(cfg, tx)
	require.True(t, val)
	cfg.GetModuleConfig().RPC.ParaChain.ForwardExecs = []string{"all"}
	val = IsForward2MainChainTx(cfg, tx)
	require.True(t, val)
	cfg.GetModuleConfig().RPC.ParaChain.ForwardExecs = []string{"coins"}
	tx.Execer = []byte("user.p.test.none")
	tx1 := &Transaction{Execer: []byte("user.p.test.paracross")}
	txs := Transactions{Txs: []*Transaction{tx, tx1}}
	val = IsForward2MainChainTx(cfg, txs.Tx())
	require.False(t, val)
	cfg.GetModuleConfig().RPC.ParaChain.ForwardExecs = []string{"none"}
	val = IsForward2MainChainTx(cfg, txs.Tx())
	require.True(t, val)

	tx.GroupCount = 2
	cfg.GetModuleConfig().RPC.ParaChain.ForwardExecs = []string{"paracross"}
	val = IsForward2MainChainTx(cfg, txs.Tx())
	require.True(t, val)

	val = IsForward2MainChainTx(cfg, tx)
	require.False(t, val)

	cfg.GetModuleConfig().RPC.ParaChain.ForwardExecs = []string{"coins"}
	hexTx := "0a05636f696e73122d18010a291080c2d72f222231387674787946483662636873536d7a396157427842543674657044727a784a32471a6d08011221034b0452139ef45a81dc90a29e90167046a30ac919ebaba6264b2d71e30f0264721a4630440220503123d7279b0ffab7a1321c9f1fa88dfd4e3247ba2a44fa171de10b5d2a9ff9022020461dfa8fe19a46796fcbb10e5aa152e8551b3921b0bbaf3f1f2d57243269ff20a08d0630be92d6bbf5ac90e8773a2231387674787946483662636873536d7a396157427842543674657044727a784a32475821"

	bTx, _ := hex.DecodeString(hexTx)
	Decode(bTx, tx)
	tx.Execer = []byte("user.p.test.coins")
	require.True(t, IsForward2MainChainTx(cfg, tx))
	cfg.GetModuleConfig().RPC.ParaChain.ForwardActionNames = []string{"withdraw"}
	require.False(t, IsForward2MainChainTx(cfg, tx))
	cfg.GetModuleConfig().RPC.ParaChain.ForwardActionNames = []string{"transfer"}
	require.True(t, IsForward2MainChainTx(cfg, tx))
}
