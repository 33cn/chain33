package rpc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.33.cn/chain33/chain33/client/mocks"
	ptypes "gitlab.33.cn/chain33/chain33/plugin/dapp/trade/types"
	rpctypes "gitlab.33.cn/chain33/chain33/rpc/types"
)

func newTestChannelClient() *Grpc {
	cli := &channelClient{
		ChannelClient: rpctypes.ChannelClient{
			QueueProtocolAPI: &mocks.QueueProtocolAPI{},
		},
	}
	return &Grpc{channelClient: cli}
}

func TestChannelClient_CreateRawTradeSellTx(t *testing.T) {
	client := newTestChannelClient()
	data, err := client.CreateRawTradeSellTx(context.Background(), nil)
	assert.NotNil(t, err)
	assert.Nil(t, data)

	token := &ptypes.TradeForSell{
		TokenSymbol:       "CNY",
		AmountPerBoardlot: 10,
		MinBoardlot:       1,
		PricePerBoardlot:  100,
		TotalBoardlot:     100,
	}
	data, err = client.CreateRawTradeSellTx(context.Background(), token)
	assert.NotNil(t, data)
	assert.Nil(t, err)
}

func TestChannelClient_CreateRawTradeBuyTx(t *testing.T) {
	client := newTestChannelClient()
	data, err := client.CreateRawTradeBuyTx(context.Background(), nil)
	assert.NotNil(t, err)
	assert.Nil(t, data)

	token := &ptypes.TradeForBuy{
		SellID:      "sadfghjkhgfdsa",
		BoardlotCnt: 100,
	}
	data, err = client.CreateRawTradeBuyTx(context.Background(), token)
	assert.NotNil(t, data)
	assert.Nil(t, err)
}

func TestChannelClient_CreateRawTradeRevokeTx(t *testing.T) {
	client := newTestChannelClient()
	data, err := client.CreateRawTradeRevokeTx(context.Background(), nil)
	assert.NotNil(t, err)
	assert.Nil(t, data)

	token := &ptypes.TradeForRevokeSell{
		SellID: "sadfghjkhgfdsa",
	}
	data, err = client.CreateRawTradeRevokeTx(context.Background(), token)
	assert.NotNil(t, data)
	assert.Nil(t, err)
}

func TestChannelClient_CreateRawTradeBuyLimitTx(t *testing.T) {
	client := newTestChannelClient()
	data, err := client.CreateRawTradeBuyLimitTx(context.Background(), nil)
	assert.NotNil(t, err)
	assert.Nil(t, data)

	token := &ptypes.TradeForBuyLimit{
		TokenSymbol:       "CNY",
		AmountPerBoardlot: 10,
		MinBoardlot:       1,
		PricePerBoardlot:  100,
		TotalBoardlot:     100,
	}
	data, err = client.CreateRawTradeBuyLimitTx(context.Background(), token)
	assert.NotNil(t, data)
	assert.Nil(t, err)
}

func TestChannelClient_CreateRawTradeSellMarketTx(t *testing.T) {
	client := newTestChannelClient()
	data, err := client.CreateRawTradeSellMarketTx(context.Background(), nil)
	assert.NotNil(t, err)
	assert.Nil(t, data)

	token := &ptypes.TradeForSellMarket{
		BuyID:       "12asdfa",
		BoardlotCnt: 100,
	}
	data, err = client.CreateRawTradeSellMarketTx(context.Background(), token)
	assert.NotNil(t, data)
	assert.Nil(t, err)
}

func TestChannelClient_CreateRawTradeRevokeBuyTx(t *testing.T) {
	client := newTestChannelClient()
	data, err := client.CreateRawTradeRevokeBuyTx(context.Background(), nil)
	assert.NotNil(t, err)
	assert.Nil(t, data)

	token := &ptypes.TradeForRevokeBuy{
		BuyID: "12asdfa",
	}
	data, err = client.CreateRawTradeRevokeBuyTx(context.Background(), token)
	assert.NotNil(t, data)
	assert.Nil(t, err)
}
