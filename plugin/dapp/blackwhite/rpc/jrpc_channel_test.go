package rpc_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	commonlog "gitlab.33.cn/chain33/chain33/common/log"
	pty "gitlab.33.cn/chain33/chain33/plugin/dapp/blackwhite/types"
	"gitlab.33.cn/chain33/chain33/rpc/jsonclient"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/util/testnode"

	_ "gitlab.33.cn/chain33/chain33/plugin"
	_ "gitlab.33.cn/chain33/chain33/system"
)

func init() {
	commonlog.SetLogLevel("error")
}

func TestJRPCChannel(t *testing.T) {
	// 启动RPCmocker
	mocker := testnode.New("--notset--", nil)
	defer func() {
		mocker.Close()
	}()
	mocker.Listen()

	jrpcClient := mocker.GetJsonC()
	assert.NotNil(t, jrpcClient)

	testCases := []struct {
		fn func(*testing.T, *jsonclient.JSONClient) error
	}{
		{fn: testCreateRawTxCmd},
		{fn: testPlayRawTxCmd},
		{fn: testShowRawTxCmd},
		{fn: testTimeoutDoneTxCmd},
		{fn: testRoundInfoCmd},
		{fn: testRoundListCmd},
		{fn: testLoopResultCmd},
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

func testCreateRawTxCmd(t *testing.T, jrpc *jsonclient.JSONClient) error {
	params := &pty.BlackwhiteCreateTxReq{}
	var res string
	return jrpc.Call("blackwhite.BlackwhiteCreateTx", params, &res)
}

func testPlayRawTxCmd(t *testing.T, jrpc *jsonclient.JSONClient) error {
	params := &pty.BlackwhitePlayTxReq{}
	var res string
	return jrpc.Call("blackwhite.BlackwhitePlayTx", params, &res)
}

func testShowRawTxCmd(t *testing.T, jrpc *jsonclient.JSONClient) error {
	params := &pty.BlackwhiteShowTxReq{}
	var res string
	return jrpc.Call("blackwhite.BlackwhiteShowTx", params, &res)
}

func testTimeoutDoneTxCmd(t *testing.T, jrpc *jsonclient.JSONClient) error {
	params := &pty.BlackwhiteTimeoutDoneTxReq{}
	var res string
	return jrpc.Call("blackwhite.BlackwhiteTimeoutDoneTx", params, &res)
}

func testRoundInfoCmd(t *testing.T, jrpc *jsonclient.JSONClient) error {
	var rep interface{}
	var params types.Query4Cli
	req := &pty.ReqBlackwhiteRoundInfo{}
	params.FuncName = pty.GetBlackwhiteRoundInfo
	params.Payload = req
	rep = &pty.ReplyBlackwhiteRoundInfo{}
	return jrpc.Call("Chain33.Query", params, rep)
}

func testRoundListCmd(t *testing.T, jrpc *jsonclient.JSONClient) error {
	var rep interface{}
	var params types.Query4Cli
	req := &pty.ReqBlackwhiteRoundList{}
	params.FuncName = pty.GetBlackwhiteByStatusAndAddr
	params.Payload = req
	rep = &pty.ReplyBlackwhiteRoundList{}

	return jrpc.Call("Chain33.Query", params, rep)
}

func testLoopResultCmd(t *testing.T, jrpc *jsonclient.JSONClient) error {
	var rep interface{}
	var params types.Query4Cli
	req := &pty.ReqLoopResult{}
	params.FuncName = pty.GetBlackwhiteloopResult
	params.Payload = req
	rep = &pty.ReplyLoopResults{}

	return jrpc.Call("Chain33.Query", params, rep)
}
