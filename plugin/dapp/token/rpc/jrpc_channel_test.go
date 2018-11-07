package rpc_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	pty "gitlab.33.cn/chain33/chain33/plugin/dapp/token/types"
	"gitlab.33.cn/chain33/chain33/rpc/jsonclient"
	rpctypes "gitlab.33.cn/chain33/chain33/rpc/types"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/util/testnode"

	_ "gitlab.33.cn/chain33/chain33/plugin"
	_ "gitlab.33.cn/chain33/chain33/system"
)

func TestJRPCChannel(t *testing.T) {
	// 启动RPCmocker
	mocker := testnode.New("--notset--", nil)
	defer func() {
		mocker.Close()
	}()
	mocker.Listen()

	jrpcClient := mocker.GetJsonC()

	testCases := []struct {
		fn func(*testing.T, *jsonclient.JSONClient) error
	}{
		{fn: testGetTokensPreCreatedCmd},
		{fn: testGetTokensFinishCreatedCmd},
		{fn: testGetTokenAssetsCmd},
		{fn: testGetTokenBalanceCmd},
		{fn: testCreateRawTokenPreCreateTxCmd},
		{fn: testCreateRawTokenFinishTxCmd},
		{fn: testCreateRawTokenRevokeTxCmd},
	}
	for index, testCase := range testCases {
		err := testCase.fn(t, jrpcClient)
		if err == nil {
			continue
		}
		assert.NotEqualf(t, err, types.ErrActionNotSupport, "test index %d", index)
		if strings.Contains(err.Error(), "rpc: can't find") {
			assert.FailNowf(t, err.Error(), "test index %d", index)
		}
	}
}

func testGetTokensPreCreatedCmd(t *testing.T, jrpc *jsonclient.JSONClient) error {
	var rep interface{}
	var params types.Query4Cli
	req := &pty.ReqTokens{}
	params.Execer = "token"
	params.FuncName = "GetTokens"
	params.Payload = req
	rep = &pty.ReplyTokens{}
	return jrpc.Call("Chain33.Query", params, rep)
}

func testGetTokensFinishCreatedCmd(t *testing.T, jrpc *jsonclient.JSONClient) error {
	params := types.Query4Cli{
		Execer:   "token",
		FuncName: "GetTokens",
		Payload:  pty.ReqTokens{},
	}
	var res pty.ReplyTokens
	return jrpc.Call("Chain33.Query", params, &res)
}

func testGetTokenAssetsCmd(t *testing.T, jrpc *jsonclient.JSONClient) error {
	var rep interface{}
	var params types.Query4Cli
	req := &pty.ReqAccountTokenAssets{}
	params.Execer = "token"
	params.FuncName = "GetAccountTokenAssets"
	params.Payload = req
	rep = &pty.ReplyAccountTokenAssets{}
	return jrpc.Call("Chain33.Query", params, rep)
}

func testGetTokenBalanceCmd(t *testing.T, jrpc *jsonclient.JSONClient) error {
	params := pty.ReqTokenBalance{}
	var res []*rpctypes.Account
	return jrpc.Call("token.GetTokenBalance", params, &res)
}

func testCreateRawTokenPreCreateTxCmd(t *testing.T, jrpc *jsonclient.JSONClient) error {
	params := pty.TokenPreCreate{}
	return jrpc.Call("token.CreateRawTokenPreCreateTx", params, nil)
}

func testCreateRawTokenFinishTxCmd(t *testing.T, jrpc *jsonclient.JSONClient) error {
	params := pty.TokenRevokeCreate{}
	return jrpc.Call("token.CreateRawTokenRevokeTx", params, nil)
}

func testCreateRawTokenRevokeTxCmd(t *testing.T, jrpc *jsonclient.JSONClient) error {
	params := pty.TokenFinishCreate{}
	return jrpc.Call("token.CreateRawTokenFinishTx", params, nil)
}
