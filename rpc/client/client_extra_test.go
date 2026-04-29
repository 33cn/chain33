package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeTx(t *testing.T) {
	// Valid minimal transaction hex
	txHex := "0a05636f696e73120e18010a0a1080c2d72f1a036f746520a08d0630f1cdebc8f7efa5e9283a22313271796f6361794e46374c7636433971573461767873324537553431664b536676"
	tx, err := DecodeTx(txHex)
	assert.Nil(t, err)
	assert.NotNil(t, tx)
	assert.Equal(t, "coins", string(tx.Execer))

	// Invalid hex
	_, err = DecodeTx("invalid")
	assert.NotNil(t, err)

	// Empty string
	_, err = DecodeTx("")
	assert.Nil(t, err)
}

func TestDecodeTxInvalidHex(t *testing.T) {
	_, err := DecodeTx("zzzz")
	assert.NotNil(t, err)
}
