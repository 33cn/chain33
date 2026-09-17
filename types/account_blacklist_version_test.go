// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBlacklistVersionEvolutionV1V2V3 三版本全量演进：
// V1=btc，V2=btc+eth（追加），V3=eth+v3（删 btc、留 eth、加 v3）。
// 每一份都是全量名单；跨过的旧版本高度上的判定不得被后续版本改写。
func TestBlacklistVersionEvolutionV1V2V3(t *testing.T) {
	const (
		h1 = 100
		h2 = 200
		h3 = 300
	)
	cfg := newBlacklistCfg(
		"ForkAccountBlacklist=100\nForkAccountBlacklistV2=200\nForkAccountBlacklistV3=300\n",
		blacklistMverSection("", nil)+
			blacklistMverSection(ForkAccountBlacklist, []string{testBlockedBtcAddr})+
			blacklistMverSection(ForkAccountBlacklistV2, []string{testBlockedBtcAddr, testBlockedEthAddr})+
			blacklistMverSection(blacklistForkV3, []string{testBlockedEthAddr, testV3EthAddr}),
		blacklistForkV3)

	cases := []struct {
		name       string
		height     int64
		btcBlocked bool
		ethBlocked bool
		v3Blocked  bool
	}{
		{"创世", 0, false, false, false},
		{"V1 前一块", h1 - 1, false, false, false},
		{"V1 生效", h1, true, false, false},
		{"V1 区间", h2 - 1, true, false, false},
		{"V2 生效，追加 eth，btc 仍拦", h2, true, true, false},
		{"V2 区间", h3 - 1, true, true, false},
		{"V3 生效，btc 放行，eth 仍拦，v3 新拦", h3, false, true, true},
		{"V3 之后", h3 + 1000, false, true, true},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, c.btcBlocked, cfg.IsBlockedAccount(testBlockedBtcAddr, c.height),
				"height=%d addr=%s", c.height, testBlockedBtcAddr)
			assert.Equal(t, c.ethBlocked, cfg.IsBlockedAccount(testBlockedEthAddr, c.height),
				"height=%d addr=%s", c.height, testBlockedEthAddr)
			assert.Equal(t, c.v3Blocked, cfg.IsBlockedAccount(testV3EthAddr, c.height),
				"height=%d addr=%s", c.height, testV3EthAddr)
			assert.False(t, cfg.IsBlockedAccount(testNormalBtcAddr, c.height),
				"height=%d 正常地址不得命中", c.height)
		})
	}
}

// TestBlacklistV2UnbanThenV3Restore V2 写空全量名单模拟放行，V3 把 V1 全部地址加回。
// V1 区间必须仍拦截（不能改历史），[H2,H3) 全部放行，>=H3 再全部拦回来。
func TestBlacklistV2UnbanThenV3Restore(t *testing.T) {
	const (
		h1 = 100
		h2 = 200
		h3 = 300
	)
	v1List := append([]string{testBlockedBtcAddr, testBlockedEthAddr}, bityuanBlockedAddrs...)
	cfg := newBlacklistCfg(
		"ForkAccountBlacklist=100\nForkAccountBlacklistV2=200\nForkAccountBlacklistV3=300\n",
		blacklistMverSection("", nil)+
			blacklistMverSection(ForkAccountBlacklist, v1List)+
			blacklistMverSection(ForkAccountBlacklistV2, nil)+
			blacklistMverSection(blacklistForkV3, v1List),
		blacklistForkV3)

	priv := mustLoadTestPriv(t)
	mkTx := func(to string) *Transaction {
		tx := &Transaction{Execer: []byte("coins"), To: to, Fee: 1e6}
		tx.Sign(SECP256K1, priv)
		return tx
	}
	assertAll := func(t *testing.T, height int64, wantBlocked bool) {
		t.Helper()
		for _, addr := range v1List {
			assert.Equal(t, wantBlocked, cfg.IsBlockedAccount(addr, height),
				"height=%d addr=%s wantBlocked=%v", height, addr, wantBlocked)
			err := CheckTxBlockedAccount(cfg, height, mkTx(addr))
			imm := CheckTxBlockedAccountImmediate(cfg, height, mkTx(addr))
			assert.Equal(t, err == nil, imm == nil, "height=%d addr=%s 两个入口必须一致", height, addr)
			if wantBlocked {
				require.Error(t, err, "height=%d addr=%s 应拦截", height, addr)
				assert.True(t, errors.Is(err, ErrBlockedAccount), "height=%d addr=%s err=%v", height, addr, err)
				continue
			}
			assert.NoError(t, err, "height=%d addr=%s V2 放行窗口不得拦截", height, addr)
		}
		assert.False(t, cfg.IsBlockedAccount(testNormalBtcAddr, height),
			"height=%d 从未入名单的地址不得命中", height)
	}

	assertAll(t, h1-1, false)
	assertAll(t, h1, true)
	assertAll(t, h2-1, true)
	assertAll(t, h2, false)
	assertAll(t, h3-1, false)
	assertAll(t, h3, true)
	assertAll(t, h3+1000, true)
}

