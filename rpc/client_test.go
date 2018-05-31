package rpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.33.cn/chain33/chain33/client/mocks"
	qmock "gitlab.33.cn/chain33/chain33/queue/mocks"
)

func newTestChannelClient() *channelClient {
	return &channelClient{
		QueueProtocolAPI: &mocks.QueueProtocolAPI{},
	}
}

// TODO
func TestInit(t *testing.T) {
	client := newTestChannelClient()
	client.Init(&qmock.Client{})
}

func TestChannelClient_CreateRawTokenPreCreateTx(t *testing.T) {
	client := newTestChannelClient()
	data, err := client.CreateRawTokenPreCreateTx(nil)
	assert.NotNil(t, err)
	assert.Nil(t, data)

	token := &TokenPreCreateTx{
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

	token := &TokenRevokeTx{
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

	token := &TokenFinishTx{
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

	token := &TradeSellTx{
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

	token := &TradeBuyTx{
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

	token := &TradeRevokeTx{
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

	token := &TradeBuyLimitTx{
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

	token := &TradeSellMarketTx{
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

	token := &TradeRevokeBuyTx{
		BuyID: "12asdfa",
		Fee:   1,
	}
	data, err = client.CreateRawTradeRevokeBuyTx(token)
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
