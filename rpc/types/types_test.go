// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"encoding/json"
	"testing"

	"github.com/33cn/chain33/client/mocks"
	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
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

	tx := &types.Transaction{Execer: []byte("coins")}
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

func TestDecodeTx2(t *testing.T) {
	bdata, err := common.FromHex("0a05636f696e73121018010a0c108084af5f1a05310a320a3320e8b31b30b9b69483d7f9d3f04c3a22314b67453376617969715a4b6866684d66744e3776743267447639486f4d6b393431")
	assert.Nil(t, err)
	var r types.Transaction
	err = types.Decode(bdata, &r)
	assert.Nil(t, err)
	data, err := DecodeTx(&r)
	assert.Nil(t, err)
	jsondata, err := json.Marshal(data)
	assert.Nil(t, err)
	assert.Equal(t, string(jsondata), `{"execer":"coins","payload":{"transfer":{"cointoken":"","amount":"200000000","note":"1\n2\n3","to":""},"ty":1},"rawPayload":"0x18010a0c108084af5f1a05310a320a33","signature":{"ty":0,"pubkey":"","signature":""},"fee":449000,"feefmt":"0.0045","expire":0,"nonce":5539796760414985017,"from":"1HT7xU2Ngenf7D4yocz2SAcnNLW7rK8d4E","to":"1KgE3vayiqZKhfhMftN7vt2gDv9HoMk941","hash":"0x6f9d543a345f6e17d8c3cc5f846c22570acf3b4b5851f48d0c2be5459d90c410"}`)
}