// TestBlacklistCheckTxBoundariesV1V2V3 共识拦截必须跟选版高度对齐，不能提前按未来版本拦
func TestBlacklistCheckTxBoundariesV1V2V3(t *testing.T) {
	const (
		h1 = 100
		h2 = 200
		h3 = 300
	)
	cfg := newBlacklistCfg(
		"ForkAccountBlacklist=100\nForkAccountBlacklistV2=200\nForkAccountBlacklistV3=300\n",
		blacklistMverSection("", nil)+
			blacklistMverSection(ForkAccountBlacklist, []string{testBlockedBtcAddr})+
			blacklistMverSection(ForkAccountBlacklistV2, []string{testBlockedBtcAddr, testBlockedEthAddr})+
			blacklistMverSection(blacklistForkV3, []string{testBlockedEthAddr, testV3EthAddr}),
		blacklistForkV3)

	priv := mustLoadTestPriv(t)
	mkTx := func(to string) *Transaction {
		tx := &Transaction{Execer: []byte("coins"), To: to, Fee: 1e6}
		tx.Sign(SECP256K1, priv)
		return tx
	}
	assertHit := func(t *testing.T, height int64, to string, wantHit bool) {
		t.Helper()
		err := CheckTxBlockedAccount(cfg, height, mkTx(to))
		imm := CheckTxBlockedAccountImmediate(cfg, height, mkTx(to))
		assert.Equal(t, err == nil, imm == nil, "height=%d to=%s 两个入口必须一致", height, to)
		if wantHit {
			require.Error(t, err, "height=%d to=%s 应拦截", height, to)
			assert.True(t, errors.Is(err, ErrBlockedAccount), "height=%d to=%s err=%v", height, to, err)
			return
		}
		assert.NoError(t, err, "height=%d to=%s 不得拦截", height, to)
	}

	assertHit(t, h2-1, testBlockedEthAddr, false)
	assertHit(t, h2, testBlockedEthAddr, true)
	assertHit(t, h3-1, testBlockedBtcAddr, true)
	assertHit(t, h3, testBlockedBtcAddr, false)
	assertHit(t, h3-1, testV3EthAddr, false)
	assertHit(t, h3, testV3EthAddr, true)

	// 交易组：V3 高度上第二笔命中 v3 地址，整组拒绝
	group := []*Transaction{mkTx(testNormalBtcAddr), mkTx(testV3EthAddr)}
	err := CheckTxsBlockedAccount(cfg, h3, group)
	require.Error(t, err, "V3 高度交易组应命中")
	assert.True(t, errors.Is(err, ErrBlockedAccount))
	assert.NoError(t, CheckTxsBlockedAccount(cfg, h3-1, group), "未到 V3 不得按 V3 名单整组拦截")
}

