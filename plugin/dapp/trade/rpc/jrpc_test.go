package rpc

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/mock"

	"gitlab.33.cn/chain33/chain33/types"

	"github.com/stretchr/testify/assert"
	"gitlab.33.cn/chain33/chain33/client/mocks"
	ptypes "gitlab.33.cn/chain33/chain33/plugin/dapp/trade/types"
	rpctypes "gitlab.33.cn/chain33/chain33/rpc/types"
)

func newTestChain33(api *mocks.QueueProtocolAPI) *Jrpc {
	cli := &channelClient{
		ChannelClient: rpctypes.ChannelClient{
			QueueProtocolAPI: api,
		},
	}
	return &Jrpc{cli: cli}
}

func TestChain33_CreateRawTradeSellTx(t *testing.T) {
	client := newTestChain33(nil)
	var testResult interface{}
	err := client.CreateRawTradeSellTx(nil, &testResult)
	assert.NotNil(t, err)
	assert.Nil(t, testResult)

	token := &ptypes.TradeSellTx{
		TokenSymbol:       "CNY",
		AmountPerBoardlot: 10,
		MinBoardlot:       1,
		PricePerBoardlot:  100,
		TotalBoardlot:     100,
		Fee:               1,
	}

	err = client.CreateRawTradeSellTx(token, &testResult)
	assert.NotNil(t, testResult)
	assert.Nil(t, err)
}

func TestChain33_CreateRawTradeBuyTx(t *testing.T) {
	client := newTestChain33(nil)
	var testResult interface{}
	err := client.CreateRawTradeBuyTx(nil, &testResult)
	assert.NotNil(t, err)
	assert.Nil(t, testResult)

	token := &ptypes.TradeBuyTx{
		SellID:      "sadfghjkhgfdsa",
		BoardlotCnt: 100,
		Fee:         1,
	}

	err = client.CreateRawTradeBuyTx(token, &testResult)
	assert.NotNil(t, testResult)
	assert.Nil(t, err)
}

func TestChain33_CreateRawTradeRevokeTx(t *testing.T) {
	client := newTestChain33(nil)
	var testResult interface{}
	err := client.CreateRawTradeRevokeTx(nil, &testResult)
	assert.NotNil(t, err)
	assert.Nil(t, testResult)

	token := &ptypes.TradeRevokeTx{
		SellID: "sadfghjkhgfdsa",
		Fee:    1,
	}

	err = client.CreateRawTradeRevokeTx(token, &testResult)
	assert.NotNil(t, testResult)
	assert.Nil(t, err)

}

func TestChain33_CreateRawTradeBuyLimitTx(t *testing.T) {
	client := newTestChain33(nil)
	var testResult interface{}
	err := client.CreateRawTradeBuyLimitTx(nil, &testResult)
	assert.NotNil(t, err)
	assert.Nil(t, testResult)

	token := &ptypes.TradeBuyLimitTx{
		TokenSymbol:       "CNY",
		AmountPerBoardlot: 10,
		MinBoardlot:       1,
		PricePerBoardlot:  100,
		TotalBoardlot:     100,
		Fee:               1,
	}

	err = client.CreateRawTradeBuyLimitTx(token, &testResult)
	assert.NotNil(t, testResult)
	assert.Nil(t, err)

}

func TestChain33_CreateRawTradeSellMarketTx(t *testing.T) {
	client := newTestChain33(nil)
	var testResult interface{}
	err := client.CreateRawTradeSellMarketTx(nil, &testResult)
	assert.NotNil(t, err)
	assert.Nil(t, testResult)

	token := &ptypes.TradeSellMarketTx{
		BuyID:       "12asdfa",
		BoardlotCnt: 100,
		Fee:         1,
	}

	err = client.CreateRawTradeSellMarketTx(token, &testResult)
	assert.NotNil(t, testResult)
	assert.Nil(t, err)

}

func TestChain33_CreateRawTradeRevokeBuyTx(t *testing.T) {
	client := newTestChain33(nil)
	var testResult interface{}
	err := client.CreateRawTradeRevokeBuyTx(nil, &testResult)
	assert.NotNil(t, err)
	assert.Nil(t, testResult)

	token := &ptypes.TradeRevokeBuyTx{
		BuyID: "12asdfa",
		Fee:   1,
	}

	err = client.CreateRawTradeRevokeBuyTx(token, &testResult)
	assert.NotNil(t, testResult)
	assert.Nil(t, err)
}

