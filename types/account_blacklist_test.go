// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"errors"
	"strings"
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

// blacklistForkV2 模拟「代码里新增了第二版名单分叉」，测试通过预注册分叉来等价 RegisterSystemFork
const blacklistForkV2 = ForkAccountBlacklist + "V2"

// defaultBlacklistSection 默认配置自带的黑名单基线段，用例自行拼装名单前需先摘掉，
// 否则同一个 toml 表被定义两次会直接解析失败
const defaultBlacklistSection = "[mver.blacklist]\naccountBlacklist=[]\n"

// newBlacklistCfg 构造一个非 local 标题的配置，使其走完整的 initForkConfig 与校验分支。
// forkSection / mverSection 直接拼进 toml，便于逐个用例定制分叉高度与名单；
// extraForks 在 chain33CfgInit 之前注册，等价于在 RegisterSystemFork 中新增分叉。
func newBlacklistCfg(forkSection, mverSection string, extraForks ...string) *Chain33Config {
	cfgstring := strings.Replace(GetDefaultCfgstring(), `Title="local"`, `Title="chain33"`, 1)
	cfgstring = strings.Replace(cfgstring, defaultBlacklistSection, "", 1)
	cfgstring += "\n" + mverSection + "\n[fork.system]\n" + forkSection + "\n"
	cfg := NewChain33ConfigNoInit(cfgstring)
	for _, fork := range extraForks {
		cfg.forks.SetFork(fork, MaxHeight)
	}
	cfg.DisableCheckFork(true)
	cfg.chain33CfgInit(cfg.GetModuleConfig())
	return cfg
}

func TestDryRunBlockedAccounts(t *testing.T) {
	// 上线前填充硬编码兜底名单后，此用例会逐条解析；当前为空名单应直接通过
	for _, addr := range blockedAccounts {
		raw, err := parseBlockedAccount(addr)
		require.NoError(t, err, "dry-run parse failed for %s", addr)
		require.Len(t, raw, 20, "dry-run raw length for %s", addr)
	}
	// 双格式样例预校验，防止解析路径回归
	for _, addr := range []string{testBlockedBtcAddr, testBlockedEthAddr} {
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

func TestParseBlockedAccountsPanic(t *testing.T) {
	assert.Panics(t, func() {
		parseBlockedAccounts("test", []string{"bad-addr"})
	})
}

// TestBlacklistVersionEvolution 名单按高度多版本演进：
// H1 之前为空，[H1,H2) 用 V1 名单，>=H2 用 V2 名单（全量，可增可删）
func TestBlacklistVersionEvolution(t *testing.T) {
	const (
		h1 = 100
		h2 = 200
	)
	cfg := newBlacklistCfg(
		"ForkAccountBlacklist=100\nForkAccountBlacklistV2=200\n",
		"[mver.blacklist]\naccountBlacklist=[]\n"+
			"[mver.blacklist.ForkAccountBlacklist]\naccountBlacklist=[\""+testBlockedBtcAddr+"\"]\n"+
			"[mver.blacklist.ForkAccountBlacklistV2]\naccountBlacklist=[\""+testBlockedEthAddr+"\"]\n",
		blacklistForkV2)

	cases := []struct {
		name       string
		height     int64
		btcBlocked bool
		ethBlocked bool
	}{
		{"创世高度无名单", 0, false, false},
		{"V1 生效前一块", h1 - 1, false, false},
		{"V1 生效", h1, true, false},
		{"V1 区间内", h2 - 1, true, false},
		{"V2 生效，V1 中被删除的地址放行", h2, false, true},
		{"V2 之后", h2 + 1000, false, true},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, c.btcBlocked, cfg.IsBlockedAccount(testBlockedBtcAddr, c.height))
			assert.Equal(t, c.ethBlocked, cfg.IsBlockedAccount(testBlockedEthAddr, c.height))
			assert.False(t, cfg.IsBlockedAccount(testNormalBtcAddr, c.height))
		})
	}
}