// TestBlacklistV3ClosedKeepsV2 V3 已写 mver 但高度为 -1 时永不启用，跨过任意高度仍用 V2 全量名单
func TestBlacklistV3ClosedKeepsV2(t *testing.T) {
	const h2 = 200
	cfg := newBlacklistCfg(
		"ForkAccountBlacklist=100\nForkAccountBlacklistV2=200\nForkAccountBlacklistV3=-1\n",
		blacklistMverSection("", nil)+
			blacklistMverSection(ForkAccountBlacklist, []string{testBlockedBtcAddr})+
			blacklistMverSection(ForkAccountBlacklistV2, []string{testBlockedBtcAddr, testBlockedEthAddr})+
			blacklistMverSection(blacklistForkV3, []string{testV3EthAddr}),
		blacklistForkV3)

	for _, height := range []int64{h2, h2 + 1, 1e12} {
		assert.True(t, cfg.IsBlockedAccount(testBlockedBtcAddr, height), "height=%d 应沿用 V2，btc 仍拦", height)
		assert.True(t, cfg.IsBlockedAccount(testBlockedEthAddr, height), "height=%d 应沿用 V2，eth 仍拦", height)
		assert.False(t, cfg.IsBlockedAccount(testV3EthAddr, height), "height=%d V3=-1 不得提前拦 v3 地址", height)
	}
}

// TestBlacklistV3FullListNotIncrement V3 若只写新增地址（当成增量），运行时会按全量覆盖，旧地址被放行。
// 这是配置错误的可观察后果，用来钉死「必须写全量」的语义。
func TestBlacklistV3FullListNotIncrement(t *testing.T) {
	const (
		h2 = 200
		h3 = 300
	)
	cfg := newBlacklistCfg(
		"ForkAccountBlacklist=100\nForkAccountBlacklistV2=200\nForkAccountBlacklistV3=300\n",
		blacklistMverSection("", nil)+
			blacklistMverSection(ForkAccountBlacklist, []string{testBlockedBtcAddr})+
			blacklistMverSection(ForkAccountBlacklistV2, []string{testBlockedBtcAddr, testBlockedEthAddr})+
			blacklistMverSection(blacklistForkV3, []string{testV3EthAddr}),
		blacklistForkV3)

	assert.True(t, cfg.IsBlockedAccount(testBlockedBtcAddr, h2), "V2 区间 btc 必须仍拦")
	assert.True(t, cfg.IsBlockedAccount(testBlockedEthAddr, h2), "V2 区间 eth 必须仍拦")
	assert.False(t, cfg.IsBlockedAccount(testBlockedBtcAddr, h3),
		"V3 只写了新地址时，btc 会从 H3 起被放行，说明名单是全量覆盖不是增量")
	assert.False(t, cfg.IsBlockedAccount(testBlockedEthAddr, h3),
		"V3 只写了新地址时，eth 会从 H3 起被放行")
	assert.True(t, cfg.IsBlockedAccount(testV3EthAddr, h3))
}

// TestBlacklistBityuanAppendV3 现网追加第二轮地址：V1/V2 段原样保留，V3 写老 8 个 + V2 newbie + 再新增
func TestBlacklistBityuanAppendV3(t *testing.T) {
	const (
		h1     = 46561600
		h2     = 50000000
		h3     = 60000000
		newbie = "0x1111111111111111111111111111111111111111"
	)
	v2List := append(append([]string{}, bityuanBlockedAddrs...), newbie)
	v3List := append(append([]string{}, v2List...), testV3EthAddr)
	cfg := newBlacklistCfg(
		"ForkAccountBlacklist=46561600\nForkAccountBlacklistV2=50000000\nForkAccountBlacklistV3=60000000\n",
		blacklistMverSection("", nil)+
			blacklistMverSection(ForkAccountBlacklist, bityuanBlockedAddrs)+
			blacklistMverSection(ForkAccountBlacklistV2, v2List)+
			blacklistMverSection(blacklistForkV3, v3List),
		blacklistForkV3)

	for _, addr := range bityuanBlockedAddrs {
		assert.False(t, cfg.IsBlockedAccount(addr, h1-1), "%s H1 之前放行", addr)
		assert.True(t, cfg.IsBlockedAccount(addr, h1), "%s 自 H1 拦截", addr)
		assert.True(t, cfg.IsBlockedAccount(addr, h2), "%s 跨 H2 仍拦截", addr)
		assert.True(t, cfg.IsBlockedAccount(addr, h3), "%s 跨 H3 仍拦截", addr)
	}
	assert.False(t, cfg.IsBlockedAccount(newbie, h2-1), "newbie 不得倒查到 V1")
	assert.True(t, cfg.IsBlockedAccount(newbie, h2))
	assert.True(t, cfg.IsBlockedAccount(newbie, h3))
	assert.False(t, cfg.IsBlockedAccount(testV3EthAddr, h3-1), "v3 地址不得倒查到 V2")
	assert.True(t, cfg.IsBlockedAccount(testV3EthAddr, h3))
}

