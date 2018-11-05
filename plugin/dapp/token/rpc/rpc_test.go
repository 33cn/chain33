package rpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.33.cn/chain33/chain33/client/mocks"
	tokenty "gitlab.33.cn/chain33/chain33/plugin/dapp/token/types"
	rpctypes "gitlab.33.cn/chain33/chain33/rpc/types"
	"gitlab.33.cn/chain33/chain33/types"
	context "golang.org/x/net/context"
)

func newTestChannelClient() *channelClient {
	api := &mocks.QueueProtocolAPI{}
	return &channelClient{
		ChannelClient: rpctypes.ChannelClient{QueueProtocolAPI: api},
	}
}

func newTestJrpcClient() *Jrpc {
	return &Jrpc{cli: newTestChannelClient()}
}

func testChannelClient_GetTokenBalanceToken(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)

	client := &channelClient{
		ChannelClient: rpctypes.ChannelClient{QueueProtocolAPI: api},
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
	var in = &tokenty.ReqTokenBalance{
		Execer:      types.ExecName(tokenty.TokenX),
		Addresses:   addrs,
		TokenSymbol: "xxx",
	}
	data, err := client.GetTokenBalance(context.Background(), in)
	assert.Nil(t, err)
	accounts := data.Acc
	assert.Equal(t, acc.Addr, accounts[0].Addr)

}

func testChannelClient_GetTokenBalanceOther(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	client := &channelClient{
		ChannelClient: rpctypes.ChannelClient{QueueProtocolAPI: api},
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
	var in = &tokenty.ReqTokenBalance{
		Execer:      types.ExecName("trade"),
		Addresses:   addrs,
		TokenSymbol: "xxx",
	}
	data, err := client.GetTokenBalance(context.Background(), in)
	assert.Nil(t, err)
	accounts := data.Acc
	assert.Equal(t, acc.Addr, accounts[0].Addr)

}

func TestChannelClient_GetTokenBalance(t *testing.T) {
	testChannelClient_GetTokenBalanceToken(t)
	testChannelClient_GetTokenBalanceOther(t)

}

func TestChannelClient_CreateRawTokenPreCreateTx(t *testing.T) {
	client := newTestJrpcClient()
	var data interface{}
	err := client.CreateRawTokenPreCreateTx(nil, &data)
	assert.NotNil(t, err)
	assert.Nil(t, data)

	token := &tokenty.TokenPreCreate{
		Owner:  "asdf134",
		Symbol: "CNY",
	}
	err = client.CreateRawTokenPreCreateTx(token, &data)
	assert.NotNil(t, data)
	assert.Nil(t, err)
}

func TestChannelClient_CreateRawTokenRevokeTx(t *testing.T) {
	client := newTestJrpcClient()
	var data interface{}
	err := client.CreateRawTokenRevokeTx(nil, &data)
	assert.NotNil(t, err)
	assert.Nil(t, data)

	token := &tokenty.TokenRevokeCreate{
		Owner:  "asdf134",
		Symbol: "CNY",
	}
	err = client.CreateRawTokenRevokeTx(token, &data)
	assert.NotNil(t, data)
	assert.Nil(t, err)
}

func TestChannelClient_CreateRawTokenFinishTx(t *testing.T) {
	client := newTestJrpcClient()
	var data interface{}
	err := client.CreateRawTokenFinishTx(nil, &data)
	assert.NotNil(t, err)
	assert.Nil(t, data)

	token := &tokenty.TokenFinishCreate{
		Owner:  "asdf134",
		Symbol: "CNY",
	}
	err = client.CreateRawTokenFinishTx(token, &data)
	assert.NotNil(t, data)
	assert.Nil(t, err)
}