// TestBlacklistImmediateMatchesConsensus mempool 入口与共识层必须同高度同结论，
// 尤其是 V2 已写进配置但高度未到达时，不得提前按 V2 拦截
func TestBlacklistImmediateMatchesConsensus(t *testing.T) {
	const h2 = 1000000
	cfg := newBlacklistCfg(
		"ForkAccountBlacklist=-1\nForkAccountBlacklistV2=1000000\n",
		"[mver.blacklist]\naccountBlacklist=[]\n"+
			"[mver.blacklist.ForkAccountBlacklistV2]\naccountBlacklist=[\""+testBlockedBtcAddr+"\"]\n",
		blacklistForkV2)

	priv := mustLoadTestPriv(t)
	mkTx := func() *Transaction {
		tx := &Transaction{Execer: []byte("coins"), To: testBlockedBtcAddr, Fee: 1e6}
		tx.Sign(SECP256K1, priv)
		return tx
	}

	for _, height := range []int64{0, 1, h2 - 1, h2, h2 + 1} {
		consensusErr := CheckTxBlockedAccount(cfg, height, mkTx())
		immediateErr := CheckTxBlockedAccountImmediate(cfg, height, mkTx())
		assert.Equal(t, consensusErr == nil, immediateErr == nil, "height %d 两个入口结论必须一致", height)
		if height >= h2 {
			require.Error(t, consensusErr, "height %d 应命中 V2", height)
			assert.True(t, errors.Is(consensusErr, ErrBlockedAccount))
		} else {
			assert.NoError(t, consensusErr, "height %d 未到 V2 高度不得拦截", height)
		}
	}
}

// TestBlacklistMigrationEquivalence 现网迁移等价性：
// [mver.blacklist.ForkAccountBlacklist] 的地址集与旧 [blacklist] 一致时，H1 前后行为与旧逻辑相同
func TestBlacklistMigrationEquivalence(t *testing.T) {
	const h1 = 46561600
	cfg := newBlacklistCfg("ForkAccountBlacklist=46561600\n",
		"[mver.blacklist]\naccountBlacklist=[]\n"+
			"[mver.blacklist.ForkAccountBlacklist]\naccountBlacklist=[\""+testBlockedBtcAddr+"\"]\n")

	priv := mustLoadTestPriv(t)
	mkTx := func() *Transaction {
		tx := &Transaction{Execer: []byte("coins"), To: testBlockedBtcAddr, Fee: 1e6}
		tx.Sign(SECP256K1, priv)
		return tx
	}
	assert.NoError(t, CheckTxBlockedAccount(cfg, h1-1, mkTx()), "H1 之前放行，与旧 IsFork 门控一致")
	require.Error(t, CheckTxBlockedAccount(cfg, h1, mkTx()), "H1 起拦截")
	require.Error(t, CheckTxBlockedAccount(cfg, h1+1, mkTx()))
}

// TestBlacklistBaseSectionNotGated 钉死一个迁移陷阱：
// base 段 [mver.blacklist] 不受任何分叉门控，即使分叉配成 -1 也自创世高度即生效。
// 迁移旧 [blacklist] 时名单必须落到 [mver.blacklist.ForkAccountBlacklist]，
// 若误放进 base 段，历史区块回放会用新名单判定旧区块，直接分叉。
func TestBlacklistBaseSectionNotGated(t *testing.T) {
	cfg := newBlacklistCfg("ForkAccountBlacklist=-1\n",
		"[mver.blacklist]\naccountBlacklist=[\""+testBlockedBtcAddr+"\"]\n")

	for _, height := range []int64{0, 1, 46561599, 46561600} {
		assert.True(t, cfg.IsBlockedAccount(testBlockedBtcAddr, height),
			"base 段自高度 0 生效，height %d 也应命中", height)
	}

	// 对照：同一份名单放在分叉段并配 -1（永不启用）时，任何高度都不生效
	gated := newBlacklistCfg("ForkAccountBlacklist=-1\n",
		"[mver.blacklist]\naccountBlacklist=[]\n"+
			"[mver.blacklist.ForkAccountBlacklist]\naccountBlacklist=[\""+testBlockedBtcAddr+"\"]\n")
	for _, height := range []int64{0, 1, 46561599, 46561600} {
		assert.False(t, gated.IsBlockedAccount(testBlockedBtcAddr, height),
			"分叉段配 -1 等同未启用，height %d 不得命中", height)
	}
}

