// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"errors"
	"testing"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/common/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 固定测试地址（均来自仓库既有 fixture）
const (
	testBlockedBtcAddr = "14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"
	testBlockedEthAddr = "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0"
	testNormalBtcAddr  = "1JmFaA6unrCFYEWPGRi7uuXY1KthTJxJEP"
)

func withBlockedAccounts(t *testing.T, addrs []string) {
	t.Helper()
	old := blockedAccountSet
	blockedAccountSet = parseBlockedAccounts(addrs)
	t.Cleanup(func() {
		blockedAccountSet = old
	})
}

// TestChain33ConfigLoadsBlacklist 验证 toml [blacklist] 段的 accountBlacklist
// 能在 NewChain33Config 时被读入并生效
func TestChain33ConfigLoadsBlacklist(t *testing.T) {
	old := blockedAccountSet
	t.Cleanup(func() {
		blockedAccountSet = old
	})

	cfgstring := GetDefaultCfgstring() + `
[blacklist]
accountBlacklist=["` + testBlockedBtcAddr + `"]
`
	NewChain33Config(cfgstring)

	assert.True(t, IsBlockedAccount(testBlockedBtcAddr))
	assert.False(t, IsBlockedAccount(testNormalBtcAddr))
}

// TestChain33ConfigNoBlacklistSection 验证未配置 [blacklist] 段时不影响既有名单
func TestChain33ConfigNoBlacklistSection(t *testing.T) {
	withBlockedAccounts(t, []string{testBlockedBtcAddr})

	NewChain33Config(GetDefaultCfgstring())

	assert.True(t, IsBlockedAccount(testBlockedBtcAddr), "无 [blacklist] 段不应清空已有名单")
}

func TestDryRunBlockedAccounts(t *testing.T) {
	// 上线前填充真实名单后，此用例会逐条解析；当前为空名单应直接通过
	for _, addr := range blockedAccounts {
		raw, err := parseBlockedAccount(addr)
		require.NoError(t, err, "dry-run parse failed for %s", addr)
		require.Len(t, raw, 20, "dry-run raw length for %s", addr)
	}
	// 双格式样例预校验，防止解析路径回归
	cases := []string{testBlockedBtcAddr, testBlockedEthAddr}
	for _, addr := range cases {
		raw, err := parseBlockedAccount(addr)
		require.NoError(t, err, addr)
		require.Len(t, raw, 20, addr)
	}
}

func TestParseBlockedAccountFormats(t *testing.T) {
	btcRaw, err := parseBlockedAccount(testBlockedBtcAddr)
	require.NoError(t, err)
	require.Len(t, btcRaw, 20)

	ethRaw, err := parseBlockedAccount(testBlockedEthAddr)
	require.NoError(t, err)
	require.Len(t, ethRaw, 20)

	// eth 大小写不敏感（IsHexAddress 接受混合大小写）
	ethLower, err := parseBlockedAccount("0x742d35cc6634c0532925a3b844bc9e7595f0beb0")
	require.NoError(t, err)
	assert.Equal(t, ethRaw, ethLower)

	_, err = parseBlockedAccount("not-an-address")
	assert.Error(t, err)

	_, err = parseBlockedAccount("0x1234")
	assert.Error(t, err)
}

func TestIsBlockedAccount(t *testing.T) {
	withBlockedAccounts(t, []string{testBlockedBtcAddr, testBlockedEthAddr})

	assert.True(t, IsBlockedAccount(testBlockedBtcAddr))
	assert.True(t, IsBlockedAccount(testBlockedEthAddr))
	assert.False(t, IsBlockedAccount(testNormalBtcAddr))
	assert.False(t, IsBlockedAccount("not-an-address"))

	btcRaw, err := parseBlockedAccount(testBlockedBtcAddr)
	require.NoError(t, err)
	assert.True(t, IsBlockedAccountRaw(btcRaw))
	assert.False(t, IsBlockedAccountRaw([]byte{1, 2, 3}))
}

