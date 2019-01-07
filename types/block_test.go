package types

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBlock(t *testing.T) {
	b := &Block{}
	assert.Equal(t, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", hex.EncodeToString(b.Hash()))
	assert.Equal(t, b.HashOld(), b.HashNew())
	assert.Equal(t, b.HashOld(), b.Hash())
	b.Height = 10
	b.Difficulty = 1
	assert.NotEqual(t, b.HashOld(), b.HashNew())
	assert.NotEqual(t, b.HashOld(), b.HashNew())
	assert.Equal(t, b.HashNew(), b.HashByForkHeight(10))
	assert.Equal(t, b.HashOld(), b.HashByForkHeight(11))
	assert.Equal(t, true, b.CheckSign())

	b.Txs = append(b.Txs, &Transaction{})
	assert.Equal(t, false, b.CheckSign())
	b.Txs = append(b.Txs, &Transaction{})
	b.Txs = append(b.Txs, &Transaction{})
	b.Txs = append(b.Txs, &Transaction{})
	b.Txs = append(b.Txs, &Transaction{})
	assert.Equal(t, false, b.CheckSign())
}