// bityuanBlockedAddrs 现网 bityuan.go 中 [blacklist] 的实际名单，用于迁移前后逐地址对照
var bityuanBlockedAddrs = []string{
	"0x36086e9f01a934f36910b45aaabfc1256ee8cb66",
	"0x2bacf52028b388f004d54958eb1cad8e3fcac263",
	"0xa1d1e29cd8de11821a31467524282f13deda2976",
	"0xf1641331e82a1b3e27b81edbdbf7c0750f7ae366",
	"0xd57d5cf08e6b82191beeb48dff3215b0492b3892",
	"0xd51d08093b8a2df658ca22f3b9145ff63fbeb62c",
	"0xba7ebf059a332468b0fe98992ff14fabed199072",
	"0x125cae868427ec5d791304ca165b040e84506737",
}

// TestBlacklistBityuanMigration 现网 bityuan 配置的迁移对照：
// [blacklist] 8 个地址 + ForkAccountBlacklist=46561600 迁到 [mver.blacklist.ForkAccountBlacklist] 后，
// 每个地址在分叉高度前后的判定必须与旧 IsFork 门控逐一致。
func TestBlacklistBityuanMigration(t *testing.T) {
	const h1 = 46561600
	quoted := make([]string, 0, len(bityuanBlockedAddrs))
	for _, addr := range bityuanBlockedAddrs {
		quoted = append(quoted, "\""+addr+"\"")
	}
	list := "[" + strings.Join(quoted, ",") + "]"
	forkSection := "[mver.blacklist.ForkAccountBlacklist]\naccountBlacklist=" + list + "\n"
	cfg := newBlacklistCfg("ForkAccountBlacklist=46561600\n",
		"[mver.blacklist]\naccountBlacklist=[]\n"+forkSection)

	priv := mustLoadTestPriv(t)
	mkTx := func(to string) *Transaction {
		tx := &Transaction{Execer: []byte("coins"), To: to, Fee: 1e6}
		tx.Sign(SECP256K1, priv)
		return tx
	}
	for _, addr := range bityuanBlockedAddrs {
		addr := addr
		t.Run(addr, func(t *testing.T) {
			// 分叉高度之前：旧代码 IsFork 为假直接放行，新代码取到空的 base 名单，同样放行
			assert.False(t, cfg.IsBlockedAccount(addr, h1-1))
			assert.NoError(t, CheckTxBlockedAccount(cfg, h1-1, mkTx(addr)))
			// 分叉高度起：两边都用同一份 8 地址名单拦截
			for _, height := range []int64{h1, h1 + 1, h1 + 1000000} {
				assert.True(t, cfg.IsBlockedAccount(addr, height), "height %d", height)
				assert.Error(t, CheckTxBlockedAccount(cfg, height, mkTx(addr)), "height %d", height)
			}
		})
	}

	// 省略 [mver.blacklist] 基线段时，base 名单为空，判定结果必须完全一致（基线段是可选的）
	t.Run("无基线段", func(t *testing.T) {
		noBase := newBlacklistCfg("ForkAccountBlacklist=46561600\n", forkSection)
		for _, addr := range bityuanBlockedAddrs {
			assert.False(t, noBase.IsBlockedAccount(addr, h1-1), addr)
			assert.True(t, noBase.IsBlockedAccount(addr, h1), addr)
		}
	})
}