// TestBlacklistV2V3SameHeightPicksV3 同高度并列时取字母序更大的分叉名，与 mver 取舍一致
func TestBlacklistV2V3SameHeightPicksV3(t *testing.T) {
	const h = 200
	cfg := newBlacklistCfg(
		"ForkAccountBlacklist=100\nForkAccountBlacklistV2=200\nForkAccountBlacklistV3=200\n",
		blacklistMverSection("", nil)+
			blacklistMverSection(ForkAccountBlacklist, []string{testBlockedBtcAddr})+
			blacklistMverSection(ForkAccountBlacklistV2, []string{testBlockedBtcAddr})+
			blacklistMverSection(blacklistForkV3, []string{testV3EthAddr}),
		blacklistForkV3)

	assert.False(t, cfg.IsBlockedAccount(testBlockedBtcAddr, h),
		"同高度 V3 胜出且名单不含 btc 时，btc 应从 H 起放行")
	assert.True(t, cfg.IsBlockedAccount(testV3EthAddr, h), "同高度应采用 V3 名单")
	v := cfg.blacklistAt(h)
	require.NotNil(t, v)
	assert.Equal(t, blacklistForkV3, v.name, "同高度保留的版本名必须是 V3")
	assert.Equal(t, int64(h), v.height)
}

// TestBlacklistV2V3ConfigPanic V2/V3 配了非 -1 高度就必须有对应 mver 段；V3 未注册不得引用
func TestBlacklistV2V3ConfigPanic(t *testing.T) {
	t.Run("V2已启用但缺少mver段", func(t *testing.T) {
		assert.Panics(t, func() {
			newBlacklistCfg(
				"ForkAccountBlacklist=-1\nForkAccountBlacklistV2=200\n",
				blacklistMverSection("", nil),
				blacklistForkV2)
		})
	})
	t.Run("V3已启用但缺少mver段", func(t *testing.T) {
		assert.Panics(t, func() {
			newBlacklistCfg(
				"ForkAccountBlacklist=100\nForkAccountBlacklistV2=200\nForkAccountBlacklistV3=300\n",
				blacklistMverSection("", nil)+
					blacklistMverSection(ForkAccountBlacklist, []string{testBlockedBtcAddr})+
					blacklistMverSection(ForkAccountBlacklistV2, []string{testBlockedBtcAddr}),
				blacklistForkV3)
		})
	})
	t.Run("mver引用未注册的V3", func(t *testing.T) {
		assert.Panics(t, func() {
			newBlacklistCfg(
				"ForkAccountBlacklist=-1\n",
				blacklistMverSection(blacklistForkV3, []string{testV3EthAddr}))
		})
	})
	t.Run("V3键名拼错", func(t *testing.T) {
		assert.Panics(t, func() {
			newBlacklistCfg(
				"ForkAccountBlacklist=-1\nForkAccountBlacklistV3=300\n",
				"[mver.blacklist.ForkAccountBlacklistV3]\naccounts=[\""+testV3EthAddr+"\"]\n",
				blacklistForkV3)
		})
	})
}

// TestBlacklistV2ClosedDoesNotRequireMver 官方默认 V2=-1 时可以不写 mver 段，节点必须能起来
func TestBlacklistV2ClosedDoesNotRequireMver(t *testing.T) {
	assert.NotPanics(t, func() {
		cfg := newBlacklistCfg(
			"ForkAccountBlacklist=-1\nForkAccountBlacklistV2=-1\n",
			blacklistMverSection("", nil))
		assert.False(t, cfg.IsBlockedAccount(testBlockedBtcAddr, 1e12))
		assert.Equal(t, int64(MaxHeight), cfg.GetFork(ForkAccountBlacklistV2))
	})
}
