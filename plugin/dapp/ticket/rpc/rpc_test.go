package rpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.33.cn/chain33/chain33/client/mocks"
	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/ticket/types"
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

func TestChannelClient_BindMiner(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)

	client := &channelClient{
		ChannelClient: rpctypes.ChannelClient{QueueProtocolAPI: api},
	}
	client.Init("ticket", nil, nil, nil)

	head := &types.Header{StateHash: []byte("sdfadasds")}
	api.On("GetLastHeader").Return(head, nil)

	var acc = &types.Account{Addr: "1Jn2qu84Z1SUUosWjySggBS9pKWdAP3tZt", Balance: 100000 * types.Coin}
	accv := types.Encode(acc)
	storevalue := &types.StoreReplyValue{}
	storevalue.Values = append(storevalue.Values, accv)
	api.On("StoreGet", mock.Anything).Return(storevalue, nil)

	var addrs = make([]string, 1)
	addrs = append(addrs, "1Jn2qu84Z1SUUosWjySggBS9pKWdAP3tZt")
	var in = &ty.ReqBindMiner{
		BindAddr:     "1Jn2qu84Z1SUUosWjySggBS9pKWdAP3tZt",
		OriginAddr:   "1Jn2qu84Z1SUUosWjySggBS9pKWdAP3tZt",
		Amount:       10000 * types.Coin,
		CheckBalance: true,
	}
	_, err := client.CreateBindMiner(context.Background(), in)
	assert.Nil(t, err)
}

func testGetTicketCountOK(t *testing.T) {
	qapi.On("GetTicketCount").Return(nil, nil)
	data, err := g.GetTicketCount(getOkCtx(), nil)
	assert.Nil(t, err, "the error should be nil")
	assert.Nil(t, data)

}

func TestGetTicketCount(t *testing.T) {
	//testGetTicketCountReject(t)
	testGetTicketCountOK(t)
}

func testSetAutoMiningOK(t *testing.T) {
	var in *pb.MinerFlag
	qapi.On("WalletAutoMiner", in).Return(nil, nil)
	data, err := g.SetAutoMining(getOkCtx(), nil)
	assert.Nil(t, err, "the error should be nil")
	assert.Nil(t, data)

}

func TestSetAutoMining(t *testing.T) {
	//testSetAutoMiningReject(t)
	testSetAutoMiningOK(t)
}

func testCloseTicketsOK(t *testing.T) {
	qapi.On("CloseTickets").Return(nil, nil)
	data, err := g.CloseTickets(getOkCtx(), nil)
	assert.Nil(t, err, "the error should be nil")
	assert.Nil(t, data)
}

func TestCloseTickets(t *testing.T) {
	//testCloseTicketsReject(t)
	testCloseTicketsOK(t)
}


var mingResult rpctypes.Reply
api.On("WalletAutoMiner", mock.Anything).Return(&types.Reply{IsOk: true, Msg: []byte("yes")}, nil)
err = jsonClient.Call("Chain33.SetAutoMining", types.MinerFlag{}, &mingResult)
assert.Nil(t, err)
assert.True(t, mingResult.IsOk, "SetAutoMining")

var ticketResult int64
var expectRet = &types.Int64{Data: 100}
api.On("GetTicketCount", mock.Anything).Return(expectRet, nil)
err = jsonClient.Call("Chain33.GetTicketCount", &types.ReqNil{}, &ticketResult)
assert.Nil(t, err)
assert.Equal(t, expectRet.GetData(), ticketResult, "GetTicketCount")