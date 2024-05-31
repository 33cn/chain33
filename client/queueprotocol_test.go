// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client_test

import (
	"os"
	"testing"
	"time"

	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/common/version"
	"github.com/33cn/chain33/pluginmgr"
	"github.com/33cn/chain33/queue"
	rpctypes "github.com/33cn/chain33/rpc/types"
	_ "github.com/33cn/chain33/system"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	mock     mockSystem
	api      client.QueueProtocolAPI
	jrpc     mockJRPCSystem
	grpcMock mockGRPCSystem
)

func TestMain(m *testing.M) {
	mock.grpcMock = &grpcMock
	mock.jrpcMock = &jrpc
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	pluginmgr.InitExec(cfg)
	api = mock.startup(0)
	flag := m.Run()
	mock.stop()
	os.Exit(flag)
}

func TestQueueProtocolAPI(t *testing.T) {
	var option client.QueueProtocolOption
	option.SendTimeout = time.Millisecond
	option.WaitTimeout = 100 * time.Millisecond

	_, err := client.New(nil, nil)
	if err == nil {
		t.Error("client.New(nil, nil) need return error")
	}

	var q = queue.New("channel")
	q.SetConfig(types.NewChain33Config(types.GetDefaultCfgstring()))
	qc, err := client.New(q.Client(), &option)
	if err != nil {
		t.Errorf("client.New() cause error %v", err)
	}
	if qc == nil {
		t.Error("queueprotoapi object is nil")
	}
	_, err = qc.Notify("", 1, "data")
	assert.Nil(t, err)

}

func TestQueueProtocol(t *testing.T) {
	testSendTx(t, api)
	testGetTxList(t, api)
	testGetBlocks(t, api)
	testGetTransactionByAddr(t, api)
	testQueryTx(t, api)
	testGetTransactionByHash(t, api)
	testGetMempool(t, api)
	testPeerInfo(t, api)
	testGetHeaders(t, api)
	testGetLastMempool(t, api)
	testGetProperFee(t, api)
	testGetBlockOverview(t, api)
	testGetAddrOverview(t, api)
	testGetBlockHash(t, api)
	testGetBlockByHashes(t, api)
	testGetBlockSequences(t, api)
	testAddSeqCallBack(t, api)
	testListSeqCallBack(t, api)
	testGetSeqCallBackLastNum(t, api)
	testGetLastBlockSequence(t, api)
	testIsSync(t, api)
	testIsNtpClockSync(t, api)
	testLocalGet(t, api)
	testLocalTransaction(t, api)
	testLocalList(t, api)
	testGetLastHeader(t, api)
	testStoreSet(t, api)
	testStoreGet(t, api)
	testStoreMemSet(t, api)
	testStoreCommit(t, api)
	testStoreRollback(t, api)
	testStoreDel(t, api)
	testStoreGetTotalCoins(t, api)
	testStoreList(t, api)
	testBlockChainQuery(t, api)
	testQueryConsensus(t, api)
	testExecWalletFunc(t, api)
	testGetSequenceByHash(t, api)
	testDialPeer(t, api)
	testClosePeer(t, api)
}

func testGetSequenceByHash(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.GetSequenceByHash(nil)
	assert.Equal(t, types.ErrInvalidParam, err)
	res, err := api.GetSequenceByHash(&types.ReqHash{})
	assert.Nil(t, err)
	assert.Equal(t, &types.Int64{Data: 1}, res)
}

func testBlockChainQuery(t *testing.T, api client.QueueProtocolAPI) {
	testCases := []struct {
		param     *types.ChainExecutor
		actualRes types.Message
		actualErr error
	}{
		{
			actualErr: types.ErrInvalidParam,
		},
		{
			param:     &types.ChainExecutor{},
			actualRes: &types.Reply{},
		},
	}
	for index, test := range testCases {
		res, err := api.QueryChain(test.param)
		require.Equalf(t, err, test.actualErr, "testBlockChainQuery case index %d", index)
		require.Equalf(t, res, test.actualRes, "testBlockChainQuery case index %d", index)
	}

	_, err := api.Query("", "", nil)
	assert.EqualError(t, types.ErrInvalidParam, err.Error())
	res, err := api.Query("", "", testCases[1].param)
	assert.Nil(t, err)
	assert.Equal(t, res, testCases[1].actualRes)
}

func testQueryConsensus(t *testing.T, api client.QueueProtocolAPI) {
	testCases := []struct {
		param     *types.ChainExecutor
		actualRes types.Message
		actualErr error
	}{
		{
			actualErr: types.ErrInvalidParam,
		},
		{
			param:     &types.ChainExecutor{},
			actualRes: &types.Reply{},
		},
	}
	for index, test := range testCases {
		res, err := api.QueryConsensus(test.param)
		require.Equalf(t, err, test.actualErr, "testQueryConsensus case index %d", index)
		require.Equalf(t, res, test.actualRes, "testQueryConsensus case index %d", index)
	}

	_, err := api.QueryConsensusFunc("", "", nil)
	assert.EqualError(t, types.ErrInvalidParam, err.Error())
	res, err := api.QueryConsensusFunc("", "", testCases[1].param)
	assert.Nil(t, err)
	assert.Equal(t, res, testCases[1].actualRes)
}

func testExecWalletFunc(t *testing.T, api client.QueueProtocolAPI) {
	testCases := []struct {
		param     *types.ChainExecutor
		actualRes types.Message
		actualErr error
	}{
		{
			actualErr: types.ErrInvalidParam,
		},
		{
			param:     &types.ChainExecutor{},
			actualRes: &types.Reply{},
		},
	}
	for index, test := range testCases {
		res, err := api.ExecWallet(test.param)
		require.Equalf(t, err, test.actualErr, "testQueryConsensus case index %d", index)
		require.Equalf(t, res, test.actualRes, "testQueryConsensus case index %d", index)
	}

	_, err := api.ExecWalletFunc("", "", nil)
	assert.EqualError(t, types.ErrInvalidParam, err.Error())
	res, err := api.ExecWalletFunc("", "", testCases[1].param)
	assert.Nil(t, err)
	assert.Equal(t, res, testCases[1].actualRes)
}

