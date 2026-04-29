package types

import (
	"math/big"
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/ethereum/go-ethereum/common"
	etypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/assert"
)

func TestCaculateRealVLegacy(t *testing.T) {
	v := big.NewInt(62)
	b, err := CaculateRealV(v, 0, etypes.LegacyTxType)
	assert.Nil(t, err)
	assert.Equal(t, byte(35), b)

	v2 := big.NewInt(99)
	b2, err := CaculateRealV(v2, 1, etypes.LegacyTxType)
	assert.Nil(t, err)
	assert.Equal(t, byte(v2.Uint64()-37), b2)
}

func TestCaculateRealVDynamicFee(t *testing.T) {
	v := big.NewInt(1)
	b, err := CaculateRealV(v, 0, etypes.DynamicFeeTxType)
	assert.Nil(t, err)
	assert.Equal(t, byte(1), b)
}

func TestCaculateRealVAccessList(t *testing.T) {
	v := big.NewInt(28)
	b, err := CaculateRealV(v, 0, etypes.AccessListTxType)
	assert.Nil(t, err)
	assert.Equal(t, byte(1), b)
}

func TestCaculateRealVInvalidType(t *testing.T) {
	_, err := CaculateRealV(big.NewInt(0), 0, 99)
	assert.NotNil(t, err)
}

func TestBlockHeaderToEthHeader(t *testing.T) {
	h := &types.Header{
		BlockTime:  1234567890,
		Height:     100,
		TxHash:     make([]byte, 32),
		Difficulty: 5,
		ParentHash: make([]byte, 32),
		StateHash:  make([]byte, 32),
	}
	eth, err := BlockHeaderToEthHeader(h)
	assert.Nil(t, err)
	assert.NotNil(t, eth)
	assert.Equal(t, int64(100), eth.Number.ToInt().Int64())
}

func TestCreateBloom(t *testing.T) {
	bloom := CreateBloom(nil)
	assert.Equal(t, etypes.Bloom{}, bloom)

	receipts := []*Receipt{{
		Logs: []*EvmLog{{
			Address: common.HexToAddress("0x1234"),
			Topics:  []common.Hash{common.HexToHash("0xabcd")},
		}},
	}}
	bloom2 := CreateBloom(receipts)
	assert.True(t, bloom2.Test(common.HexToAddress("0x1234").Bytes()))
}
