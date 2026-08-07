// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"testing"

	"github.com/33cn/chain33/common/crypto"
	"github.com/stretchr/testify/assert"
)

// foreignChainID 与默认配置的 chainID 不同, 代表另一条链
const foreignChainID = int32(99)

func chainIDTestKey(t *testing.T) crypto.PrivKey {
	c, err := crypto.Load("secp256k1", -1)
	assert.Nil(t, err)
	priv, err := c.GenKey()
	assert.Nil(t, err)
	return priv
}

func newForeignTx(t *testing.T, priv crypto.PrivKey, fee int64) *Transaction {
	tx := &Transaction{
		Execer:  []byte("coins"),
		Payload: []byte("chainid-test-payload"),
		Fee:     fee,
		To:      "1J7X82xR9tJr8s2oCDgrWHu8ThULyfmzRb",
		ChainID: foreignChainID,
	}
	tx.Sign(SECP256K1, priv)
	return tx
}

// 交易组曾经可以完全绕过 chainID 校验:
// CheckWithFork 对组内每笔固定传 minfee=0, 命中 check 的提前返回。
// 攻击者可以把其他链上已合法签名的交易原样打成交易组重放到本链。
func TestTxGroupChainIDNotBypassable(t *testing.T) {
	cfg := NewChain33Config(GetDefaultCfgstring())
	cfg.forks.ReplaceFork(ForkTxChainIDStrict, 0)

	priv := chainIDTestKey(t)
	minFee := cfg.GetMinTxFeeRate()
	maxFee := cfg.GetMaxTxFee(1)

	// 基线: 单笔外链交易必须被拒绝
	single := newForeignTx(t, priv, 1e6)
	assert.Equal(t, ErrTxChainID, single.Check(cfg, 1, minFee, maxFee))
	// 签名本身是有效的, 拦截来自 chainID 而非验签
	assert.True(t, single.CheckSign(1))

	// 同样的交易打成交易组后同样必须被拒绝
	group, err := CreateTxGroup([]*Transaction{
		newForeignTx(t, priv, 0),
		newForeignTx(t, priv, 0),
	}, minFee)
	assert.Nil(t, err)
	assert.Nil(t, group.SignN(0, SECP256K1, priv))
	assert.Nil(t, group.SignN(1, SECP256K1, priv))

	assert.True(t, group.CheckSign(1))
	assert.Equal(t, ErrTxChainID, group.Check(cfg, 1, minFee, maxFee))
}

// minTxFeeRate=0 时单笔交易同样不能跳过 chainID 校验
// minTxFeeRate=0 时单笔交易也必须校验 chainID:
// minfee 的提前返回只应跳过「最低手续费」校验, 不应连带跳过 chainID。
// 联盟链通过 minfee=0 免手续费, 不等于可以跨链重放。
func TestSingleTxChainIDWithZeroMinFee(t *testing.T) {
	cfg := NewChain33Config(GetDefaultCfgstring())
	cfg.forks.ReplaceFork(ForkTxChainIDStrict, 0)

	priv := chainIDTestKey(t)
	tx := newForeignTx(t, priv, 1e6)
	maxFee := cfg.GetMaxTxFee(1)

	assert.Equal(t, ErrTxChainID, tx.Check(cfg, 1, 100000, maxFee))
	// minfee=0 同样必须被拦, 否则跨链重放可借免手续费链绕过
	assert.Equal(t, ErrTxChainID, tx.Check(cfg, 1, 0, maxFee))
}

// 本链 chainID 的交易组不受影响, 避免收紧校验误伤正常交易
func TestTxGroupSameChainIDStillValid(t *testing.T) {
	cfg := NewChain33Config(GetDefaultCfgstring())
	cfg.forks.ReplaceFork(ForkTxChainIDStrict, 0)

	priv := chainIDTestKey(t)
	minFee := cfg.GetMinTxFeeRate()
	maxFee := cfg.GetMaxTxFee(1)

	mk := func(fee int64) *Transaction {
		tx := &Transaction{
			Execer:  []byte("coins"),
			Payload: []byte("chainid-test-payload"),
			Fee:     fee,
			To:      "1J7X82xR9tJr8s2oCDgrWHu8ThULyfmzRb",
			ChainID: cfg.GetChainID(),
		}
		tx.Sign(SECP256K1, priv)
		return tx
	}

	group, err := CreateTxGroup([]*Transaction{mk(0), mk(0)}, minFee)
	assert.Nil(t, err)
	assert.Nil(t, group.SignN(0, SECP256K1, priv))
	assert.Nil(t, group.SignN(1, SECP256K1, priv))
	assert.Nil(t, group.Check(cfg, 1, minFee, maxFee))
}

// fork 未启用时保持历史行为, 保证旧区块可以原样重放
func TestTxChainIDBeforeFork(t *testing.T) {
	cfg := NewChain33Config(GetDefaultCfgstring())
	cfg.forks.ReplaceFork(ForkTxChainIDStrict, MaxHeight)

	priv := chainIDTestKey(t)
	minFee := cfg.GetMinTxFeeRate()
	maxFee := cfg.GetMaxTxFee(1)

	group, err := CreateTxGroup([]*Transaction{
		newForeignTx(t, priv, 0),
		newForeignTx(t, priv, 0),
	}, minFee)
	assert.Nil(t, err)
	assert.Nil(t, group.SignN(0, SECP256K1, priv))
	assert.Nil(t, group.SignN(1, SECP256K1, priv))

	// fork 之前, 交易组内不校验 chainID(历史行为)
	assert.Nil(t, group.Check(cfg, 1, minFee, maxFee))
}