func testGetBlockByHashes(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.GetBlockByHashes(nil)
	assert.EqualError(t, types.ErrInvalidParam, err.Error())
	res, err := api.GetBlockByHashes(&types.ReqHashes{Hashes: [][]byte{}})
	assert.Nil(t, err)
	assert.Equal(t, &types.BlockDetails{}, res)
}

func testGetBlockSequences(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.GetBlockSequences(nil)
	assert.EqualError(t, types.ErrInvalidParam, err.Error())
	res, err := api.GetBlockSequences(&types.ReqBlocks{Start: 0, End: 1})
	assert.Nil(t, err)
	assert.Equal(t, &types.BlockSequences{}, res)
}

func testAddSeqCallBack(t *testing.T, api client.QueueProtocolAPI) {
	res, err := api.AddPushSubscribe(&types.PushSubscribeReq{})
	assert.Nil(t, err)
	assert.Equal(t, &types.ReplySubscribePush{}, res)
}

func testListSeqCallBack(t *testing.T, api client.QueueProtocolAPI) {
	res, err := api.ListPushes()
	assert.Nil(t, err)
	assert.Equal(t, &types.PushSubscribes{}, res)
}

func testGetSeqCallBackLastNum(t *testing.T, api client.QueueProtocolAPI) {
	res, err := api.GetPushSeqLastNum(&types.ReqString{})
	assert.Nil(t, err)
	assert.Equal(t, &types.Int64{}, res)
}

func testStoreSet(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.StoreSet(&types.StoreSetWithSync{})
	if err != nil {
		t.Error("Call StoreSet Failed.", err)
	}

	_, err = api.StoreSet(nil)
	if err == nil {
		t.Error("StoreSet(nil) need return error.")
	}
}

func testStoreGet(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.StoreGet(&types.StoreGet{})
	if err != nil {
		t.Error("Call StoreGet Failed.", err)
	}

	_, err = api.StoreGet(nil)
	if err == nil {
		t.Error("StoreGet(nil) need return error.")
	}
}

func testStoreMemSet(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.StoreMemSet(&types.StoreSetWithSync{})
	if err != nil {
		t.Error("Call StoreMemSet Failed.", err)
	}

	_, err = api.StoreMemSet(nil)
	if err == nil {
		t.Error("StoreMemSet(nil) need return error.")
	}
}

func testStoreCommit(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.StoreCommit(&types.ReqHash{})
	if err != nil {
		t.Error("Call StoreCommit Failed.", err)
	}

	_, err = api.StoreCommit(nil)
	if err == nil {
		t.Error("StoreCommit(nil) need return error.")
	}
}

func testStoreRollback(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.StoreRollback(&types.ReqHash{})
	if err != nil {
		t.Error("Call StoreRollback Failed.", err)
	}

	_, err = api.StoreRollback(nil)
	if err == nil {
		t.Error("StoreRollback(nil) need return error.")
	}
}

func testStoreDel(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.StoreDel(&types.StoreDel{})
	if err != nil {
		t.Error("Call StoreDel Failed.", err)
	}

	_, err = api.StoreDel(nil)
	if err == nil {
		t.Error("StoreDel(nil) need return error.")
	}
}

func testStoreGetTotalCoins(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.StoreGetTotalCoins(&types.IterateRangeByStateHash{})
	if err != nil {
		t.Error("Call StoreGetTotalCoins Failed.", err)
	}
	_, err = api.StoreGetTotalCoins(nil)
	if err == nil {
		t.Error("StoreGetTotalCoins(nil) need return error.")
	}
	_, err = api.StoreGetTotalCoins(&types.IterateRangeByStateHash{Count: 10})
	if err == nil {
		t.Error("StoreGetTotalCoins(&types.IterateRangeByStateHash{Count:10}) need return error.")
	}
}

func testStoreList(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.StoreList(&types.StoreList{})
	if err == nil {
		t.Error("Call StoreList Failed.", err)
	}

	_, err = api.StoreList(nil)
	if err == nil {
		t.Error("StoreList(nil) need return error.")
	}
}

func testGetLastHeader(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.GetLastHeader()
	if err != nil {
		t.Error("Call GetLastHeader Failed.", err)
	}
}

func testLocalGet(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.LocalGet(nil)
	if nil == err {
		t.Error("LocalGet(nil) need return error.")
	}
	_, err = api.LocalGet(&types.LocalDBGet{})
	if err != nil {
		t.Error("Call LocalGet Failed.", err)
	}
}

func testLocalList(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.LocalList(nil)
	if nil == err {
		t.Error("LocalList(nil) need return error.")
	}
	_, err = api.LocalList(&types.LocalDBList{})
	if nil != err {
		t.Error("Call LocalList Failed.", err)
	}
}

func testLocalTransaction(t *testing.T, api client.QueueProtocolAPI) {
	txid, err := api.LocalNew(false)
	assert.Nil(t, err)
	assert.Equal(t, txid.Data, int64(9999))
	err = api.LocalBegin(txid)
	assert.Nil(t, err)
	err = api.LocalCommit(nil)
	assert.Equal(t, types.ErrInvalidParam, err)
	err = api.LocalCommit(txid)
	assert.Nil(t, err)
	err = api.LocalRollback(nil)
	assert.Equal(t, types.ErrInvalidParam, err)
	err = api.LocalRollback(txid)
	assert.Nil(t, err)
	err = api.LocalSet(nil)
	assert.Equal(t, types.ErrInvalidParam, err)
	param := &types.LocalDBSet{Txid: txid.Data}
	err = api.LocalSet(param)
	assert.Nil(t, err)
	err = api.LocalClose(txid)
	assert.Nil(t, err)
}

