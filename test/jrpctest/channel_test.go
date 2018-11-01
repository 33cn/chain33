package jrpctest

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	commonlog "gitlab.33.cn/chain33/chain33/common/log"
	"gitlab.33.cn/chain33/chain33/rpc/jsonclient"
	rpctypes "gitlab.33.cn/chain33/chain33/rpc/types"
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

	jrpcClient, err := mocker.CreateJRPCClient()
	assert.NoError(t, err)
	assert.NotNil(t, jrpcClient)

	testCases := []struct {
		fn func(*testing.T, *jsonclient.JSONClient) error
	}{
		// manager相关的命令测试
		{fn: testQueryConfigCmd},
		// account相关的命令测试
		{fn: testDumpKeyCmd},
		{fn: testGetAccountListCmd},
		{fn: testGetBalanceCmd},
		{fn: testImportKeyCmd},
		{fn: testNewAccountCmd},
		{fn: testSetLabelCmd},
		// block相关的命令测试
		{fn: testGetBlocksCmd},
		{fn: testGetBlockHashCmd},
		{fn: testGetBlockOverviewCmd},
		{fn: testGetHeadersCmd},
		{fn: testGetLastHeaderCmd},
		{fn: testGetBlockByHashsCmd},
		{fn: testGetBlockSequencesCmd},
		{fn: testGetLastBlockSequenceCmd},
		// bty相关的命令测试
		{fn: testCreatePub2PrivTxCmd},
		{fn: testCreatePriv2PrivTxCmd},
		{fn: testCreatePriv2PubTxCmd},
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

func testQueryConfigCmd(t *testing.T, jrpc *jsonclient.JSONClient) error {
	var params types.Query4Cli
	req := &types.ReqString{}
	params.Execer = "manage"
	params.FuncName = "GetConfigItem"
	params.Payload = req
	rep := &types.ReplyConfig{}
	return jrpc.Call("Chain33.Query", params, rep)
}

func testDumpKeyCmd(t *testing.T, jrpc *jsonclient.JSONClient) error {
	params := types.ReqString{}
	var res types.ReplyString
	return jrpc.Call("Chain33.DumpPrivkey", params, &res)
}

func testGetAccountListCmd(t *testing.T, jrpc *jsonclient.JSONClient) error {
	res := &rpctypes.WalletAccounts{}
	return jrpc.Call("Chain33.GetAccounts", nil, &res)
}

func testGetBalanceCmd(t *testing.T, jrpc *jsonclient.JSONClient) error {
	params := types.ReqBalance{}
	var res []*rpctypes.Account
	return jrpc.Call("Chain33.GetBalance", params, &res)
}

func testImportKeyCmd(t *testing.T, jrpc *jsonclient.JSONClient) error {
	params := types.ReqWalletImportPrivkey{}
	var res types.WalletAccount
	return jrpc.Call("Chain33.ImportPrivkey", params, &res)
}

func testNewAccountCmd(t *testing.T, jrpc *jsonclient.JSONClient) error {
	params := types.ReqNewAccount{}
	var res types.WalletAccount
	return jrpc.Call("Chain33.NewAccount", params, &res)
}

func testSetLabelCmd(t *testing.T, jrpc *jsonclient.JSONClient) error {
	params := types.ReqWalletSetLabel{}
	var res types.WalletAccount
	return jrpc.Call("Chain33.SetLabl", params, &res)
}

func testGetBlocksCmd(t *testing.T, jrpc *jsonclient.JSONClient) error {
	params := rpctypes.BlockParam{}
	var res rpctypes.BlockDetails
	return jrpc.Call("Chain33.GetBlocks", params, &res)
}

func testGetBlockHashCmd(t *testing.T, jrpc *jsonclient.JSONClient) error {
	params := types.ReqInt{}
	var res rpctypes.ReplyHash
	return jrpc.Call("Chain33.GetBlockHash", params, &res)
}

func testGetBlockOverviewCmd(t *testing.T, jrpc *jsonclient.JSONClient) error {
	params := rpctypes.QueryParm{}
	var res rpctypes.BlockOverview
	return jrpc.Call("Chain33.GetBlockOverview", params, &res)
}

func testGetHeadersCmd(t *testing.T, jrpc *jsonclient.JSONClient) error {
	params := types.ReqBlocks{}
	var res rpctypes.Headers
	return jrpc.Call("Chain33.GetHeaders", params, &res)
}

func testGetLastHeaderCmd(t *testing.T, jrpc *jsonclient.JSONClient) error {
	var res rpctypes.Header
	return jrpc.Call("Chain33.GetLastHeader", nil, &res)
}

func testGetBlockByHashsCmd(t *testing.T, jrpc *jsonclient.JSONClient) error {
	params := rpctypes.ReqHashes{}
	var res types.BlockDetails
	return jrpc.Call("Chain33.GetBlockByHashes", params, &res)
}

func testGetBlockSequencesCmd(t *testing.T, jrpc *jsonclient.JSONClient) error {
	params := rpctypes.BlockParam{}
	var res rpctypes.ReplyBlkSeqs
	return jrpc.Call("Chain33.GetBlockSequences", params, &res)
}

func testGetLastBlockSequenceCmd(t *testing.T, jrpc *jsonclient.JSONClient) error {
	var res int64
	return jrpc.Call("Chain33.GetLastBlockSequence", nil, &res)
}

func testCreatePub2PrivTxCmd(t *testing.T, jrpc *jsonclient.JSONClient) error {
	params := types.ReqCreateTransaction{}
	return jrpc.Call("privacy.CreateRawTransaction", params, nil)
}

func testCreatePriv2PrivTxCmd(t *testing.T, jrpc *jsonclient.JSONClient) error {
	params := types.ReqCreateTransaction{}
	return jrpc.Call("privacy.CreateRawTransaction", params, nil)
}

func testCreatePriv2PubTxCmd(t *testing.T, jrpc *jsonclient.JSONClient) error {
	params := types.ReqCreateTransaction{}
	return jrpc.Call("privacy.CreateRawTransaction", params, nil)
}
