package rpc

import (
	"testing"

	"fmt"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/client/mocks"
	slog "gitlab.33.cn/chain33/chain33/common/log"
	qmock "gitlab.33.cn/chain33/chain33/queue/mocks"
	"gitlab.33.cn/chain33/chain33/types"
	_ "gitlab.33.cn/chain33/chain33/types/executor"
	exec "gitlab.33.cn/chain33/chain33/types/executor"
	evmtype "gitlab.33.cn/chain33/chain33/types/executor/evm"
	hashlocktype "gitlab.33.cn/chain33/chain33/types/executor/hashlock"
	retrievetype "gitlab.33.cn/chain33/chain33/types/executor/retrieve"
	tokentype "gitlab.33.cn/chain33/chain33/types/executor/token"
	tradetype "gitlab.33.cn/chain33/chain33/types/executor/trade"
)

func init() {
	slog.SetLogLevel("error")
	types.SetTitle("user.p.guodun.")
}

func newTestChannelClient() *channelClient {
	return &channelClient{
		QueueProtocolAPI: &mocks.QueueProtocolAPI{},
	}
}

// TODO
func TestInit(t *testing.T) {
	client := newTestChannelClient()
	client.Init(&qmock.Client{})
	exec.Init()
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
	tx := types.CreateTx{ExecName: types.ExecName(types.CoinsX), Amount: -1, To: "1MY4pMgjpS2vWiaSDZasRhN47pcwEire32"}

	client := newTestChannelClient()
	_, err := client.CreateRawTransaction(&tx)
	assert.Equal(t, types.ErrAmount, err)
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

	//v := &types.CoinsAction_Transfer{
	//	Transfer:&types.CoinsTransfer{
	//		Amount:ctx.Amount,
	//		Note:ctx.To,
	//	},
	//}
	//transfer := &types.CoinsAction{
	//	Value:v,
	//	Ty:types.CoinsActionTransfer,
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
	assert.Equal(t, []byte(types.ExecName(types.CoinsX)), tx.Execer)

	var transfer types.CoinsAction
	types.Decode(tx.Payload, &transfer)
	assert.Equal(t, int32(types.CoinsActionTransfer), transfer.Ty)
}

