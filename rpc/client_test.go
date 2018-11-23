// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpc

import (
	"testing"

	"fmt"

	"github.com/33cn/chain33/account"
	"github.com/33cn/chain33/client/mocks"
	"github.com/33cn/chain33/common/address"
	slog "github.com/33cn/chain33/common/log"
	"github.com/33cn/chain33/pluginmgr"
	qmock "github.com/33cn/chain33/queue/mocks"
	cty "github.com/33cn/chain33/system/dapp/coins/types"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func init() {
	slog.SetLogLevel("error")
	types.Init("local", nil)
	pluginmgr.InitExec(nil)
}

func newTestChannelClient() *channelClient {
	return &channelClient{
		QueueProtocolAPI: &mocks.QueueProtocolAPI{},
	}
}

// TODO
func TestInit(t *testing.T) {
	client := newTestChannelClient()
	client.Init(&qmock.Client{}, nil)
}

func testCreateRawTransactionNil(t *testing.T) {
	client := newTestChannelClient()
	_, err := client.CreateRawTransaction(nil)
	assert.Equal(t, types.ErrInvalidParam, err)
}

func testCreateRawTransactionExecNameErr(t *testing.T) {
	tx := types.CreateTx{ExecName: "aaa", To: "1MY4pMgjpS2vWiaSDZasRhN47pcwEire32"}

	client := newTestChannelClient()
	_, err := client.CreateRawTransaction(&tx)
	assert.Equal(t, types.ErrExecNameNotMatch, err)
}

func testCreateRawTransactionAmoutErr(t *testing.T) {
	tx := types.CreateTx{ExecName: types.ExecName(cty.CoinsX), Amount: -1, To: "1MY4pMgjpS2vWiaSDZasRhN47pcwEire32"}

	client := newTestChannelClient()
	_, err := client.CreateRawTransaction(&tx)
	assert.Equal(t, types.ErrAmount, err)
}

func testCreateRawTransactionTo(t *testing.T) {
	name := types.ExecName(cty.CoinsX)
	tx := types.CreateTx{ExecName: name, Amount: 1, To: "1MY4pMgjpS2vWiaSDZasRhN47pcwEire32"}

	client := newTestChannelClient()
	rawtx, err := client.CreateRawTransaction(&tx)
	assert.Nil(t, err)
	var mytx types.Transaction
	err = types.Decode(rawtx, &mytx)
	assert.Nil(t, err)
	if types.IsPara() {
		assert.Equal(t, address.ExecAddress(name), mytx.To)
	} else {
		assert.Equal(t, tx.To, mytx.To)
	}
}

func testCreateRawTransactionCoinTransfer(t *testing.T) {
	ctx := types.CreateTx{
		ExecName:   "",
		Amount:     10,
		IsToken:    false,
		IsWithdraw: false,
		To:         "to",
		Note:       "note",
	}

	//v := &cty.CoinsAction_Transfer{
	//	Transfer:&cty.CoinsTransfer{
	//		Amount:ctx.Amount,
	//		Note:ctx.To,
	//	},
	//}
	//transfer := &cty.CoinsAction{
	//	Value:v,
	//	Ty:cty.CoinsActionTransfer,
	//}
	//
	//tx := &types.Transaction{
	//	Execer:[]byte("coins"),
	//	Payload:types.Encode(transfer),
	//	To:ctx.To,
	//}

	client := newTestChannelClient()
	txHex, err := client.CreateRawTransaction(&ctx)
	assert.Nil(t, err)
	var tx types.Transaction
	types.Decode(txHex, &tx)
	assert.Equal(t, []byte(types.ExecName(cty.CoinsX)), tx.Execer)

	var transfer cty.CoinsAction
	types.Decode(tx.Payload, &transfer)
	assert.Equal(t, int32(cty.CoinsActionTransfer), transfer.Ty)
}

func testCreateRawTransactionCoinTransferExec(t *testing.T) {
	name := types.ExecName("coins")
	ctx := types.CreateTx{
		ExecName:   name,
		Amount:     10,
		IsToken:    false,
		IsWithdraw: false,
		To:         "to",
		Note:       "note",
	}

	client := newTestChannelClient()
	txHex, err := client.CreateRawTransaction(&ctx)
	assert.Nil(t, err)
	var tx types.Transaction
	types.Decode(txHex, &tx)
	assert.Equal(t, []byte(types.ExecName(cty.CoinsX)), tx.Execer)

	var transfer cty.CoinsAction
	types.Decode(tx.Payload, &transfer)
	assert.Equal(t, int32(cty.CoinsActionTransferToExec), transfer.Ty)
	if types.IsPara() {
		assert.Equal(t, address.ExecAddress(types.ExecName(cty.CoinsX)), tx.To)
	} else {
		assert.Equal(t, ctx.To, tx.To)
	}
}

func testCreateRawTransactionCoinWithdraw(t *testing.T) {
	ctx := types.CreateTx{
		ExecName:   types.ExecName("coins"),
		Amount:     10,
		IsToken:    false,
		IsWithdraw: true,
		To:         "to",
		Note:       "note",
	}

	client := newTestChannelClient()
	txHex, err := client.CreateRawTransaction(&ctx)
	assert.Nil(t, err)
	var tx types.Transaction
	types.Decode(txHex, &tx)
	assert.Equal(t, []byte(types.ExecName(cty.CoinsX)), tx.Execer)

	var transfer cty.CoinsAction
	types.Decode(tx.Payload, &transfer)
	assert.Equal(t, int32(cty.CoinsActionWithdraw), transfer.Ty)

	if types.IsPara() {
		assert.Equal(t, address.ExecAddress(types.ExecName(cty.CoinsX)), tx.To)
	} else {
		assert.Equal(t, ctx.To, tx.To)
	}
}

func TestChannelClient_CreateRawTransaction(t *testing.T) {
	testCreateRawTransactionNil(t)
	testCreateRawTransactionExecNameErr(t)
	testCreateRawTransactionAmoutErr(t)
	testCreateRawTransactionTo(t)
	testCreateRawTransactionCoinTransfer(t)
	testCreateRawTransactionCoinTransferExec(t)
	testCreateRawTransactionCoinWithdraw(t)
}

func testSendRawTransactionNil(t *testing.T) {
	client := newTestChannelClient()
	_, err := client.SendRawTransaction(nil)
	assert.Equal(t, types.ErrInvalidParam, err)
}

func testSendRawTransactionErr(t *testing.T) {
	var param = types.SignedTx{
		Unsign: []byte("123"),
		Sign:   []byte("123"),
		Pubkey: []byte("123"),
		Ty:     1,
	}

	client := newTestChannelClient()
	_, err := client.SendRawTransaction(&param)
	assert.NotEmpty(t, err)
}

func testSendRawTransactionOk(t *testing.T) {
	transfer := &types.Transaction{
		Execer: []byte(types.ExecName("ticket")),
	}
	payload := types.Encode(transfer)

	api := new(mocks.QueueProtocolAPI)
	client := &channelClient{
		QueueProtocolAPI: api,
	}
	api.On("SendTx", mock.Anything).Return(nil, nil)

	var param = types.SignedTx{
		Unsign: payload,
		Sign:   []byte("123"),
		Pubkey: []byte("123"),
		Ty:     1,
	}

	_, err := client.SendRawTransaction(&param)
	assert.Nil(t, err)
}

func TestChannelClient_SendRawTransaction(t *testing.T) {
	testSendRawTransactionNil(t)
	testSendRawTransactionOk(t)
	testSendRawTransactionErr(t)
}

func testChannelClientGetAddrOverviewNil(t *testing.T) {
	parm := &types.ReqAddr{
		Addr: "abcde",
	}
	client := newTestChannelClient()
	_, err := client.GetAddrOverview(parm)
	assert.Equal(t, types.ErrInvalidAddress, err)
}

func testChannelClientGetAddrOverviewErr(t *testing.T) {
	parm := &types.ReqAddr{
		Addr: "1Jn2qu84Z1SUUosWjySggBS9pKWdAP3tZt",
	}
	api := new(mocks.QueueProtocolAPI)
	client := &channelClient{
		QueueProtocolAPI: api,
	}

	api.On("GetAddrOverview", mock.Anything).Return(nil, fmt.Errorf("error"))
	_, err := client.GetAddrOverview(parm)
	assert.EqualError(t, err, "error")

}

func testChannelClientGetAddrOverviewOK(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	db := new(account.DB)
	client := &channelClient{
		QueueProtocolAPI: api,
		accountdb:        db,
	}

	addr := &types.AddrOverview{}
	api.On("GetAddrOverview", mock.Anything).Return(addr, nil)

	head := &types.Header{StateHash: []byte("sdfadasds")}
	api.On("GetLastHeader").Return(head, nil)

	var acc = &types.Account{Addr: "1Jn2qu84Z1SUUosWjySggBS9pKWdAP3tZt", Balance: 100}
	accv := types.Encode(acc)
	storevalue := &types.StoreReplyValue{}
	storevalue.Values = append(storevalue.Values, accv)
	api.On("StoreGet", mock.Anything).Return(storevalue, nil)

	//reply := types.AddrOverview{}
	parm := &types.ReqAddr{
		Addr: "1Jn2qu84Z1SUUosWjySggBS9pKWdAP3tZt",
	}
	data, err := client.GetAddrOverview(parm)
	assert.Nil(t, err, "error")
	assert.Equal(t, acc.Balance, data.Balance)

}

func TestChannelClient_GetAddrOverview(t *testing.T) {
	testChannelClientGetAddrOverviewNil(t)
	testChannelClientGetAddrOverviewOK(t)
	testChannelClientGetAddrOverviewErr(t)
}

func testChannelClient_GetBalanceCoin(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	db := new(account.DB)
	client := &channelClient{
		QueueProtocolAPI: api,
		accountdb:        db,
	}

	//addr := &types.AddrOverview{}
	//api.On("GetAddrOverview", mock.Anything).Return(addr, nil)

	head := &types.Header{StateHash: []byte("sdfadasds")}
	api.On("GetLastHeader").Return(head, nil)

	var acc = &types.Account{Addr: "1Jn2qu84Z1SUUosWjySggBS9pKWdAP3tZt", Balance: 100}
	accv := types.Encode(acc)
	storevalue := &types.StoreReplyValue{}
	storevalue.Values = append(storevalue.Values, accv)
	api.On("StoreGet", mock.Anything).Return(storevalue, nil)

	var addrs = make([]string, 1)
	addrs = append(addrs, "1Jn2qu84Z1SUUosWjySggBS9pKWdAP3tZt")
	var in = &types.ReqBalance{
		Execer:    "coins",
		Addresses: addrs,
	}
	data, err := client.GetBalance(in)
	assert.Nil(t, err)
	assert.Equal(t, acc.Addr, data[0].Addr)

}

func testChannelClient_GetBalanceOther(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	db := new(account.DB)
	client := &channelClient{
		QueueProtocolAPI: api,
		accountdb:        db,
	}

	//addr := &types.AddrOverview{}
	//api.On("GetAddrOverview", mock.Anything).Return(addr, nil)

	head := &types.Header{StateHash: []byte("sdfadasds")}
	api.On("GetLastHeader").Return(head, nil)

	var acc = &types.Account{Addr: "1Jn2qu84Z1SUUosWjySggBS9pKWdAP3tZt", Balance: 100}
	accv := types.Encode(acc)
	storevalue := &types.StoreReplyValue{}
	storevalue.Values = append(storevalue.Values, accv)
	api.On("StoreGet", mock.Anything).Return(storevalue, nil)

	var addrs = make([]string, 1)
	addrs = append(addrs, "1Jn2qu84Z1SUUosWjySggBS9pKWdAP3tZt")
	var in = &types.ReqBalance{
		Execer:    types.ExecName("ticket"),
		Addresses: addrs,
	}
	data, err := client.GetBalance(in)
	assert.Nil(t, err)
	assert.Equal(t, acc.Addr, data[0].Addr)

}

func TestChannelClient_GetBalance(t *testing.T) {
	testChannelClient_GetBalanceCoin(t)
	testChannelClient_GetBalanceOther(t)
}

// func TestChannelClient_GetTotalCoins(t *testing.T) {
// 	client := newTestChannelClient()
// 	data, err := client.GetTotalCoins(nil)
// 	assert.NotNil(t, err)
// 	assert.Nil(t, data)
//
// 	// accountdb =
// 	token := &types.ReqGetTotalCoins{
// 		Symbol:    "CNY",
// 		StateHash: []byte("1234"),
// 		StartKey:  []byte("sad"),
// 		Count:     1,
// 		Execer:    "coin",
// 	}
// 	data, err = client.GetTotalCoins(token)
// 	assert.NotNil(t, data)
// 	assert.Nil(t, err)
// }