func testIsNtpClockSync(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.IsNtpClockSync()
	if err != nil {
		t.Error("Call IsNtpClockSync Failed.", err)
	}
}

func testIsSync(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.IsSync()
	if err != nil {
		t.Error("Call IsSync Failed.", err)
	}
}

func testGetBlockHash(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.GetBlockHash(&types.ReqInt{})
	if err != nil {
		t.Error("Call GetBlockHash Failed.", err)
	}
	_, err = api.GetBlockHash(nil)
	if err == nil {
		t.Error("GetBlockHash(nil) need return error.")
	}
	_, err = api.GetBlockHash(&types.ReqInt{Height: 10})
	if err == nil {
		t.Error("GetBlockHash(&types.ReqInt{Height:10}) need return error.")
	}
}

func testGetAddrOverview(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.GetAddrOverview(&types.ReqAddr{})
	if err != nil {
		t.Error("Call GetAddrOverview Failed.", err)
	}
	_, err = api.GetAddrOverview(nil)
	if err == nil {
		t.Error("GetAddrOverview(nil) need return error.")
	}
	_, err = api.GetAddrOverview(&types.ReqAddr{Addr: "case1"})
	if err == nil {
		t.Error("GetAddrOverview(&types.ReqAddr{Addr:\"case1\"}) need return error.")
	}
}

func testGetBlockOverview(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.GetBlockOverview(&types.ReqHash{})
	if err != nil {
		t.Error("Call GetBlockOverview Failed.", err)
	}
	_, err = api.GetBlockOverview(nil)
	if err == nil {
		t.Error("GetBlockOverview(nil) need return error.")
	}
	_, err = api.GetBlockOverview(&types.ReqHash{Hash: []byte("case1")})
	if err == nil {
		t.Error("GetBlockOverview(&types.ReqHash{Hash:[]byte(\"case1\")}) need return error.")
	}
}

func testGetLastMempool(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.GetLastMempool()
	if err != nil {
		t.Error("Call GetLastMempool Failed.", err)
	}
}

func testGetProperFee(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.GetProperFee(nil)
	if err != nil {
		t.Error("Call GetProperFee Failed.", err)
	}
	_, err = api.GetProperFee(&types.ReqProperFee{})
	if err != nil {
		t.Error("Call GetProperFee Failed.", err)
	}
}

func testGetHeaders(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.GetHeaders(&types.ReqBlocks{})
	if err != nil {
		t.Error("Call GetHeaders Failed.", err)
	}
	_, err = api.GetHeaders(nil)
	if err == nil {
		t.Error("GetHeaders(nil) need return error.")
	}
	_, err = api.GetHeaders(&types.ReqBlocks{Start: 10})
	if err == nil {
		t.Error("GetHeaders(&types.ReqBlocks{Start:10}) need return error.")
	}
}

func testPeerInfo(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.PeerInfo(&types.P2PGetPeerReq{})
	if err != nil {
		t.Error("Call PeerInfo Failed.", err)
	}
}

func testGetMempool(t *testing.T, api client.QueueProtocolAPI) {
	req := types.ReqGetMempool{IsAll: false}
	_, err := api.GetMempool(&req)
	if err != nil {
		t.Error("Call GetMempool Failed.", err)
	}
}

func testGetTransactionByHash(t *testing.T, api client.QueueProtocolAPI) {
	hashs := types.ReqHashes{}
	hashs.Hashes = make([][]byte, 1)

	_, err := api.GetTransactionByHash(&hashs)
	if err != nil {
		t.Error("Call GetTransactionByHash Failed.", err)
	}
	_, err = api.GetTransactionByHash(nil)
	if err == nil {
		t.Error("GetTransactionByHash(nil) need return error.")
	}

	hashs.Hashes[0] = []byte("case1")
	_, err = api.GetTransactionByHash(&hashs)
	if err == nil {
		t.Error("GetTransactionByHash(&hashs) need return error.")
	}
}

func testQueryTx(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.QueryTx(&types.ReqHash{})
	if err != nil {
		t.Error("Call QueryTx Failed.", err)
	}
	_, err = api.QueryTx(nil)
	if err == nil {
		t.Error("QueryTx(nil) need return error.")
	}
	_, err = api.QueryTx(&types.ReqHash{Hash: []byte("case1")})
	if err == nil {
		t.Error("QueryTx(&ReqHash{Hash:[]byte(\"case1\")}) need return error.")
	}
}

func testGetTransactionByAddr(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.GetTransactionByAddr(&types.ReqAddr{})
	if err != nil {
		t.Error("Call GetTransactionByAddr Failed.", err)
	}
	_, err = api.GetTransactionByAddr(nil)
	if err == nil {
		t.Error("GetTransactionByAddr(nil) need return error.")
	}
	_, err = api.GetTransactionByAddr(&types.ReqAddr{Flag: 1})
	if err == nil {
		t.Error("GetTransactionByAddr(&types.ReqAddr{Flag:1}) need return error.")
	}
}

func testGetBlocks(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.GetBlocks(&types.ReqBlocks{})
	if err != nil {
		t.Error("Call GetBlocks Failed.", err)
	}
	_, err = api.GetBlocks(nil)
	if err == nil {
		t.Error("GetBlocks(nil) need return error.")
	}
	_, err = api.GetBlocks(&types.ReqBlocks{Start: 1})
	if err == nil {
		t.Error("GetBlocks(&types.ReqBlocks{Start:1}) need return error.")
	}
}

func testGetLastBlockSequence(t *testing.T, api client.QueueProtocolAPI) {
	res, err := api.GetLastBlockSequence()
	assert.Nil(t, err)
	assert.Equal(t, &types.Int64{}, res)
}

