// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	etypes "github.com/ethereum/go-ethereum/core/types"
)

func ethEnvelopeTx(t *testing.T, cfg *Chain33Config, gas uint64, fee int64) *Transaction {
	t.Helper()
	raw, err := etypes.NewTx(&etypes.LegacyTx{Gas: gas, Value: big.NewInt(0)}).MarshalBinary()
	assert.Nil(t, err)
	payload := Encode(&EVMContractAction4Chain33{Note: hex.EncodeToString(raw)})
	return &Transaction{
		Execer:    []byte(cfg.ExecName("evm")),
		Payload:   payload,
		Fee:       fee,
		To:        "1J7X82xR9tJr8s2oCDgrWHu8ThULyfmzRb",
		ChainID:   cfg.GetChainID(),
		Signature: &Signature{Ty: EncodeSignID(SECP256K1ETH, EthAddressID)},
	}
}

// TestEthEnvelopePinsUncoveredFields 覆盖 B2: 执行器名、chainID、手续费都在 eth 签名
// 覆盖范围之外，改写后交易哈希变化，靠哈希去重拦不住，必须在交易检查层钉住。
func TestEthEnvelopePinsUncoveredFields(t *testing.T) {
	cfg := NewChain33Config(ReadFile("../cmd/chain33/chain33.test.toml"))
	minFee := cfg.GetMinTxFeeRate()
	maxFee := cfg.GetMaxTxFee(1)
	assertCheck := func(tx *Transaction, want error) {
		t.Helper()
		tx.Signature.Pubkey = []byte("pubkey")
		tx.Signature.Signature = []byte("sig")
		assert.Equal(t, want, tx.Check(cfg, 1, minFee, maxFee))
	}

	// baseline: 组装器产出的形状通过
	assertCheck(ethEnvelopeTx(t, cfg, 100000, minFee), nil)

	// 执行器名被改写
	renamed := ethEnvelopeTx(t, cfg, 100000, minFee)
	renamed.Execer = []byte("user.evm.other")
	assertCheck(renamed, ErrExecNameNotAllow)

	// chainID 被改写
	foreign := ethEnvelopeTx(t, cfg, 100000, minFee)
	foreign.ChainID = cfg.GetChainID() + 1
	assertCheck(foreign, ErrTxChainID)

	// 手续费被抬高到 eth gas 之外
	overFee := ethEnvelopeTx(t, cfg, 100000, minFee+1)
	assertCheck(overFee, ErrTxFeeTooHigh)

	// eth gas 更高时，手续费上限随 gas 抬高
	assertCheck(ethEnvelopeTx(t, cfg, 200000, 200000), nil)
	assertCheck(ethEnvelopeTx(t, cfg, 200000, 200001), ErrTxFeeTooHigh)
}

// TestEthEnvelopeOnlyAppliesToCarrier 确认普通 chain33 格式交易不受新规则影响。
func TestEthEnvelopeOnlyAppliesToCarrier(t *testing.T) {
	cfg := NewChain33Config(ReadFile("../cmd/chain33/chain33.test.toml"))
	tx := &Transaction{
		Execer:    []byte("coins"),
		Payload:   []byte("not an evm payload"),
		Fee:       cfg.GetMinTxFeeRate() + 1,
		To:        "1J7X82xR9tJr8s2oCDgrWHu8ThULyfmzRb",
		ChainID:   cfg.GetChainID(),
		Signature: &Signature{Ty: EncodeSignID(SECP256K1ETH, EthAddressID)},
	}
	assert.Nil(t, tx.Check(cfg, 1, cfg.GetMinTxFeeRate(), cfg.GetMaxTxFee(1)))
}