func testCreateRawTransactionCoinTransferExec(t *testing.T) {
	ctx := types.CreateTx{
		ExecName:   types.ExecName(types.TicketX),
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
	assert.Equal(t, []byte(types.ExecName(types.CoinsX)), tx.Execer)

	var transfer types.CoinsAction
	types.Decode(tx.Payload, &transfer)
	assert.Equal(t, int32(types.CoinsActionTransferToExec), transfer.Ty)
}

func testCreateRawTransactionCoinWithdraw(t *testing.T) {
	ctx := types.CreateTx{
		ExecName:   types.ExecName(types.TicketX),
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
	assert.Equal(t, []byte(types.ExecName(types.CoinsX)), tx.Execer)

	var transfer types.CoinsAction
	types.Decode(tx.Payload, &transfer)
	assert.Equal(t, int32(types.CoinsActionWithdraw), transfer.Ty)
}

func testCreateRawTransactionTokenTransfer(t *testing.T) {
	ctx := types.CreateTx{
		ExecName:   types.ExecName(types.TokenX),
		Amount:     10,
		IsToken:    true,
		IsWithdraw: false,
		To:         "to",
		Note:       "note",
	}

	client := newTestChannelClient()
	txHex, err := client.CreateRawTransaction(&ctx)
	assert.Nil(t, err)
	var tx types.Transaction
	types.Decode(txHex, &tx)
	assert.Equal(t, []byte(types.ExecName(types.TokenX)), tx.Execer)

	var transfer types.TokenAction
	types.Decode(tx.Payload, &transfer)
	assert.Equal(t, int32(types.ActionTransfer), transfer.Ty)
}

func testCreateRawTransactionTokenWithdraw(t *testing.T) {
	ctx := types.CreateTx{
		ExecName:   types.ExecName(types.TokenX),
		Amount:     10,
		IsToken:    true,
		IsWithdraw: true,
		To:         "to",
		Note:       "note",
	}

	client := newTestChannelClient()
	txHex, err := client.CreateRawTransaction(&ctx)
	assert.Nil(t, err)
	var tx types.Transaction
	types.Decode(txHex, &tx)
	assert.Equal(t, []byte(types.ExecName(types.TokenX)), tx.Execer)

	var transfer types.TokenAction
	types.Decode(tx.Payload, &transfer)
	assert.Equal(t, int32(types.ActionWithdraw), transfer.Ty)
}

func TestChannelClient_CreateRawTransaction(t *testing.T) {
	testCreateRawTransactionNil(t)
	testCreateRawTransactionExecNameErr(t)
	testCreateRawTransactionAmoutErr(t)
	testCreateRawTransactionCoinTransfer(t)
	testCreateRawTransactionCoinTransferExec(t)
	testCreateRawTransactionCoinWithdraw(t)

	testCreateRawTransactionTokenTransfer(t)
	testCreateRawTransactionTokenWithdraw(t)

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
		Execer: []byte(types.ExecName(types.TicketX)),
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
		Execer:    types.ExecName(types.TicketX),
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

func testChannelClient_GetTokenBalanceToken(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	db := new(account.DB)
	client := &channelClient{
		QueueProtocolAPI: api,
		accountdb:        db,
	}

	head := &types.Header{StateHash: []byte("sdfadasds")}
	api.On("GetLastHeader").Return(head, nil)

	var acc = &types.Account{Addr: "1Jn2qu84Z1SUUosWjySggBS9pKWdAP3tZt", Balance: 100}
	accv := types.Encode(acc)
	storevalue := &types.StoreReplyValue{}
	storevalue.Values = append(storevalue.Values, accv)
	api.On("StoreGet", mock.Anything).Return(storevalue, nil)

	var addrs = make([]string, 1)
	addrs = append(addrs, "1Jn2qu84Z1SUUosWjySggBS9pKWdAP3tZt")
	var in = &types.ReqTokenBalance{
		Execer:      types.ExecName(types.TokenX),
		Addresses:   addrs,
		TokenSymbol: "xxx",
	}
	data, err := client.GetTokenBalance(in)
	assert.Nil(t, err)
	assert.Equal(t, acc.Addr, data[0].Addr)

}

func testChannelClient_GetTokenBalanceOther(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	db := new(account.DB)
	client := &channelClient{
		QueueProtocolAPI: api,
		accountdb:        db,
	}

	head := &types.Header{StateHash: []byte("sdfadasds")}
	api.On("GetLastHeader").Return(head, nil)

	var acc = &types.Account{Addr: "1Jn2qu84Z1SUUosWjySggBS9pKWdAP3tZt", Balance: 100}
	accv := types.Encode(acc)
	storevalue := &types.StoreReplyValue{}
	storevalue.Values = append(storevalue.Values, accv)
	api.On("StoreGet", mock.Anything).Return(storevalue, nil)

	var addrs = make([]string, 1)
	addrs = append(addrs, "1Jn2qu84Z1SUUosWjySggBS9pKWdAP3tZt")
	var in = &types.ReqTokenBalance{
		Execer:      types.ExecName(types.TradeX),
		Addresses:   addrs,
		TokenSymbol: "xxx",
	}
	data, err := client.GetTokenBalance(in)
	assert.Nil(t, err)
	assert.Equal(t, acc.Addr, data[0].Addr)

}

func TestChannelClient_GetTokenBalance(t *testing.T) {
	testChannelClient_GetTokenBalanceToken(t)
	testChannelClient_GetTokenBalanceOther(t)

}

func TestChannelClient_CreateRawTokenPreCreateTx(t *testing.T) {
	client := newTestChannelClient()
	data, err := client.CreateRawTokenPreCreateTx(nil)
	assert.NotNil(t, err)
	assert.Nil(t, data)

	token := &tokentype.TokenPreCreateTx{
		OwnerAddr: "asdf134",
		Symbol:    "CNY",
		Fee:       123,
	}
	data, err = client.CreateRawTokenPreCreateTx(token)
	assert.NotNil(t, data)
	assert.Nil(t, err)
}

func TestChannelClient_CreateRawTokenRevokeTx(t *testing.T) {
	client := newTestChannelClient()
	data, err := client.CreateRawTokenRevokeTx(nil)
	assert.NotNil(t, err)
	assert.Nil(t, data)

	token := &tokentype.TokenRevokeTx{
		OwnerAddr: "asdf134",
		Symbol:    "CNY",
		Fee:       123,
	}
	data, err = client.CreateRawTokenRevokeTx(token)
	assert.NotNil(t, data)
	assert.Nil(t, err)
}

func TestChannelClient_CreateRawTokenFinishTx(t *testing.T) {
	client := newTestChannelClient()
	data, err := client.CreateRawTokenFinishTx(nil)
	assert.NotNil(t, err)
	assert.Nil(t, data)

	token := &tokentype.TokenFinishTx{
		OwnerAddr: "asdf134",
		Symbol:    "CNY",
		Fee:       123,
	}
	data, err = client.CreateRawTokenFinishTx(token)
	assert.NotNil(t, data)
	assert.Nil(t, err)
}

func TestChannelClient_CreateRawTradeSellTx(t *testing.T) {
	client := newTestChannelClient()
	data, err := client.CreateRawTradeSellTx(nil)
	assert.NotNil(t, err)
	assert.Nil(t, data)

	token := &tradetype.TradeSellTx{
		TokenSymbol:       "CNY",
		AmountPerBoardlot: 10,
		MinBoardlot:       1,
		PricePerBoardlot:  100,
		TotalBoardlot:     100,
		Fee:               1,
	}
	data, err = client.CreateRawTradeSellTx(token)
	assert.NotNil(t, data)
	assert.Nil(t, err)
}

func TestChannelClient_CreateRawTradeBuyTx(t *testing.T) {
	client := newTestChannelClient()
	data, err := client.CreateRawTradeBuyTx(nil)
	assert.NotNil(t, err)
	assert.Nil(t, data)

	token := &tradetype.TradeBuyTx{
		SellID:      "sadfghjkhgfdsa",
		BoardlotCnt: 100,
		Fee:         1,
	}
	data, err = client.CreateRawTradeBuyTx(token)
	assert.NotNil(t, data)
	assert.Nil(t, err)
}

func TestChannelClient_CreateRawTradeRevokeTx(t *testing.T) {
	client := newTestChannelClient()
	data, err := client.CreateRawTradeRevokeTx(nil)
	assert.NotNil(t, err)
	assert.Nil(t, data)

	token := &tradetype.TradeRevokeTx{
		SellID: "sadfghjkhgfdsa",
		Fee:    1,
	}
	data, err = client.CreateRawTradeRevokeTx(token)
	assert.NotNil(t, data)
	assert.Nil(t, err)
}

func TestChannelClient_CreateRawTradeBuyLimitTx(t *testing.T) {
	client := newTestChannelClient()
	data, err := client.CreateRawTradeBuyLimitTx(nil)
	assert.NotNil(t, err)
	assert.Nil(t, data)

	token := &tradetype.TradeBuyLimitTx{
		TokenSymbol:       "CNY",
		AmountPerBoardlot: 10,
		MinBoardlot:       1,
		PricePerBoardlot:  100,
		TotalBoardlot:     100,
		Fee:               1,
	}
	data, err = client.CreateRawTradeBuyLimitTx(token)
	assert.NotNil(t, data)
	assert.Nil(t, err)
}

func TestChannelClient_CreateRawTradeSellMarketTx(t *testing.T) {
	client := newTestChannelClient()
	data, err := client.CreateRawTradeSellMarketTx(nil)
	assert.NotNil(t, err)
	assert.Nil(t, data)

	token := &tradetype.TradeSellMarketTx{
		BuyID:       "12asdfa",
		BoardlotCnt: 100,
		Fee:         1,
	}
	data, err = client.CreateRawTradeSellMarketTx(token)
	assert.NotNil(t, data)
	assert.Nil(t, err)
}

func TestChannelClient_CreateRawTradeRevokeBuyTx(t *testing.T) {
	client := newTestChannelClient()
	data, err := client.CreateRawTradeRevokeBuyTx(nil)
	assert.NotNil(t, err)
	assert.Nil(t, data)

	token := &tradetype.TradeRevokeBuyTx{
		BuyID: "12asdfa",
		Fee:   1,
	}
	data, err = client.CreateRawTradeRevokeBuyTx(token)
	assert.NotNil(t, data)
	assert.Nil(t, err)
}

func TestChannelClient_CreateRawRetrieveBackupTx(t *testing.T) {
	client := newTestChannelClient()
	data, err := client.CreateRawRetrieveBackupTx(nil)
	assert.NotNil(t, err)
	assert.Nil(t, data)

	backup := &retrievetype.RetrieveBackupTx{
		BackupAddr:  "12asdfa",
		DefaultAddr: "0x3456",
		DelayPeriod: 1,
		Fee:         1,
	}
	data, err = client.CreateRawRetrieveBackupTx(backup)
	assert.NotNil(t, data)
	assert.Nil(t, err)
}

func TestChannelClient_CreateRawRetrievePrepareTx(t *testing.T) {
	client := newTestChannelClient()
	data, err := client.CreateRawRetrievePrepareTx(nil)
	assert.NotNil(t, err)
	assert.Nil(t, data)

	prepare := &retrievetype.RetrievePrepareTx{
		BackupAddr:  "12asdfa",
		DefaultAddr: "0x3456",
		Fee:         1,
	}
	data, err = client.CreateRawRetrievePrepareTx(prepare)
	assert.NotNil(t, data)
	assert.Nil(t, err)
}

func TestChannelClient_CreateRawRetrievePerformTx(t *testing.T) {
	client := newTestChannelClient()
	data, err := client.CreateRawRetrievePerformTx(nil)
	assert.NotNil(t, err)
	assert.Nil(t, data)

	perform := &retrievetype.RetrievePerformTx{
		BackupAddr:  "12asdfa",
		DefaultAddr: "0x3456",
		Fee:         1,
	}
	data, err = client.CreateRawRetrievePerformTx(perform)
	assert.NotNil(t, data)
	assert.Nil(t, err)
}

func TestChannelClient_CreateRawRetrieveCancelTx(t *testing.T) {
	client := newTestChannelClient()
	data, err := client.CreateRawRetrieveCancelTx(nil)
	assert.NotNil(t, err)
	assert.Nil(t, data)

	cancel := &retrievetype.RetrieveCancelTx{
		BackupAddr:  "12asdfa",
		DefaultAddr: "0x3456",
		Fee:         1,
	}
	data, err = client.CreateRawRetrieveCancelTx(cancel)
	assert.NotNil(t, data)
	assert.Nil(t, err)
}

func TestChannelClient_CreateRawHashlockLockTx(t *testing.T) {
	client := newTestChannelClient()
	data, err := client.CreateRawHashlockLockTx(nil)
	assert.NotNil(t, err)
	assert.Nil(t, data)

	lock := &hashlocktype.HashlockLockTx{
		Secret:     "12asdfa",
		Amount:     100,
		Time:       100,
		ToAddr:     "12asdfa",
		ReturnAddr: "0x3456",
		Fee:        1,
	}
	data, err = client.CreateRawHashlockLockTx(lock)
	assert.NotNil(t, data)
	assert.Nil(t, err)
}

func TestChannelClient_CreateRawHashlockUnlockTx(t *testing.T) {
	client := newTestChannelClient()
	data, err := client.CreateRawHashlockUnlockTx(nil)
	assert.NotNil(t, err)
	assert.Nil(t, data)

	unlock := &hashlocktype.HashlockUnlockTx{
		Secret: "12asdfa",
		Fee:    1,
	}
	data, err = client.CreateRawHashlockUnlockTx(unlock)
	assert.NotNil(t, data)
	assert.Nil(t, err)
}

func TestChannelClient_CreateRawHashlockSendTx(t *testing.T) {
	client := newTestChannelClient()
	data, err := client.CreateRawHashlockSendTx(nil)
	assert.NotNil(t, err)
	assert.Nil(t, data)

	send := &hashlocktype.HashlockSendTx{
		Secret: "12asdfa",
		Fee:    1,
	}
	data, err = client.CreateRawHashlockSendTx(send)
	assert.NotNil(t, data)
	assert.Nil(t, err)
}

func TestChannelClient_CreateRawEvmCreateCallTx(t *testing.T) {
	client := newTestChannelClient()
	data, err := client.CreateRawEvmCreateCallTx(nil)
	assert.NotNil(t, err)
	assert.Nil(t, data)

	createcall := &evmtype.CreateCallTx{
		Amount:   0,
		Code:     "12",
		GasLimit: 0,
		GasPrice: 0,
		Note:     "12",
		Alias:    "12",
		Fee:      0,
		Name:     "12",
		IsCreate: true,
	}

	data, err = client.CreateRawEvmCreateCallTx(createcall)
	assert.NotNil(t, data)
	assert.Nil(t, err)
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