func testGetTxList(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.GetTxList(&types.TxHashList{})
	if err != nil {
		t.Error("Call GetTxList Failed.", err)
	}
	_, err = api.GetTxList(nil)
	if err == nil {
		t.Error("GetTxList(nil) need return error.")
	}
	_, err = api.GetTxList(&types.TxHashList{Count: 1})
	if err == nil {
		t.Error("SendTx(&types.TxHashList{Count:1}) need return error.")
	}
}

func testSendTx(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.SendTx(&types.Transaction{})
	if err != nil {
		t.Error("Call SendTx Failed.", err)
	}
	_, err = api.SendTx(nil)
	if err == nil {
		t.Error("SendTx(nil) need return error.")
	}
	_, err = api.SendTx(&types.Transaction{Execer: []byte("case1")})
	if err == nil {
		t.Error("SendTx(&types.Transaction{Execer:[]byte(\"case1\")}) need return error.")
	}
	_, err = api.SendTx(&types.Transaction{Execer: []byte("case2")})
	if err == nil {
		t.Error("SendTx(&types.Transaction{Execer:[]byte(\"case2\")}) need return error.")
	}
}

func TestJsonRPC(t *testing.T) {
	testGetBlocksJSONRPC(t, &jrpc)
	testGetBlockOverviewJSONRPC(t, &jrpc)
	testGetBlockHashJSONRPC(t, &jrpc)
	testGetHeadersCmdJSONRPC(t, &jrpc)
	testGetLastHeaderJSONRPC(t, &jrpc)
	testGetMempoolJSONRPC(t, &jrpc)
	testGetLastMemPoolJSONRPC(t, &jrpc)
	testGetProperFeeJSONRPC(t, &jrpc)
	testGenSeedJSONRPC(t, &jrpc)
	testGetPeerInfoJSONRPC(t, &jrpc)
	testIsNtpClockSyncJSONRPC(t, &jrpc)
	testIsSyncJSONRPC(t, &jrpc)
	testGetNetInfoJSONRPC(t, &jrpc)
	testGetWalletStatusJSONRPC(t, &jrpc)
	testDumpPrivkeyJSONRPC(t, &jrpc)
	testDumpPrivkeysFileJSONRPC(t, &jrpc)
	testImportPrivkeysFileJSONRPC(t, &jrpc)
	testGetAccountsJSONRPC(t, &jrpc)
}

func testGetAccountsJSONRPC(t *testing.T, rpc *mockJRPCSystem) {
	var res rpctypes.WalletAccounts
	err := rpc.newRPCCtx("Chain33.GetAccounts", &types.ReqNil{}, &res)
	if err != nil {
		t.Error("testGetAccountsJSONRPC Failed.", err)
	}
}

func testDumpPrivkeyJSONRPC(t *testing.T, rpc *mockJRPCSystem) {
	var res types.ReplyString
	err := rpc.newRPCCtx("Chain33.DumpPrivkey", &types.ReqString{}, &res)
	if err != nil {
		t.Error("testDumpPrivkeyJSONRPC Failed.", err)
	}
}

func testDumpPrivkeysFileJSONRPC(t *testing.T, rpc *mockJRPCSystem) {
	var res rpctypes.Reply
	err := rpc.newRPCCtx("Chain33.DumpPrivkeysFile", &types.ReqPrivkeysFile{}, &res)
	if err != nil {
		t.Error("testDumpPrivkeysFileJSONRPC Failed.", err)
	}
}

func testImportPrivkeysFileJSONRPC(t *testing.T, rpc *mockJRPCSystem) {
	var res rpctypes.Reply
	err := rpc.newRPCCtx("Chain33.ImportPrivkeysFile", &types.ReqPrivkeysFile{}, &res)
	if err != nil {
		t.Error("testImportPrivkeysFileJSONRPC Failed.", err)
	}
}

func testGetWalletStatusJSONRPC(t *testing.T, rpc *mockJRPCSystem) {
	var res rpctypes.WalletStatus
	err := rpc.newRPCCtx("Chain33.GetWalletStatus", &types.ReqNil{}, &res)
	if err != nil {
		t.Error("testGetWalletStatusJSONRPC Failed.", err)
	} else {
		if res.IsTicketLock || res.IsAutoMining || !res.IsHasSeed || !res.IsWalletLock {
			t.Error("testGetWalletStatusJSONRPC return type error.")
		}
	}
}

func testGetNetInfoJSONRPC(t *testing.T, rpc *mockJRPCSystem) {
	var res rpctypes.NodeNetinfo
	err := rpc.newRPCCtx("Chain33.GetNetInfo",
		types.P2PGetNetInfoReq{}, &res)
	if err != nil {
		t.Error("testGetNetInfoJSONRPC failed. Error", err)
	}
}

func testIsSyncJSONRPC(t *testing.T, rpc *mockJRPCSystem) {
	var res bool
	err := rpc.newRPCCtx("Chain33.IsSync",
		nil, &res)
	if err != nil {
		t.Error("testIsSyncJSONRPC failed. Error", err)
	}
}

func testIsNtpClockSyncJSONRPC(t *testing.T, rpc *mockJRPCSystem) {
	var res bool
	err := rpc.newRPCCtx("Chain33.IsNtpClockSync",
		nil, &res)
	if err != nil {
		t.Error("testIsNtpClockSyncJSONRPC failed. Error", err)
	}
}

func testGetPeerInfoJSONRPC(t *testing.T, rpc *mockJRPCSystem) {
	var res types.PeerList
	err := rpc.newRPCCtx("Chain33.GetPeerInfo",
		types.P2PGetPeerReq{}, &res)
	if err != nil {
		t.Error("testGetPeerInfoJSONRPC failed. Error", err)
	}
}

