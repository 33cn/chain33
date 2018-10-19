package rpc

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.33.cn/chain33/chain33/client/mocks"
	pty "gitlab.33.cn/chain33/chain33/plugin/dapp/privacy/types"
	rpctypes "gitlab.33.cn/chain33/chain33/rpc/types"
)

//todo: privacy rpc test
//后面可以用我们的mock node 来做测试
func newTestChannelClient() *channelClient {
	api := &mocks.QueueProtocolAPI{}
	return &channelClient{
		ChannelClient: rpctypes.ChannelClient{QueueProtocolAPI: api},
	}
}

func newTestJrpcClient() *Jrpc {
	return &Jrpc{cli: newTestChannelClient()}
}

func newTestJrpcClient2(api *mocks.QueueProtocolAPI) *Jrpc {
	return &Jrpc{cli: &channelClient{
		ChannelClient: rpctypes.ChannelClient{QueueProtocolAPI: api},
	}}
}

func TestChain33_PrivacyTxList(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestJrpcClient2(api)
	actual := &pty.ReqPrivacyTransactionList{}
	api.On("ExecWalletFunc", "privacy", "PrivacyTransactionList", actual).Return(nil, errors.New("error value"))
	var testResult interface{}
	err := testChain33.PrivacyTxList(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}
