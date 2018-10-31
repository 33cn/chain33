package rpc

import (
	"testing"
	"time"

	"gitlab.33.cn/chain33/chain33/types"

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
	mock33.Listen()
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
	var utxo1 = &pty.UTXO{10, &pty.UTXOBasic{&pty.UTXOGlobalIndex{[]byte("hash1"), 1}, []byte("hello")}}
	var utxo2 = &pty.UTXO{11, &pty.UTXOBasic{&pty.UTXOGlobalIndex{[]byte("hash2"), 2}, []byte("world")}}
	var res = pty.ReplyPrivacyAccount{
		Utxos: &pty.UTXOs{[]*pty.UTXO{utxo1, utxo2}},
	}
	api.On("ExecWalletFunc", "privacy", "ShowPrivacyAccountInfo", &params).Return(&res, nil)
	var result pty.ReplyPrivacyAccount
	err = jsonClient.Call("privacy.ShowPrivacyAccountInfo", params, &result)
	assert.Nil(t, err)
	assert.Equal(t, res, result)
}

func TestJsonToPB(t *testing.T) {
	jsonStr := `{"utxoHaveTxHashs":[{"amount":"200000000","txHash":"230ef9ecd3be28b1b2b22d2c3aba58aa72b90dc0231e5922609ad9f5f75a5fcf","utxoBasic":{"utxoGlobalIndex":{"txhash":"0x230ef9ecd3be28b1b2b22d2c3aba58aa72b90dc0231e5922609ad9f5f75a5fcf","outindex":0},"onetimePubkey":"0xa82a79ccf75f67e15c45905a651ad55bb08d935f39a353079f8576bb00722d15"}},{"amount":"200000000","txHash":"230ef9ecd3be28b1b2b22d2c3aba58aa72b90dc0231e5922609ad9f5f75a5fcf","utxoBasic":{"utxoGlobalIndex":{"txhash":"0x230ef9ecd3be28b1b2b22d2c3aba58aa72b90dc0231e5922609ad9f5f75a5fcf","outindex":0},"onetimePubkey":"0xa82a79ccf75f67e15c45905a651ad55bb08d935f39a353079f8576bb00722d15"}},{"amount":"200000000","txHash":"230ef9ecd3be28b1b2b22d2c3aba58aa72b90dc0231e5922609ad9f5f75a5fcf","utxoBasic":{"utxoGlobalIndex":{"txhash":"0x230ef9ecd3be28b1b2b22d2c3aba58aa72b90dc0231e5922609ad9f5f75a5fcf","outindex":0},"onetimePubkey":"0xa82a79ccf75f67e15c45905a651ad55bb08d935f39a353079f8576bb00722d15"}},{"amount":"200000000","txHash":"230ef9ecd3be28b1b2b22d2c3aba58aa72b90dc0231e5922609ad9f5f75a5fcf","utxoBasic":{"utxoGlobalIndex":{"txhash":"0x230ef9ecd3be28b1b2b22d2c3aba58aa72b90dc0231e5922609ad9f5f75a5fcf","outindex":0},"onetimePubkey":"0xa82a79ccf75f67e15c45905a651ad55bb08d935f39a353079f8576bb00722d15"}},{"amount":"200000000","txHash":"230ef9ecd3be28b1b2b22d2c3aba58aa72b90dc0231e5922609ad9f5f75a5fcf","utxoBasic":{"utxoGlobalIndex":{"txhash":"0x230ef9ecd3be28b1b2b22d2c3aba58aa72b90dc0231e5922609ad9f5f75a5fcf","outindex":0},"onetimePubkey":"0xa82a79ccf75f67e15c45905a651ad55bb08d935f39a353079f8576bb00722d15"}},{"amount":"200000000","txHash":"230ef9ecd3be28b1b2b22d2c3aba58aa72b90dc0231e5922609ad9f5f75a5fcf","utxoBasic":{"utxoGlobalIndex":{"txhash":"0x230ef9ecd3be28b1b2b22d2c3aba58aa72b90dc0231e5922609ad9f5f75a5fcf","outindex":0},"onetimePubkey":"0xa82a79ccf75f67e15c45905a651ad55bb08d935f39a353079f8576bb00722d15"}},{"amount":"200000000","txHash":"230ef9ecd3be28b1b2b22d2c3aba58aa72b90dc0231e5922609ad9f5f75a5fcf","utxoBasic":{"utxoGlobalIndex":{"txhash":"0x230ef9ecd3be28b1b2b22d2c3aba58aa72b90dc0231e5922609ad9f5f75a5fcf","outindex":0},"onetimePubkey":"0xa82a79ccf75f67e15c45905a651ad55bb08d935f39a353079f8576bb00722d15"}},{"amount":"200000000","txHash":"230ef9ecd3be28b1b2b22d2c3aba58aa72b90dc0231e5922609ad9f5f75a5fcf","utxoBasic":{"utxoGlobalIndex":{"txhash":"0x230ef9ecd3be28b1b2b22d2c3aba58aa72b90dc0231e5922609ad9f5f75a5fcf","outindex":0},"onetimePubkey":"0xa82a79ccf75f67e15c45905a651ad55bb08d935f39a353079f8576bb00722d15"}},{"amount":"200000000","txHash":"230ef9ecd3be28b1b2b22d2c3aba58aa72b90dc0231e5922609ad9f5f75a5fcf","utxoBasic":{"utxoGlobalIndex":{"txhash":"0x230ef9ecd3be28b1b2b22d2c3aba58aa72b90dc0231e5922609ad9f5f75a5fcf","outindex":0},"onetimePubkey":"0xa82a79ccf75f67e15c45905a651ad55bb08d935f39a353079f8576bb00722d15"}},{"amount":"200000000","txHash":"230ef9ecd3be28b1b2b22d2c3aba58aa72b90dc0231e5922609ad9f5f75a5fcf","utxoBasic":{"utxoGlobalIndex":{"txhash":"0x230ef9ecd3be28b1b2b22d2c3aba58aa72b90dc0231e5922609ad9f5f75a5fcf","outindex":0},"onetimePubkey":"0xa82a79ccf75f67e15c45905a651ad55bb08d935f39a353079f8576bb00722d15"}},{"amount":"200000000","txHash":"230ef9ecd3be28b1b2b22d2c3aba58aa72b90dc0231e5922609ad9f5f75a5fcf","utxoBasic":{"utxoGlobalIndex":{"txhash":"0x230ef9ecd3be28b1b2b22d2c3aba58aa72b90dc0231e5922609ad9f5f75a5fcf","outindex":0},"onetimePubkey":"0xa82a79ccf75f67e15c45905a651ad55bb08d935f39a353079f8576bb00722d15"}},{"amount":"200000000","txHash":"230ef9ecd3be28b1b2b22d2c3aba58aa72b90dc0231e5922609ad9f5f75a5fcf","utxoBasic":{"utxoGlobalIndex":{"txhash":"0x230ef9ecd3be28b1b2b22d2c3aba58aa72b90dc0231e5922609ad9f5f75a5fcf","outindex":0},"onetimePubkey":"0xa82a79ccf75f67e15c45905a651ad55bb08d935f39a353079f8576bb00722d15"}},{"amount":"500000000","txHash":"aa50a7c0b83d9c1eb9edb3f8da4d52df0bbc337db7aff4d1ae165bfa5f7a13fe","utxoBasic":{"utxoGlobalIndex":{"txhash":"0xf70df4c801db3e3759d20ccb8e4565906691680e411bef2a73ddbf53115d4d26","outindex":13},"onetimePubkey":"0x42353b06625107408ffc72dc59e101ea8e2f144d0364caff2e649e782533fdc0"}},{"amount":"100000000","txHash":"bd4f3e9963b762b769f5c40a9afe83fa968e2b1e936682ecc02b07958c321305","utxoBasic":{"utxoGlobalIndex":{"txhash":"0x2c4aa7aea82de4a971bceb6cfef3d09dbeac7c7df3a4b49b5a311d23d772f027","outindex":2},"onetimePubkey":"0xd13f979b2b4e269c738413e810aa3b30c3e6cd600573f7287202403198179a03"}},{"amount":"500000000","txHash":"bd4f3e9963b762b769f5c40a9afe83fa968e2b1e936682ecc02b07958c321305","utxoBasic":{"utxoGlobalIndex":{"txhash":"0xf70df4c801db3e3759d20ccb8e4565906691680e411bef2a73ddbf53115d4d26","outindex":5},"onetimePubkey":"0xe3605ab776be46c1baee4e0377bd284c51320d48d93c514f301624078086fc8c"}},{"amount":"500000000","txHash":"e6e98297f741d8649c9a520c494b9abab6803a2179f8f0eaa19614571fc709cf","utxoBasic":{"utxoGlobalIndex":{"txhash":"0xf70df4c801db3e3759d20ccb8e4565906691680e411bef2a73ddbf53115d4d26","outindex":12},"onetimePubkey":"0x370bd63e8d6bae4f004f21d5fdbd0ad0b0cbd4052968f75332dfb9920a963f52"}}]}`

	var res pty.UTXOHaveTxHashs
	err := types.JsonToPB([]byte(jsonStr), &res)
	assert.NoError(t, err)
}