func testGenSeedJSONRPC(t *testing.T, rpc *mockJRPCSystem) {
	params := &types.GenSeedLang{
		Lang: 1,
	}
	var res types.ReplySeed
	err := rpc.newRPCCtx("Chain33.GenSeed",
		params, &res)
	if err != nil {
		t.Error("testGenSeedJSONRPC failed. Error", err)
	}
}

func testGetLastMemPoolJSONRPC(t *testing.T, rpc *mockJRPCSystem) {
	var res rpctypes.ReplyTxList
	err := rpc.newRPCCtx("Chain33.GetLastMemPool",
		nil, &res)
	if err != nil {
		t.Error("testGetLastMemPoolJSONRPC failed. Error", err)
	}
}

func testGetProperFeeJSONRPC(t *testing.T, rpc *mockJRPCSystem) {
	var res rpctypes.ReplyProperFee
	err := rpc.newRPCCtx("Chain33.GetProperFee",
		nil, &res)
	if err != nil {
		t.Error("testGetProperFeeJSONRPC failed. Error", err)
	}
}

func testGetMempoolJSONRPC(t *testing.T, rpc *mockJRPCSystem) {
	var res rpctypes.ReplyTxList
	err := rpc.newRPCCtx("Chain33.GetMempool",
		nil, &res)
	if err != nil {
		t.Error("testGetMempoolJSONRPC failed. Error", err)
	}
}

func testGetLastHeaderJSONRPC(t *testing.T, rpc *mockJRPCSystem) {
	var res rpctypes.Header
	err := rpc.newRPCCtx("Chain33.GetLastHeader",
		nil, &res)
	if err != nil {
		t.Error("testGetLastHeaderJSONRPC failed. Error", err)
	}
}

func testGetHeadersCmdJSONRPC(t *testing.T, rpc *mockJRPCSystem) {
	params := types.ReqBlocks{
		Start:    1,
		End:      1,
		IsDetail: true,
	}

	var res rpctypes.Headers
	err := rpc.newRPCCtx("Chain33.GetHeaders",
		&params, &res)
	if err != nil {
		t.Error("testGetHeadersCmdJSONRPC failed. Error", err)
	}
}

func testGetBlockOverviewJSONRPC(t *testing.T, rpc *mockJRPCSystem) {
	params := rpctypes.QueryParm{
		Hash: "0x67c58d6ba9175313f0468ae4e0ddec946549af7748037c2fdd5d54298afd20b6",
	}

	var res rpctypes.BlockOverview
	err := rpc.newRPCCtx("Chain33.GetBlockOverview",
		params, &res)
	if err != nil {
		t.Error("testGetBlockOverviewJSONRPC failed. Error", err)
	}
}

func testGetBlocksJSONRPC(t *testing.T, rpc *mockJRPCSystem) {
	params := rpctypes.BlockParam{
		Start:    100,
		End:      1000,
		Isdetail: true,
	}

	var res rpctypes.BlockDetails
	err := rpc.newRPCCtx("Chain33.GetBlocks",
		params, &res)
	if err != nil {
		t.Error("testGetBlocksJSONRPC failed. Error", err)
	}
}

func testGetBlockHashJSONRPC(t *testing.T, rpc *mockJRPCSystem) {
	params := types.ReqInt{
		Height: 100,
	}
	var res rpctypes.ReplyHash
	err := rpc.newRPCCtx("Chain33.GetBlockHash",
		&params, &res)
	if err != nil {
		t.Error("testGetBlockHashJSONRPC failed. Error", err)
	}
}

func TestGRPC(t *testing.T) {
	testSendTxGRPC(t, &grpcMock)
	testGetBlocksGRPC(t, &grpcMock)
	testGetLastHeaderGRPC(t, &grpcMock)
	testCreateRawTransactionGRPC(t, &grpcMock)
	testQueryTransactionGRPC(t, &grpcMock)
	testSendTransactionGRPC(t, &grpcMock)
	testGetTransactionByAddrGRPC(t, &grpcMock)
	testGetTransactionByHashesGRPC(t, &grpcMock)
	testGetMemPoolGRPC(t, &grpcMock)
	testGetAccountsGRPC(t, &grpcMock)
	testNewAccountGRPC(t, &grpcMock)
	testWalletTransactionListGRPC(t, &grpcMock)
	testImportPrivKeyGRPC(t, &grpcMock)
	testSendToAddressGRPC(t, &grpcMock)
	testSetTxFeeGRPC(t, &grpcMock)
	testSetLablGRPC(t, &grpcMock)
	testMergeBalanceGRPC(t, &grpcMock)
	testSetPasswdGRPC(t, &grpcMock)
	testLockGRPC(t, &grpcMock)
	testUnLockGRPC(t, &grpcMock)
	testGetPeerInfoGRPC(t, &grpcMock)
	testGetLastMemPoolGRPC(t, &grpcMock)
	testGetProperFeeGRPC(t, &grpcMock)
	testGetWalletStatusGRPC(t, &grpcMock)
	testGetBlockOverviewGRPC(t, &grpcMock)
	testGetAddrOverviewGRPC(t, &grpcMock)
	testGetBlockHashGRPC(t, &grpcMock)
	testGetSequenceByHashGRPC(t, &grpcMock)
	testGetBlockBySeqGRPC(t, &grpcMock)
	testGenSeedGRPC(t, &grpcMock)
	testGetSeedGRPC(t, &grpcMock)
	testSaveSeedGRPC(t, &grpcMock)
	testGetBalanceGRPC(t, &grpcMock)
	testQueryChainGRPC(t, &grpcMock)
	testGetHexTxByHashGRPC(t, &grpcMock)
	testDumpPrivkeyGRPC(t, &grpcMock)
	testDumpPrivkeysFileGRPC(t, &grpcMock)
	testImportPrivkeysFileGRPC(t, &grpcMock)
	testVersionGRPC(t, &grpcMock)
	testIsSyncGRPC(t, &grpcMock)
	testIsNtpClockSyncGRPC(t, &grpcMock)
	testNetInfoGRPC(t, &grpcMock)
	testGetParaTxByTitleGRPC(t, &grpcMock)
	testLoadParaTxByTitleGRPC(t, &grpcMock)
	testGetParaTxByHeightGRPC(t, &grpcMock)
}

func testNetInfoGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.NodeNetInfo
	err := rpc.newRPCCtx("NetInfo", &types.P2PGetNetInfoReq{}, &res)
	if err != nil {
		t.Error("Call NetInfo Failed.", err)
	}
}

func testIsNtpClockSyncGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.Reply
	err := rpc.newRPCCtx("IsNtpClockSync", &types.ReqNil{}, &res)
	if err != nil {
		t.Error("Call IsNtpClockSync Failed.", err)
	}
}

func testIsSyncGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.Reply
	err := rpc.newRPCCtx("IsSync", &types.ReqNil{}, &res)
	if err != nil {
		t.Error("Call IsSync Failed.", err)
	}
}

func testVersionGRPC(t *testing.T, rpc *mockGRPCSystem) {
	res := &types.VersionInfo{}
	err := rpc.newRPCCtx("Version", &types.ReqNil{}, res)
	if err != nil {
		t.Error("Call Version Failed.", err)
	}
	assert.Equal(t, version.GetVersion(), rpc.ctx.Res.(*types.VersionInfo).Chain33)
}

func testDumpPrivkeyGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.ReplyString
	err := rpc.newRPCCtx("DumpPrivkey", &types.ReqString{}, &res)
	if err != nil {
		t.Error("Call DumpPrivkey Failed.", err)
	}
}

func testDumpPrivkeysFileGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.Reply
	err := rpc.newRPCCtx("DumpPrivkeysFile", &types.ReqPrivkeysFile{}, &res)
	if err != nil {
		t.Error("Call DumpPrivkeysFile Failed.", err)
	}
}

func testImportPrivkeysFileGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.Reply
	err := rpc.newRPCCtx("ImportPrivkeysFile", &types.ReqPrivkeysFile{}, &res)
	if err != nil {
		t.Error("Call ImportPrivkeysFile Failed.", err)
	}
}

func testGetHexTxByHashGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.HexTx
	err := rpc.newRPCCtx("GetHexTxByHash", &types.ReqHash{Hash: []byte("fdafdsafds")}, &res)
	if err != nil {
		t.Error("Call GetHexTxByHash Failed.", err)
	}
}

func testQueryChainGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.Reply
	err := rpc.newRPCCtx("QueryChain", &types.ChainExecutor{}, &res)
	if err != nil {
		t.Error("Call QueryChain Failed.", err)
	}
}

func testGetBalanceGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.Accounts
	err := rpc.newRPCCtx("GetBalance", &types.ReqBalance{}, &res)
	if err != nil {
		t.Error("Call GetBalance Failed.", err)
	}
}

func testSaveSeedGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.Reply
	err := rpc.newRPCCtx("SaveSeed", &types.SaveSeedByPw{}, &res)
	if err != nil {
		t.Error("Call SaveSeed Failed.", err)
	}
}

func testGetSeedGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.ReplySeed
	err := rpc.newRPCCtx("GetSeed", &types.GetSeedByPw{}, &res)
	if err != nil {
		t.Error("Call GetSeed Failed.", err)
	}
}

func testGenSeedGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.ReplySeed
	err := rpc.newRPCCtx("GenSeed", &types.GenSeedLang{}, &res)
	if err != nil {
		t.Error("Call GenSeed Failed.", err)
	}
}

func testGetBlockHashGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.ReplyHash
	err := rpc.newRPCCtx("GetBlockHash", &types.ReqInt{}, &res)
	if err != nil {
		t.Error("Call GetBlockHash Failed.", err)
	}
}

func testGetAddrOverviewGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.AddrOverview
	err := rpc.newRPCCtx("GetAddrOverview", &types.ReqAddr{Addr: "13cS5G1BDN2YfGudsxRxr7X25yu6ZdgxMU"}, &res)
	if err != nil {
		t.Error("Call GetAddrOverview Failed.", err)
	}
}

func testGetBlockOverviewGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.BlockOverview
	err := rpc.newRPCCtx("GetBlockOverview", &types.ReqHash{}, &res)
	if err != nil {
		t.Error("Call GetBlockOverview Failed.", err)
	}
}

func testGetWalletStatusGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.WalletStatus
	err := rpc.newRPCCtx("GetWalletStatus", &types.ReqNil{}, &res)
	if err != nil {
		t.Error("Call GetWalletStatus Failed.", err)
	}
}

func testGetLastMemPoolGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.ReplyTxList
	err := rpc.newRPCCtx("GetLastMemPool", &types.ReqNil{}, &res)
	if err != nil {
		t.Error("Call GetLastMemPool Failed.", err)
	}
}

func testGetProperFeeGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.ReplyProperFee
	err := rpc.newRPCCtx("GetProperFee", &types.ReqProperFee{}, &res)
	if err != nil {
		t.Error("Call GetProperFee Failed.", err)
	}
}

func testGetPeerInfoGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.PeerList
	err := rpc.newRPCCtx("GetPeerInfo", &types.P2PGetPeerReq{}, &res)
	if err != nil {
		t.Error("Call GetPeerInfo Failed.", err)
	}
}

func testUnLockGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.Reply
	err := rpc.newRPCCtx("UnLock", &types.WalletUnLock{}, &res)
	if err != nil {
		t.Error("Call UnLock Failed.", err)
	}
}

func testLockGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.Reply
	err := rpc.newRPCCtx("Lock", &types.ReqNil{}, &res)
	if err != nil {
		t.Error("Call Lock Failed.", err)
	}
}

func testSetPasswdGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.Reply
	err := rpc.newRPCCtx("SetPasswd", &types.ReqWalletSetPasswd{}, &res)
	if err != nil {
		t.Error("Call SetPasswd Failed.", err)
	}
}

func testMergeBalanceGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.ReplyHashes
	err := rpc.newRPCCtx("MergeBalance", &types.ReqWalletMergeBalance{}, &res)
	if err != nil {
		t.Error("Call MergeBalance Failed.", err)
	}
}

func testSetLablGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.WalletAccount
	err := rpc.newRPCCtx("SetLabl", &types.ReqWalletSetLabel{}, &res)
	if err != nil {
		t.Error("Call SetLabl Failed.", err)
	}
}

func testSetTxFeeGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.Reply
	err := rpc.newRPCCtx("SetTxFee", &types.ReqWalletSetFee{}, &res)
	if err != nil {
		t.Error("Call SetTxFee Failed.", err)
	}
}

func testSendToAddressGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.ReplyHash
	err := rpc.newRPCCtx("SendToAddress", &types.ReqWalletSendToAddress{}, &res)
	if err != nil {
		t.Error("Call SendToAddress Failed.", err)
	}
}

func testImportPrivKeyGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.WalletAccount
	err := rpc.newRPCCtx("ImportPrivkey", &types.ReqWalletImportPrivkey{}, &res)
	if err != nil {
		t.Error("Call ImportPrivKey Failed.", err)
	}
}

func testWalletTransactionListGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.WalletTxDetails
	err := rpc.newRPCCtx("WalletTransactionList", &types.ReqWalletTransactionList{}, &res)
	if err != nil {
		t.Error("Call WalletTransactionList Failed.", err)
	}
}

func testNewAccountGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.WalletAccount
	err := rpc.newRPCCtx("NewAccount", &types.ReqNewAccount{}, &res)
	if err != nil {
		t.Error("Call NewAccount Failed.", err)
	}
}

func testGetAccountsGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.WalletAccounts
	err := rpc.newRPCCtx("GetAccounts", &types.ReqNil{}, &res)
	if err != nil {
		t.Error("Call GetAccounts Failed.", err)
	}
}

func testGetMemPoolGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.ReplyTxList
	err := rpc.newRPCCtx("GetMemPool", &types.ReqGetMempool{}, &res)
	if err != nil {
		t.Error("Call GetMemPool Failed.", err)
	}
}

func testGetTransactionByHashesGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.TransactionDetails
	err := rpc.newRPCCtx("GetTransactionByHashes", &types.ReqHashes{}, &res)
	if err != nil {
		t.Error("Call GetTransactionByHashes Failed.", err)
	}
}

func testGetTransactionByAddrGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.ReplyTxInfos
	err := rpc.newRPCCtx("GetTransactionByAddr", &types.ReqAddr{}, &res)
	if err != nil {
		t.Error("Call GetTransactionByAddr Failed.", err)
	}
}

func testSendTransactionGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.Reply
	err := rpc.newRPCCtx("SendTransaction", &types.Transaction{}, &res)
	if err != nil {
		t.Error("Call SendTransaction Failed.", err)
	}
}

func testQueryTransactionGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.TransactionDetail
	err := rpc.newRPCCtx("QueryTransaction", &types.ReqHash{}, &res)
	if err != nil {
		t.Error("Call QueryTransaction Failed.", err)
	}
}

func testCreateRawTransactionGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.UnsignTx
	err := rpc.newRPCCtx("CreateRawTransaction",
		&types.CreateTx{To: "1EDDghAtgBsamrNEtNmYdQzC1QEhLkr87t",
			Amount:     10000000,
			Fee:        1000000,
			IsWithdraw: false,
			IsToken:    false,
			ExecName:   "coins",
		},
		&res)
	if err != nil {
		t.Error("Call CreateRawTransaction Failed.", err)
	}
}

func testGetLastHeaderGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.Header
	err := rpc.newRPCCtx("GetLastHeader", &types.ReqNil{}, &res)
	if err != nil {
		t.Error("Call GetLastHeader Failed.", err)
	}
}

func testGetBlocksGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.Reply
	err := rpc.newRPCCtx("GetBlocks", &types.ReqBlocks{}, &res)
	if err != nil {
		t.Error("Call GetBlocks Failed.", err)
	}
}

func testSendTxGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.Reply
	err := rpc.newRPCCtx("SendTransaction", &types.Transaction{}, &res)
	if err != nil {
		t.Error("Call SendTransaction Failed.", err)
	}
}

func testGetSequenceByHashGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.Int64
	err := rpc.newRPCCtx("GetSequenceByHash", &types.ReqHash{}, &res)
	if err != nil {
		t.Error("Call GetSequenceByHash Failed.", err)
	}
}

func testGetBlockBySeqGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.BlockSeq
	//just for coverage
	err := rpc.newRPCCtx("GetBlockBySeq", &types.Int64{Data: 1}, &res)
	assert.Nil(t, err)

	err = rpc.newRPCCtx("GetBlockBySeq", &types.Int64{Data: 10}, &res)
	assert.NotNil(t, err)

}

func TestGetBlockBySeq(t *testing.T) {
	q := client.QueueProtocol{}
	_, err := q.GetBlockBySeq(nil)
	assert.NotNil(t, err)

}

func TestGetMainSeq(t *testing.T) {
	net := queue.New("test-seq-api")
	defer net.Close()

	chain := &mockBlockChain{}
	chain.SetQueueClient(net)
	defer chain.Close()
	net.SetConfig(types.NewChain33Config(types.GetDefaultCfgstring()))
	api, err := client.New(net.Client(), nil)
	assert.Nil(t, err)

	seq, err := api.GetMainSequenceByHash(&types.ReqHash{Hash: []byte("exist-hash")})
	assert.Nil(t, err)
	assert.Equal(t, int64(9999), seq.Data)

	seq, err = api.GetMainSequenceByHash(&types.ReqHash{Hash: []byte("")})
	assert.NotNil(t, err)

	seq1, err := api.GetLastBlockMainSequence()
	assert.Nil(t, err)
	assert.Equal(t, int64(9999), seq1.Data)
}

func testGetParaTxByTitleGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.ParaTxDetails
	var req types.ReqParaTxByTitle
	req.Start = 0
	req.End = 0
	req.Title = "user"
	err := rpc.newRPCCtx("GetParaTxByTitle", &req, &res)
	assert.NotNil(t, err)

	req.Title = "user.p.para."
	err = rpc.newRPCCtx("GetParaTxByTitle", &req, &res)
	assert.Nil(t, err)

}

func TestGetParaTxByTitle(t *testing.T) {
	q := client.QueueProtocol{}
	_, err := q.GetParaTxByTitle(nil)
	assert.NotNil(t, err)

}

func testLoadParaTxByTitleGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.ReplyHeightByTitle
	var req types.ReqHeightByTitle
	req.Count = 1
	req.Direction = 0
	req.Title = "user"
	req.Height = 0

	err := rpc.newRPCCtx("LoadParaTxByTitle", &req, &res)
	assert.NotNil(t, err)

	req.Title = "user.p.para."
	err = rpc.newRPCCtx("LoadParaTxByTitle", &req, &res)
	assert.Nil(t, err)
}

func TestLoadParaTxByTitle(t *testing.T) {
	q := client.QueueProtocol{}
	_, err := q.LoadParaTxByTitle(nil)
	assert.NotNil(t, err)
}

func testGetParaTxByHeightGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.ParaTxDetails
	var req types.ReqParaTxByHeight
	req.Items = append(req.Items, 0)
	req.Title = "user"
	err := rpc.newRPCCtx("GetParaTxByHeight", &req, &res)
	assert.NotNil(t, err)

	req.Title = "user.p.para."
	err = rpc.newRPCCtx("GetParaTxByHeight", &req, &res)
	assert.Nil(t, err)
}

func TestGetParaTxByHeight(t *testing.T) {
	q := client.QueueProtocol{}
	_, err := q.GetParaTxByHeight(nil)
	assert.NotNil(t, err)
}

func TestQueueProtocol_SendDelayTx(t *testing.T) {
	q := queue.New("delaytx")
	q.SetConfig(types.NewChain33Config(types.GetDefaultCfgstring()))
	defer q.Close()
	api, err := client.New(q.Client(), nil)
	require.Nil(t, err)
	_, err = api.SendDelayTx(&types.DelayTx{}, true)
	require.Equal(t, types.ErrNilTransaction, err)
	replyChan := make(chan interface{}, 1)
	go func() {
		cli := q.Client()
		cli.Sub("mempool")
		for msg := range cli.Recv() {
			if msg.Ty == types.EventAddDelayTx {
				replyMsg := cli.NewMessage("rpc", types.EventReply, nil)
				replyMsg.Data = <-replyChan
				msg.Reply(replyMsg)
			}
		}
	}()

	replyChan <- &types.ReqNil{}
	_, err = api.SendDelayTx(&types.DelayTx{Tx: &types.Transaction{}}, true)
	require.Equal(t, types.ErrTypeAsset, err)
	errMsg := "errMsg"
	replyChan <- &types.Reply{Msg: []byte(errMsg)}
	testDelayTx := &types.DelayTx{Tx: &types.Transaction{Payload: []byte("delaytx")}}
	_, err = api.SendDelayTx(testDelayTx, true)
	require.Equal(t, errMsg, err.Error())
	replyChan <- &types.Reply{IsOk: true}
	reply, err := api.SendDelayTx(testDelayTx, true)
	require.Nil(t, err)
	require.Equal(t, testDelayTx.GetTx().Hash(), reply.GetMsg())
	reply, err = api.SendDelayTx(testDelayTx, false)
	require.Nil(t, err)
	require.Nil(t, reply)
}

func testDialPeer(t *testing.T, api client.QueueProtocolAPI) {

	reply, err := api.DialPeer(&types.SetPeer{})
	if err != nil {
		t.Error("Call PeerInfo Failed.", err)
	}
	assert.Equal(t, types.ErrInvalidParam.Error(), string(reply.GetMsg()))

	reply, err = api.DialPeer(&types.SetPeer{PeerAddr: "/ip4/192.168.1.1/tcp/13803/p2p/16uwr23wefwrwrwwerwerwerwerwewe"})
	assert.Nil(t, err)
	assert.Equal(t, "success", string(reply.GetMsg()))

}

func testClosePeer(t *testing.T, api client.QueueProtocolAPI) {
	reply, err := api.ClosePeer(&types.SetPeer{})
	if err != nil {
		t.Error("Call PeerInfo Failed.", err)
	}
	assert.Equal(t, types.ErrInvalidParam.Error(), string(reply.GetMsg()))
	reply, err = api.DialPeer(&types.SetPeer{PeerAddr: "/ip4/192.168.1.1/tcp/13803/p2p/16uwr23wefwrwrwwerwerwerwerwewe"})
	assert.Nil(t, err)
	assert.Equal(t, "success", string(reply.GetMsg()))

}

func TestQueueProtocol_GetFinalizedBlock(t *testing.T) {

	q := queue.New("test")

	cli := q.Client()
	api, err := client.New(cli, nil)
	require.Nil(t, err)
	defer cli.Close()
	go func() {

		cli.Sub("blockchain")
		for msg := range cli.Recv() {
			msg.Reply(cli.NewMessage("", 0, &types.SnowChoice{Height: 1}))
		}
	}()

	sc, err := api.GetFinalizedBlock()
	require.Nil(t, err)
	require.Equal(t, 1, int(sc.GetHeight()))
}
