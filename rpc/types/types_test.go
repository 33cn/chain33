// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
	"github.com/33cn/chain33/client/mocks"
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

func TestDecodeLog(t *testing.T) {
	execer := []byte("coins")
	log := &ReceiptLog{}
	receipt := &ReceiptData{Ty: 2, Logs: []*ReceiptLog{log}}
	_, err := DecodeLog(execer, receipt)
	assert.NoError(t, err)
}

func TestConvertWalletTxDetailToJSON(t *testing.T) {

	tx := &types.Transaction{Execer:[]byte("coins")}
	log := &types.ReceiptLog{Ty: 0, Log: []byte("test")}
	receipt := &types.ReceiptData{Ty: 0, Logs: []*types.ReceiptLog{log}}
	detail := &types.WalletTxDetail{Tx: tx, Receipt: receipt}
	in := &types.WalletTxDetails{TxDetails: []*types.WalletTxDetail{detail}}
	out := &WalletTxDetails{}
	err := ConvertWalletTxDetailToJSON(in, out)
	assert.NoError(t, err)
}

func TestServer(t *testing.T) {
	api := &mocks.QueueProtocolAPI{}
	ch := ChannelClient{QueueProtocolAPI: api}
	ch.Init("test", nil, nil, nil)
	db := ch.GetCoinsAccountDB()
	assert.NotNil(t, db)
}