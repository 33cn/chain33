package rpc_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	_ "gitlab.33.cn/chain33/chain33/plugin"
	"gitlab.33.cn/chain33/chain33/rpc/jsonclient"
	_ "gitlab.33.cn/chain33/chain33/system"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/util/testnode"
)

func TestJRPCChannel(t *testing.T) {
	// 启动RPCmocker
	mocker := testnode.New("--notset--", nil)
	defer mocker.Close()
	mocker.Listen()

	jrpcClient := mocker.GetJsonC()

	testCases := []struct {
		fn func(*testing.T, *jsonclient.JSONClient) error
	}{
		{fn: testCountTicketCmd},
		{fn: testCloseTicketCmd},
		{fn: testGetColdAddrByMinerCmd},
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
		t.Log(err.Error())
	}
}

func testCountTicketCmd(t *testing.T, jrpc *jsonclient.JSONClient) error {
	var res types.ReplyHashes
	return jrpc.Call("ticket.CloseTickets", nil, &res)
}

func testCloseTicketCmd(t *testing.T, jrpc *jsonclient.JSONClient) error {
	var res int64
	return jrpc.Call("ticket.GetTicketCount", nil, &res)
}

func testGetColdAddrByMinerCmd(t *testing.T, jrpc *jsonclient.JSONClient) error {
	var rep interface{}
	var params types.Query4Cli
	req := &types.ReqString{}
	params.Execer = "ticket"
	params.FuncName = "MinerSourceList"
	params.Payload = req
	rep = &types.ReplyStrings{}
	return jrpc.Call("Chain33.Query", params, rep)
}
