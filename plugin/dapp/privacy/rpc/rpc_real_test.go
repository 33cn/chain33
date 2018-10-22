package rpc

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.33.cn/chain33/chain33/client/mocks"
	pty "gitlab.33.cn/chain33/chain33/plugin/dapp/privacy/types"
	"gitlab.33.cn/chain33/chain33/rpc/jsonclient"
	"gitlab.33.cn/chain33/chain33/util/testnode"

	_ "gitlab.33.cn/chain33/chain33/system"
)

func TestRPC_Call(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	cfg, sub := testnode.GetDefaultConfig()
	mock33 := testnode.NewWithConfig(cfg, sub, api)
	defer func() {
		mock33.Close()
		mock.AssertExpectationsForObjects(t, api)
	}()
	g := newGrpc(api)
	g.Init("privacy", mock33.GetRPC(), newJrpc(api), g)
	time.Sleep(time.Millisecond)
	mock33.GetRPC().Listen()
	time.Sleep(time.Millisecond)
	api.On("Close").Return()
	rpcCfg := mock33.GetCfg().Rpc
	jsonClient, err := jsonclient.NewJSONClient("http://" + rpcCfg.JrpcBindAddr + "/")
	assert.Nil(t, err)
	assert.NotNil(t, jsonClient)

	//调用:
	params := pty.ReqPPrivacyAccount{
		Addr:        "addr",
		Token:       "token",
		Displaymode: 3,
	}
	var utxo1 = &pty.UTXO{10, &pty.UTXOBasic{nil, []byte("hello")}}
	var utxo2 = &pty.UTXO{11, &pty.UTXOBasic{nil, []byte("world")}}
	var res = pty.ReplyPrivacyAccount{
		Utxos: &pty.UTXOs{[]*pty.UTXO{utxo1, utxo2}},
	}
	api.On("ExecWalletFunc", "privacy", "ShowPrivacyAccountInfo", &params).Return(&res, nil)
	var result pty.ReplyPrivacyAccount
	err = jsonClient.Call("privacy.ShowPrivacyAccountInfo", params, &result)
	assert.Nil(t, err)
	assert.Equal(t, res, result)
}
