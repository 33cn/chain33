package rpc

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.33.cn/chain33/chain33/types"
)

func TestTokenPayloadType(t *testing.T) {
	msg, err := tokenPayloadType("GetTokens")
	assert.Equal(t, &types.ReqTokens{}, msg)
	assert.Nil(t, err)

	msg, err = tokenPayloadType("GetTokenInfo")
	assert.Equal(t, &types.ReqString{}, msg)
	assert.Nil(t, err)

	msg, err = tokenPayloadType("GetAddrReceiverforTokens")
	assert.Equal(t, &types.ReqAddrTokens{}, msg)
	assert.Nil(t, err)

	msg, err = tokenPayloadType("GetAccountTokenAssets")
	assert.Equal(t, &types.ReqAccountTokenAssets{}, msg)
	assert.Nil(t, err)

	msg, err = tokenPayloadType("wzw")
	assert.Nil(t, msg)
	assert.Equal(t, err, types.ErrInputPara)
}

func TestCoinsPayloadType(t *testing.T) {
	msg, err := coinsPayloadType("GetAddrReciver")
	assert.Equal(t, &types.ReqAddr{}, msg)
	assert.Nil(t, err)

	msg, err = coinsPayloadType("GetTxsByAddr")
	assert.Equal(t, &types.ReqAddr{}, msg)
	assert.Nil(t, err)

	msg, err = coinsPayloadType("wzw")
	assert.Nil(t, msg)
	assert.Equal(t, err, types.ErrInputPara)
}

func TestManagePayloadType(t *testing.T) {
	msg, err := managePayloadType("GetConfigItem")
	assert.Equal(t, &types.ReqString{}, msg)
	assert.Nil(t, err)

	msg, err = managePayloadType("wzw")
	assert.Nil(t, msg)
	assert.Equal(t, err, types.ErrInputPara)
}

func TestRetrievePayloadType(t *testing.T) {
	msg, err := retrievePayloadType("GetRetrieveInfo")
	assert.Equal(t, &types.ReqRetrieveInfo{}, msg)
	assert.Nil(t, err)

	msg, err = retrievePayloadType("wzw")
	assert.Nil(t, msg)
	assert.Equal(t, err, types.ErrInputPara)
}

func TestTicketPayloadType(t *testing.T) {
	msg, err := ticketPayloadType("TicketInfos")
	assert.Equal(t, &types.TicketInfos{}, msg)
	assert.Nil(t, err)

	msg, err = ticketPayloadType("TicketList")
	assert.Equal(t, &types.TicketList{}, msg)
	assert.Nil(t, err)

	msg, err = ticketPayloadType("MinerAddress")
	assert.Equal(t, &types.ReqString{}, msg)
	assert.Nil(t, err)

	msg, err = ticketPayloadType("MinerSourceList")
	assert.Equal(t, &types.ReqString{}, msg)
	assert.Nil(t, err)

	msg, err = ticketPayloadType("wzw")
	assert.Nil(t, msg)
	assert.Equal(t, err, types.ErrInputPara)
}

func TestTradePayloadType(t *testing.T) {
	msg, err := tradePayloadType("GetOnesSellOrder")
	assert.Equal(t, &types.ReqAddrTokens{}, msg)
	assert.Nil(t, err)

	msg, err = tradePayloadType("GetOnesBuyOrder")
	assert.Equal(t, &types.ReqAddrTokens{}, msg)
	assert.Nil(t, err)

	msg, err = tradePayloadType("GetOnesSellOrderWithStatus")
	assert.Equal(t, &types.ReqAddrTokens{}, msg)
	assert.Nil(t, err)

	msg, err = tradePayloadType("GetOnesBuyOrderWithStatus")
	assert.Equal(t, &types.ReqAddrTokens{}, msg)
	assert.Nil(t, err)

	msg, err = tradePayloadType("GetTokenSellOrderByStatus")
	assert.Equal(t, &types.ReqTokenSellOrder{}, msg)
	assert.Nil(t, err)

	msg, err = tradePayloadType("GetTokenBuyOrderByStatus")
	assert.Equal(t, &types.ReqTokenBuyOrder{}, msg)
	assert.Nil(t, err)

	msg, err = tradePayloadType("wzw")
	assert.Nil(t, msg)
	assert.Equal(t, err, types.ErrInputPara)
}

func TestPayloadType(t *testing.T) {
	msg, err := payloadType(types.ExecName(types.TokenX), "GetTokens")
	assert.Equal(t, &types.ReqTokens{}, msg)
	assert.Nil(t, err)

	msg, err = payloadType(types.ExecName(types.CoinsX), "GetAddrReciver")
	assert.Equal(t, &types.ReqAddr{}, msg)
	assert.Nil(t, err)

	msg, err = payloadType(types.ExecName(types.ManageX), "GetConfigItem")
	assert.Equal(t, &types.ReqString{}, msg)
	assert.Nil(t, err)

	msg, err = payloadType(types.ExecName(types.RetrieveX), "GetRetrieveInfo")
	assert.Equal(t, &types.ReqRetrieveInfo{}, msg)
	assert.Nil(t, err)

	msg, err = payloadType(types.ExecName(types.TicketX), "TicketInfos")
	assert.Equal(t, &types.TicketInfos{}, msg)
	assert.Nil(t, err)

	msg, err = payloadType(types.ExecName(types.TradeX), "GetOnesSellOrder")
	assert.Equal(t, &types.ReqAddrTokens{}, msg)
	assert.Nil(t, err)

	msg, err = payloadType(types.ExecName("wzw"), "wzw")
	assert.Nil(t, msg)
	assert.Equal(t, err, types.ErrInputPara)
}

func TestProtoPayload(t *testing.T) {
	msg, err := protoPayload(types.ExecName(types.TokenX), "GetTokens", nil)
	assert.Equal(t, types.ErrInputPara, err)
	assert.Nil(t, msg)

	var tokens = make([]string, 1)
	tokens[0] = "CNY"
	var token = &types.ReqTokens{
		QueryAll: true,
		Status:   1,
		Tokens:   tokens,
	}

	data, err := json.Marshal(token)
	assert.Nil(t, err)
	assert.NotNil(t, data)

	d := json.RawMessage(data)
	msg, err = protoPayload(types.ExecName(types.TokenX), "GetTokens", &d)
	assert.NotNil(t, data)
	assert.Nil(t, err)

	msg, err = protoPayload("wzw", "wzw", nil)
	assert.Equal(t, types.ErrInputPara, err)
	assert.Nil(t, msg)

}
