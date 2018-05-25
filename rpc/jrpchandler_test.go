package rpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.33.cn/chain33/chain33/types"
)

func TestDecodeUserWrite(t *testing.T) {
	payload := []byte("#md#hello#world")
	data := decodeUserWrite(payload)
	assert.Equal(t, data, &userWrite{Topic: "md", Content: "hello#world"})

	payload = []byte("hello#world")
	data = decodeUserWrite(payload)
	assert.Equal(t, data, &userWrite{Topic: "", Content: "hello#world"})

	payload = []byte("123#hello#suyanlong")
	data = decodeUserWrite(payload)
	assert.NotEqual(t, data, &userWrite{Topic: "123", Content: "hello#world"})
}

func TestDecodeTx(t *testing.T) {
	tx := types.Transaction{
		Execer:  []byte("coin"),
		Payload: []byte("342412abcd"),
		Nonce:   8978167239,
		To:      "1asd234dsf43fds",
	}

	data, err := DecodeTx(&tx)
	assert.NotNil(t, data)
	assert.Nil(t, err)

	tx.Execer = []byte("coins")
	data, err = DecodeTx(&tx)
	assert.NotNil(t, err)
	assert.Nil(t, data)

	tx = types.Transaction{
		Execer:  []byte("hashlock"),
		Payload: []byte("34"),
		Nonce:   8978167239,
		To:      "1asd234dsf43fds",
	}

	t.Log(string(tx.Execer))
	data, err = DecodeTx(&tx)
	assert.NotNil(t, data)
	assert.Nil(t, err)
}

func TestDecodeLog(t *testing.T) {
	var data = &ReceiptData{
		Ty: 5,
	}
	result, err := DecodeLog(data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
}