func TestCheckTxBlockedAccount(t *testing.T) {
	cfg := NewChain33Config(GetDefaultCfgstring())
	// local title 下 SetAllFork(0)，ForkAccountBlacklist 从高度 0 启用
	withBlockedAccounts(t, []string{testBlockedBtcAddr, testBlockedEthAddr})

	priv := mustLoadTestPriv(t)
	normalTo := testNormalBtcAddr

	t.Run("before fork height uses MaxHeight path", func(t *testing.T) {
		// 构造一个未启用 fork 的 cfg：直接改 forks map
		cfg2 := NewChain33Config(GetDefaultCfgstring())
		cfg2.forks.SetFork(ForkAccountBlacklist, MaxHeight)
		tx := &Transaction{Execer: []byte("coins"), To: testBlockedBtcAddr, Fee: 1e6}
		tx.Sign(SECP256K1, priv)
		assert.NoError(t, CheckTxBlockedAccount(cfg2, 0, tx))
	})

	// 门控差异：同一笔命中交易，Fork 入口在高度未达时放行，Immediate 入口始终拦截
	t.Run("immediate ignores fork gate", func(t *testing.T) {
		cfg2 := NewChain33Config(GetDefaultCfgstring())
		cfg2.forks.SetFork(ForkAccountBlacklist, MaxHeight)
		tx := &Transaction{Execer: []byte("coins"), To: testBlockedBtcAddr, Fee: 1e6}
		tx.Sign(SECP256K1, priv)

		assert.NoError(t, CheckTxBlockedAccount(cfg2, 0, tx), "fork 未达高度应放行")
		err := CheckTxBlockedAccountImmediate(tx)
		require.Error(t, err, "Immediate 不看 fork，必须拦截")
		assert.True(t, errors.Is(err, ErrBlockedAccount))
	})

	// fork 高度边界：H-1 放行，H 拦截
	t.Run("fork height boundary", func(t *testing.T) {
		const forkHeight = 100
		cfg2 := NewChain33Config(GetDefaultCfgstring())
		cfg2.forks.SetFork(ForkAccountBlacklist, forkHeight)
		tx := &Transaction{Execer: []byte("coins"), To: testBlockedBtcAddr, Fee: 1e6}
		tx.Sign(SECP256K1, priv)

		assert.NoError(t, CheckTxBlockedAccount(cfg2, forkHeight-1, tx))
		err := CheckTxBlockedAccount(cfg2, forkHeight, tx)
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrBlockedAccount))
	})

	t.Run("hit from", func(t *testing.T) {
		// 用被拉黑地址对应私钥签名，命中 from 维度
		blockedPriv := mustLoadBlockedPriv(t)
		tx := &Transaction{Execer: []byte("coins"), To: normalTo, Fee: 1e6}
		tx.Sign(SECP256K1, blockedPriv)
		require.Equal(t, testBlockedBtcAddr, tx.From())
		err := CheckTxBlockedAccount(cfg, 0, tx)
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrBlockedAccount))
	})

	t.Run("hit to", func(t *testing.T) {
		tx := &Transaction{Execer: []byte("coins"), To: testBlockedBtcAddr, Fee: 1e6}
		tx.Sign(SECP256K1, priv)
		err := CheckTxBlockedAccount(cfg, 0, tx)
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrBlockedAccount))
	})

	t.Run("normal pass", func(t *testing.T) {
		tx := &Transaction{Execer: []byte("coins"), To: normalTo, Fee: 1e6}
		tx.Sign(SECP256K1, priv)
		// from 是 priv 派生地址，不在名单；to 正常
		assert.False(t, IsBlockedAccount(tx.From()))
		assert.NoError(t, CheckTxBlockedAccount(cfg, 0, tx))
	})

	t.Run("hit evm contractAddr", func(t *testing.T) {
		action := &EVMContractAction4Chain33{
			Amount:       0,
			GasLimit:     10000,
			GasPrice:     1,
			ContractAddr: testBlockedEthAddr,
		}
		tx := &Transaction{
			Execer:  []byte("evm"),
			To:      address.ExecAddress("evm"),
			Payload: Encode(action),
			Fee:     1e6,
		}
		tx.Sign(SECP256K1, priv)
		err := CheckTxBlockedAccount(cfg, 0, tx)
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrBlockedAccount))
	})

	t.Run("hit evm para 20 bytes", func(t *testing.T) {
		raw, err := parseBlockedAccount(testBlockedEthAddr)
		require.NoError(t, err)
		action := &EVMContractAction4Chain33{
			Amount:       1,
			GasLimit:     10000,
			GasPrice:     1,
			Para:         raw,
			ContractAddr: address.ExecAddress("evm"),
		}
		tx := &Transaction{
			Execer:  []byte("evm"),
			To:      address.ExecAddress("evm"),
			Payload: Encode(action),
			Fee:     1e6,
		}
		tx.Sign(SECP256K1, priv)
		err = CheckTxBlockedAccount(cfg, 0, tx)
		require.Error(t, err)
		assert.True(t, errors.Is(err, ErrBlockedAccount))
	})

	t.Run("nil cfg", func(t *testing.T) {
		assert.NoError(t, CheckTxBlockedAccount(nil, 0, &Transaction{To: testBlockedBtcAddr}))
	})
}