// TestBlacklistBityuanMigrationTypo 迁移时把键名写成 aaccountBlacklist 之类的笔误必须启动即失败，
// 否则该版本名单不会进入 versionList，节点会静默按空名单放行攻击地址
func TestBlacklistBityuanMigrationTypo(t *testing.T) {
	assert.Panics(t, func() {
		newBlacklistCfg("ForkAccountBlacklist=46561600\n",
			"[mver.blacklist]\naccountBlacklist=[]\n"+
				"[mver.blacklist.ForkAccountBlacklist]\naaccountBlacklist=[\""+bityuanBlockedAddrs[0]+"\"]\n")
	})
}

// TestBlacklistEmptyConfigNoOp 空名单等价性回归：
// 默认未启用配置下，任何高度、任何入口都必须放行，保证迁移上线不改变共识判定
func TestBlacklistEmptyConfigNoOp(t *testing.T) {
	cfg := NewChain33Config(GetDefaultCfgstring())
	priv := mustLoadTestPriv(t)
	blockedPriv := mustLoadBlockedPriv(t)

	for _, height := range []int64{0, 1, 100, 46561600, MaxHeight - 1} {
		tx := &Transaction{Execer: []byte("coins"), To: testBlockedBtcAddr, Fee: 1e6}
		tx.Sign(SECP256K1, priv)
		assert.NoError(t, CheckTxBlockedAccount(cfg, height, tx), "height %d", height)
		assert.NoError(t, CheckTxBlockedAccountImmediate(cfg, height, tx), "height %d", height)

		tx2 := &Transaction{Execer: []byte("coins"), To: testNormalBtcAddr, Fee: 1e6}
		tx2.Sign(SECP256K1, blockedPriv)
		assert.NoError(t, CheckTxBlockedAccount(cfg, height, tx2), "height %d", height)
		assert.False(t, cfg.IsBlockedAccount(testBlockedBtcAddr, height))
	}
}

// TestBlacklistLocalConfigInit 反向用例：默认 local 配置（SetAllFork(0) 且无 mver 子段）必须能正常初始化
func TestBlacklistLocalConfigInit(t *testing.T) {
	assert.NotPanics(t, func() {
		cfg := NewChain33Config(GetDefaultCfgstring())
		require.NotNil(t, cfg.blacklistAt(0), "base 版本必须存在，查表不得返回 nil")
		assert.Empty(t, cfg.blacklistAt(0).set)
	})
}

// TestBlacklistConfigPanic 配置写错时必须启动失败，不能静默沿用旧名单
func TestBlacklistConfigPanic(t *testing.T) {
	t.Run("引用未注册的分叉", func(t *testing.T) {
		assert.Panics(t, func() {
			newBlacklistCfg("ForkAccountBlacklist=-1\n",
				"[mver.blacklist.ForkAccountBlacklistNotExist]\naccountBlacklist=[]\n")
		})
	})

	t.Run("分叉已启用但缺少 mver 段", func(t *testing.T) {
		assert.Panics(t, func() {
			newBlacklistCfg("ForkAccountBlacklist=100\n", "[mver.blacklist]\naccountBlacklist=[]\n")
		})
	})

	t.Run("段内键名拼错", func(t *testing.T) {
		assert.Panics(t, func() {
			newBlacklistCfg("ForkAccountBlacklist=-1\n", "[mver.blacklist]\naccountBlacklists=[]\n")
		})
	})

	t.Run("子段内键名拼错", func(t *testing.T) {
		assert.Panics(t, func() {
			newBlacklistCfg("ForkAccountBlacklist=100\n",
				"[mver.blacklist.ForkAccountBlacklist]\naccounts=[]\n")
		})
	})

	t.Run("地址无法解析", func(t *testing.T) {
		assert.Panics(t, func() {
			newBlacklistCfg("ForkAccountBlacklist=100\n",
				"[mver.blacklist.ForkAccountBlacklist]\naccountBlacklist=[\"bad-addr\"]\n")
		})
	})

	t.Run("残留静态 blacklist 段", func(t *testing.T) {
		assert.Panics(t, func() {
			newBlacklistCfg("ForkAccountBlacklist=-1\n",
				"[blacklist]\naccountBlacklist=[\""+testBlockedBtcAddr+"\"]\n")
		})
	})
}

