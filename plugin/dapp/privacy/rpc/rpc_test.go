package rpc

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/client/mocks"
	pty "gitlab.33.cn/chain33/chain33/plugin/dapp/privacy/types"
	rpctypes "gitlab.33.cn/chain33/chain33/rpc/types"
)

func newGrpc(api client.QueueProtocolAPI) *channelClient {
	return &channelClient{
		ChannelClient: rpctypes.ChannelClient{QueueProtocolAPI: api},
	}
}

func newJrpc(api client.QueueProtocolAPI) *Jrpc {
	return &Jrpc{cli: newGrpc(api)}
}

func TestChain33_PrivacyTxList(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newJrpc(api)
	actual := &pty.ReqPrivacyTransactionList{}
	api.On("ExecWalletFunc", "privacy", "PrivacyTransactionList", actual).Return(nil, errors.New("error value"))
	var testResult interface{}
	err := testChain33.PrivacyTxList(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}
