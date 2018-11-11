package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/33cn/chain33/types"
)

func TestDecodeUserWrite(t *testing.T) {
	payload := []byte("#md#hello#world")
	data := decodeUserWrite(payload)
	assert.Equal(t, data, &types.UserWrite{Topic: "md", Content: "hello#world"})

	payload = []byte("hello#world")
	data = decodeUserWrite(payload)
	assert.Equal(t, data, &types.UserWrite{Topic: "", Content: "hello#world"})

	payload = []byte("123#hello#wzw")
	data = decodeUserWrite(payload)
	assert.NotEqual(t, data, &types.UserWrite{Topic: "123", Content: "hello#world"})
}

func TestDecodeTx(t *testing.T) {
	tx := types.Transaction{
		Execer:  []byte(types.ExecName("coin")),
		Payload: []byte("342412abcd"),
		Nonce:   8978167239,
		To:      "1asd234dsf43fds",
	}

	data, err := DecodeTx(&tx)
	assert.NotNil(t, data)
	assert.Nil(t, err)

	tx.Execer = []byte(types.ExecName("coins"))
	data, err = DecodeTx(&tx)
	assert.NotNil(t, data)
	assert.Nil(t, err)

	tx = types.Transaction{
		Execer:  []byte(types.ExecName("hashlock")),
		Payload: []byte("34"),
		Nonce:   8978167239,
		To:      "1asd234dsf43fds",
	}

	t.Log(string(tx.Execer))
	data, err = DecodeTx(&tx)
	assert.NotNil(t, data)
	assert.Nil(t, err)
}
