package relay

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.33.cn/chain33/chain33/types"
)

func TestVerifyBlockHeader(t *testing.T) {
	var head = &types.BtcHeader{
		Version:      1,
		Hash:         "00000000d1145790a8694403d4063f323d499e655c83426834d4ce2f8dd4a2ee",
		MerkleRoot:   "7dac2c5666815c17a3b36427de37bb9d2e2c5ccec3f8633eb91a4205cb4c10ff",
		PreviousHash: "000000002a22cfee1f2c846adbd12b3e183d4f97683f85dad08a79780a84bd55",
		Bits:         "486604799",
		Nonce:        1889418792,
		Time:         1231731025,
		Height:       2,
	}

	var store = &btcStore{
		lastHeader: &types.BtcHeader{},
	}
	store.lastHeader.Hash = "000000002a22cfee1f2c846adbd12b3e183d4f97683f85dad08a79780a84bd55"
	store.lastHeader.Height = 1

	re := store.verifyBlockHeader(head)
	assert.Equal(t, true, re)
}