func TestDecodeLogTradeSellLimit(t *testing.T) {
	var logTmp = &ptypes.ReceiptTradeSellLimit{}
	dec := types.Encode(logTmp)
	strdec := hex.EncodeToString(dec)
	rlog := &rpctypes.ReceiptLog{
		Ty:  types.TyLogTradeSellLimit,
		Log: "0x" + strdec,
	}

	logs := []*rpctypes.ReceiptLog{}
	logs = append(logs, rlog)

	var data = &rpctypes.ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := rpctypes.DecodeLog([]byte("trade"), data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogTradeSell", result.Logs[0].TyName)
}

func TestDecodeLogTradeSellRevoke(t *testing.T) {
	var logTmp = &ptypes.ReceiptTradeBuyMarket{}
	dec := types.Encode(logTmp)
	strdec := hex.EncodeToString(dec)
	rlog := &rpctypes.ReceiptLog{
		Ty:  types.TyLogTradeSellRevoke,
		Log: "0x" + strdec,
	}

	logs := []*rpctypes.ReceiptLog{}
	logs = append(logs, rlog)

	var data = &rpctypes.ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := rpctypes.DecodeLog([]byte("trade"), data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogTradeSellRevoke", result.Logs[0].TyName)
}

func TestDecodeLogTradeBuyLimit(t *testing.T) {
	var logTmp = &ptypes.ReceiptTradeBuyLimit{}
	dec := types.Encode(logTmp)
	strdec := hex.EncodeToString(dec)
	rlog := &rpctypes.ReceiptLog{
		Ty:  types.TyLogTradeBuyLimit,
		Log: "0x" + strdec,
	}

	logs := []*rpctypes.ReceiptLog{}
	logs = append(logs, rlog)

	var data = &rpctypes.ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := rpctypes.DecodeLog([]byte("trade"), data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogTradeBuyLimit", result.Logs[0].TyName)
}

func TestDecodeLogTradeSellMarket(t *testing.T) {
	var logTmp = &ptypes.ReceiptSellMarket{}
	dec := types.Encode(logTmp)
	strdec := hex.EncodeToString(dec)
	rlog := &rpctypes.ReceiptLog{
		Ty:  types.TyLogTradeSellMarket,
		Log: "0x" + strdec,
	}

	logs := []*rpctypes.ReceiptLog{}
	logs = append(logs, rlog)

	var data = &rpctypes.ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := rpctypes.DecodeLog([]byte("trade"), data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogTradeSellMarket", result.Logs[0].TyName)
}

func TestDecodeLogTradeBuyRevoke(t *testing.T) {
	var logTmp = &ptypes.ReceiptTradeBuyRevoke{}
	dec := types.Encode(logTmp)
	strdec := hex.EncodeToString(dec)
	rlog := &rpctypes.ReceiptLog{
		Ty:  types.TyLogTradeBuyRevoke,
		Log: "0x" + strdec,
	}

	logs := []*rpctypes.ReceiptLog{}
	logs = append(logs, rlog)

	var data = &rpctypes.ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := rpctypes.DecodeLog([]byte("trade"), data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogTradeBuyRevoke", result.Logs[0].TyName)
}

func TestDecodeLogTradeBuyMarket(t *testing.T) {
	var logTmp = &ptypes.ReceiptTradeBuyMarket{}
	dec := types.Encode(logTmp)
	strdec := hex.EncodeToString(dec)
	rlog := &rpctypes.ReceiptLog{
		Ty:  types.TyLogTradeBuyMarket,
		Log: "0x" + strdec,
	}

	logs := []*rpctypes.ReceiptLog{}
	logs = append(logs, rlog)

	var data = &rpctypes.ReceiptData{
		Ty:   5,
		Logs: logs,
	}
	result, err := rpctypes.DecodeLog([]byte("trade"), data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "LogTradeBuyMarket", result.Logs[0].TyName)
}

func TestChain33_GetLastMemPoolOk(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	var txlist types.ReplyTxList
	var action ptypes.Trade
	act := types.Encode(&action)
	var tx = &types.Transaction{
		Execer:  []byte(types.ExecName(ptypes.TradeX)),
		Payload: act,
		To:      "to",
	}
	txlist.Txs = append(txlist.Txs, tx)

	// expected := &types.ReqBlocks{}
	api.On("GetLastMempool").Return(&txlist, nil)

	var testResult interface{}
	actual := types.ReqNil{}
	err := testChain33.GetLastMemPool(actual, &testResult)
	assert.Nil(t, err)
	assert.Equal(t, testResult.(*rpctypes.ReplyTxList).Txs[0].Execer, string(tx.Execer))

	mock.AssertExpectationsForObjects(t, api)
}
