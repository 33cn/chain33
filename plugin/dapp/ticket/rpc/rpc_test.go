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

func newGrpcClient(api *mocks.QueueProtocolAPI) *channelClient {
	return &channelClient{
		ChannelClient: rpctypes.ChannelClient{QueueProtocolAPI: api},
	}
}

func newJrpcClient(api *mocks.QueueProtocolAPI) *Jrpc {
	return &Jrpc{cli: newGrpcClient(api)}
}

func TestChannelClient_BindMiner(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	client := newGrpcClient(api)
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
	api := &mocks.QueueProtocolAPI{}
	g := newGrpcClient(api)
	api.On("QueryConsensusFunc", "ticket", "GetTicketCount", mock.Anything).Return(&types.Int64{}, nil)
	data, err := g.GetTicketCount(context.Background(), nil)
	assert.Nil(t, err, "the error should be nil")
	assert.Equal(t, data, &types.Int64{})
}

func TestGetTicketCount(t *testing.T) {
	//testGetTicketCountReject(t)
	testGetTicketCountOK(t)
}

func testSetAutoMiningOK(t *testing.T) {
	api := &mocks.QueueProtocolAPI{}
	g := newGrpcClient(api)
	in := &ty.MinerFlag{}
	api.On("ExecWalletFunc", "ticket", "WalletAutoMiner", in).Return(&types.Reply{}, nil)
	data, err := g.SetAutoMining(context.Background(), in)
	assert.Nil(t, err, "the error should be nil")
	assert.Equal(t, data, &types.Reply{})

}

func TestSetAutoMining(t *testing.T) {
	//testSetAutoMiningReject(t)
	testSetAutoMiningOK(t)
}

func testCloseTicketsOK(t *testing.T) {
	api := &mocks.QueueProtocolAPI{}
	g := newGrpcClient(api)
	var in *types.ReqNil
	api.On("ExecWalletFunc", "ticket", "CloseTickets", in).Return(&types.ReplyHashes{}, nil)
	data, err := g.CloseTickets(context.Background(), in)
	assert.Nil(t, err, "the error should be nil")
	assert.Equal(t, data, &types.ReplyHashes{})
}

func TestCloseTickets(t *testing.T) {
	//testCloseTicketsReject(t)
	testCloseTicketsOK(t)
}

func TestJrpc_SetAutoMining(t *testing.T) {
	api := &mocks.QueueProtocolAPI{}
	client := &Jrpc{cli: &channelClient{ChannelClient: rpctypes.ChannelClient{QueueProtocolAPI: api}}}
	var mingResult rpctypes.Reply
	api.On("ExecWalletFunc", mock.Anything, mock.Anything, mock.Anything).Return(&types.Reply{IsOk: true, Msg: []byte("yes")}, nil)
	err := client.SetAutoMining(&ty.MinerFlag{}, &mingResult)
	assert.Nil(t, err)
	assert.True(t, mingResult.IsOk, "SetAutoMining")
}

func TestJrpc_GetTicketCount(t *testing.T) {
	api := &mocks.QueueProtocolAPI{}
	client := &Jrpc{cli: &channelClient{ChannelClient: rpctypes.ChannelClient{QueueProtocolAPI: api}}}

	var ticketResult int64
	var expectRet = &types.Int64{Data: 100}
	api.On("QueryConsensusFunc", mock.Anything, mock.Anything, mock.Anything).Return(expectRet, nil)
	err := client.GetTicketCount(&types.ReqNil{}, &ticketResult)
	assert.Nil(t, err)
	assert.Equal(t, expectRet.GetData(), ticketResult, "GetTicketCount")
}