func mustLoadBlockedPriv(t *testing.T) crypto.PrivKey {
	t.Helper()
	// TestPrivkeyList[1]，派生地址 14KEKbYtKKQm4wMthSK9J4La4nAiidGozt（testBlockedBtcAddr）
	cr, err := crypto.Load(GetSignName("", SECP256K1), -1)
	require.NoError(t, err)
	bkey, err := common.FromHex("CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944")
	require.NoError(t, err)
	priv, err := cr.PrivKeyFromBytes(bkey)
	require.NoError(t, err)
	return priv
}

// TestCheckTxsBlockedAccount 交易组便利函数：任一笔命中整组 error，全通过返回 nil
func TestCheckTxsBlockedAccount(t *testing.T) {
	cfg := NewChain33Config(GetDefaultCfgstring())
	withBlockedAccounts(t, []string{testBlockedBtcAddr})
	priv := mustLoadTestPriv(t)

	mkTx := func(to string) *Transaction {
		tx := &Transaction{Execer: []byte("coins"), To: to, Fee: 1e6}
		tx.Sign(SECP256K1, priv)
		return tx
	}

	// 任一笔 to 命中 -> error
	blocked := []*Transaction{mkTx(testNormalBtcAddr), mkTx(testBlockedBtcAddr)}
	err := CheckTxsBlockedAccount(cfg, 0, blocked)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBlockedAccount))

	// 全部正常 -> nil
	ok := []*Transaction{mkTx(testNormalBtcAddr), mkTx(testNormalBtcAddr)}
	assert.NoError(t, CheckTxsBlockedAccount(cfg, 0, ok))

	// Immediate 变体同样工作
	err = CheckTxsBlockedAccountImmediate(blocked)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBlockedAccount))
	assert.NoError(t, CheckTxsBlockedAccountImmediate(ok))
}

func TestParseBlockedAccountsPanic(t *testing.T) {
	assert.Panics(t, func() {
		parseBlockedAccounts([]string{"bad-addr"})
	})
}

func mustLoadTestPriv(t *testing.T) crypto.PrivKey {
	t.Helper()
	// TestPrivkeyList[0]，派生地址 12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv，不在黑名单
	cr, err := crypto.Load(GetSignName("", SECP256K1), -1)
	require.NoError(t, err)
	bkey, err := common.FromHex("4257D8692EF7FE13C68B65D6A52F03933DB2FA5CE8FAF210B5B8B80C721CED01")
	require.NoError(t, err)
	priv, err := cr.PrivKeyFromBytes(bkey)
	require.NoError(t, err)
	return priv
}