func TestCheckTxBlockedAccount(t *testing.T) {
	cfg := NewChain33Config(GetDefaultCfgstring())
	restore := cfg.SetBlockedAccountsForTest(0, []string{testBlockedBtcAddr, testBlockedEthAddr})
	t.Cleanup(restore)

	priv := mustLoadTestPriv(t)

	t.Run("fork height boundary", func(t *testing.T) {
		const forkHeight = 100
		cfg2 := NewChain33Config(GetDefaultCfgstring())
		defer cfg2.SetBlockedAccountsForTest(forkHeight, []string{testBlockedBtcAddr})()
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
		tx := &Transaction{Execer: []byte("coins"), To: testNormalBtcAddr, Fee: 1e6}
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
		tx := &Transaction{Execer: []byte("coins"), To: testNormalBtcAddr, Fee: 1e6}
		tx.Sign(SECP256K1, priv)
		// from 是 priv 派生地址，不在名单；to 正常
		assert.False(t, cfg.IsBlockedAccount(tx.From(), 0))
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

		assert.True(t, cfg.IsBlockedAccountRaw(raw, 0))
		assert.False(t, cfg.IsBlockedAccountRaw([]byte{1, 2, 3}, 0))
	})

	t.Run("nil cfg", func(t *testing.T) {
		assert.NoError(t, CheckTxBlockedAccount(nil, 0, &Transaction{To: testBlockedBtcAddr}))
		assert.NoError(t, CheckTxsBlockedAccount(nil, 0, []*Transaction{{To: testBlockedBtcAddr}}))
		assert.False(t, (*Chain33Config)(nil).IsBlockedAccount(testBlockedBtcAddr, 0))
	})

	t.Run("快照未构建", func(t *testing.T) {
		cfg3 := NewChain33ConfigNoInit(GetDefaultCfgstring())
		assert.Nil(t, cfg3.blacklistAt(0))
		assert.NoError(t, CheckTxBlockedAccount(cfg3, 0, &Transaction{To: testBlockedBtcAddr}))
	})
}

// TestCheckTxsBlockedAccount 交易组便利函数：任一笔命中整组 error，全通过返回 nil
func TestCheckTxsBlockedAccount(t *testing.T) {
	cfg := NewChain33Config(GetDefaultCfgstring())
	defer cfg.SetBlockedAccountsForTest(0, []string{testBlockedBtcAddr})()
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
	err = CheckTxsBlockedAccountImmediate(cfg, 0, blocked)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBlockedAccount))
	assert.NoError(t, CheckTxsBlockedAccountImmediate(cfg, 0, ok))
}

func mustLoadBlockedPriv(t *testing.T) crypto.PrivKey {
	t.Helper()
	// TestPrivkeyList[1]，派生地址 14KEKbYtKKQm4wMthSK9J4La4nAiidGozt（testBlockedBtcAddr）
	return mustLoadPriv(t, "CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944")
}

func mustLoadTestPriv(t *testing.T) crypto.PrivKey {
	t.Helper()
	// TestPrivkeyList[0]，派生地址 12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv，不在黑名单
	return mustLoadPriv(t, "4257D8692EF7FE13C68B65D6A52F03933DB2FA5CE8FAF210B5B8B80C721CED01")
}

func mustLoadPriv(t *testing.T, hexKey string) crypto.PrivKey {
	t.Helper()
	cr, err := crypto.Load(GetSignName("", SECP256K1), -1)
	require.NoError(t, err)
	bkey, err := common.FromHex(hexKey)
	require.NoError(t, err)
	priv, err := cr.PrivKeyFromBytes(bkey)
	require.NoError(t, err)
	return priv
}
