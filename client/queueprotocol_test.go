// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client_test

import (
	"os"
	"testing"

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
	pluginmgr.InitExec(nil)
	api = mock.startup(0)
	flag := m.Run()
	mock.stop()
	os.Exit(flag)
}

func TestQueueProtocolAPI(t *testing.T) {
	var option client.QueueProtocolOption
	option.SendTimeout = 100
	option.WaitTimeout = 200

	_, err := client.New(nil, nil)
	if err == nil {
		t.Error("client.New(nil, nil) need return error")
	}

	var q = queue.New("channel")
	qc, err := client.New(q.Client(), &option)
	if err != nil {
		t.Errorf("client.New() cause error %v", err)
	}
	if qc == nil {
		t.Error("queueprotoapi object is nil")
	}

}

func TestQueueProtocol(t *testing.T) {
	testSendTx(t, api)
	testGetTxList(t, api)
	testGetBlocks(t, api)
	testGetTransactionByAddr(t, api)
	testQueryTx(t, api)
	testGetTransactionByHash(t, api)
	testGetMempool(t, api)
	testWalletGetAccountList(t, api)
	testNewAccount(t, api)
	testWalletTransactionList(t, api)
	testWalletImportprivkey(t, api)
	testWalletSendToAddress(t, api)
	testWalletSetFee(t, api)
	testWalletSetLabel(t, api)
	testWalletMergeBalance(t, api)
	testWalletSetPasswd(t, api)
	testWalletLock(t, api)
	testWalletUnLock(t, api)
	testPeerInfo(t, api)
	testGetHeaders(t, api)
	testGetLastMempool(t, api)
	testGetProperFee(t, api)
	testGetBlockOverview(t, api)
	testGetAddrOverview(t, api)
	testGetBlockHash(t, api)
	testGenSeed(t, api)
	testSaveSeed(t, api)
	testGetSeed(t, api)
	testGetWalletStatus(t, api)
	testDumpPrivkey(t, api)
	testIsSync(t, api)
	testIsNtpClockSync(t, api)
	testLocalGet(t, api)
	testLocalTransaction(t, api)
	testLocalList(t, api)
	testGetLastHeader(t, api)
	testSignRawTx(t, api)
	testStoreGetTotalCoins(t, api)
	testStoreList(t, api)
	testBlockChainQuery(t, api)
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

func testSignRawTx(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.SignRawTx(&types.ReqSignRawTx{})
	if err != nil {
		t.Error("Call SignRawTx Failed.", err)
	}
	_, err = api.SignRawTx(nil)
	if err == nil {
		t.Error("SignRawTx(nil) need return error.")
	}
	_, err = api.SignRawTx(&types.ReqSignRawTx{Addr: "case1"})
	if err == nil {
		t.Error("SignRawTx(&types.ReqStr{Addr:\"case1\"}) need return error.")
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
	txid, err := api.LocalNew(nil)
	assert.Nil(t, err)
	assert.Equal(t, txid.Data, int64(9999))
	err = api.LocalBegin(txid)
	assert.Nil(t, err)
	err = api.LocalCommit(txid)
	assert.Nil(t, err)
	err = api.LocalRollback(txid)
	assert.Nil(t, err)
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

func testDumpPrivkey(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.DumpPrivkey(&types.ReqString{})
	if err != nil {
		t.Error("Call DumpPrivkey Failed.", err)
	}
	_, err = api.DumpPrivkey(nil)
	if err == nil {
		t.Error("DumpPrivkey(nil) need return error.")
	}
	_, err = api.DumpPrivkey(&types.ReqString{Data: "case1"})
	if err == nil {
		t.Error("DumpPrivkey(&types.ReqStr{ReqStr:\"case1\"}) need return error.")
	}
}

func testGetWalletStatus(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.GetWalletStatus()
	if err != nil {
		t.Error("Call GetWalletStatus Failed.", err)
	}
}

func testGetSeed(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.GetSeed(&types.GetSeedByPw{})
	if err != nil {
		t.Error("Call GetSeed Failed.", err)
	}
	_, err = api.GetSeed(nil)
	if err == nil {
		t.Error("GetSeed(nil) need return error.")
	}
	_, err = api.GetSeed(&types.GetSeedByPw{Passwd: "case1"})
	if err == nil {
		t.Error("GetSeed(&types.GetSeedByPw{Passwd:\"case1\"}) need return error.")
	}
}

func testSaveSeed(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.SaveSeed(&types.SaveSeedByPw{})
	if err != nil {
		t.Error("Call SaveSeed Failed.", err)
	}
	_, err = api.SaveSeed(nil)
	if err == nil {
		t.Error("SaveSeed(nil) need return error.")
	}
	_, err = api.SaveSeed(&types.SaveSeedByPw{Seed: "case1"})
	if err == nil {
		t.Error("SaveSeed(&types.SaveSeedByPw{Seed:\"case1\"}) need return error.")
	}
}

func testGenSeed(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.GenSeed(&types.GenSeedLang{})
	if err != nil {
		t.Error("Call GenSeed Failed.", err)
	}
	_, err = api.GenSeed(nil)
	if err == nil {
		t.Error("GenSeed(nil) need return error.")
	}
	_, err = api.GenSeed(&types.GenSeedLang{Lang: 10})
	if err == nil {
		t.Error("GenSeed(&types.GenSeedLang{Lang:10}) need return error.")
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
	_, err := api.GetProperFee()
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
	_, err := api.PeerInfo()
	if err != nil {
		t.Error("Call PeerInfo Failed.", err)
	}
}

func testWalletUnLock(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.WalletUnLock(&types.WalletUnLock{})
	if err != nil {
		t.Error("Call WalletUnLock Failed.", err)
	}
	_, err = api.WalletUnLock(nil)
	if err == nil {
		t.Error("WalletUnLock(nil) need return error.")
	}
	_, err = api.WalletUnLock(&types.WalletUnLock{Passwd: "case1"})
	if err == nil {
		t.Error("WalletUnLock(&types.WalletUnLock{Passwd:\"case1\"}) need return error.")
	}
}

func testWalletLock(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.WalletLock()
	if err != nil {
		t.Error("Call WalletLock Failed.", err)
	}
}

func testWalletSetPasswd(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.WalletSetPasswd(&types.ReqWalletSetPasswd{})
	if err != nil {
		t.Error("Call WalletSetPasswd Failed.", err)
	}
	_, err = api.WalletSetPasswd(nil)
	if err == nil {
		t.Error("WalletSetPasswd(nil) need return error.")
	}
	_, err = api.WalletSetPasswd(&types.ReqWalletSetPasswd{OldPass: "case1"})
	if err == nil {
		t.Error("WalletSetPasswd(&types.ReqWalletSetPasswd{OldPass:\"case1\"}) need return error.")
	}
}

func testWalletMergeBalance(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.WalletMergeBalance(&types.ReqWalletMergeBalance{})
	if err != nil {
		t.Error("Call WalletMergeBalance Failed.", err)
	}
	_, err = api.WalletMergeBalance(nil)
	if err == nil {
		t.Error("WalletMergeBalance(nil) need return error.")
	}
	_, err = api.WalletMergeBalance(&types.ReqWalletMergeBalance{To: "case1"})
	if err == nil {
		t.Error("WalletMergeBalance(&types.ReqWalletMergeBalance{To:\"case1\"}) need return error.")
	}
}

func testWalletSetLabel(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.WalletSetLabel(&types.ReqWalletSetLabel{})
	if err != nil {
		t.Error("Call WalletSetLabel Failed.", err)
	}
	_, err = api.WalletSetLabel(nil)
	if err == nil {
		t.Error("WalletSetLabel(nil) need return error.")
	}
	_, err = api.WalletSetLabel(&types.ReqWalletSetLabel{Label: "case1"})
	if err == nil {
		t.Error("WalletSetLabel(&types.ReqWalletSetLabel{Label:\"case1\"}) need return error.")
	}
}

func testWalletSetFee(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.WalletSetFee(&types.ReqWalletSetFee{})
	if err != nil {
		t.Error("Call WalletSetFee Failed.", err)
	}
	_, err = api.WalletSetFee(nil)
	if err == nil {
		t.Error("WalletSetFee(nil) need return error.")
	}
	_, err = api.WalletSetFee(&types.ReqWalletSetFee{Amount: 1000})
	if err == nil {
		t.Error("WalletSetFee(&types.ReqWalletSetFee{Amount:1000}) need return error.")
	}
}

func testWalletSendToAddress(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.WalletSendToAddress(&types.ReqWalletSendToAddress{})
	if err != nil {
		t.Error("Call WalletSendToAddress Failed.", err)
	}
	_, err = api.WalletSendToAddress(nil)
	if err == nil {
		t.Error("WalletSendToAddress(nil) need return error.")
	}
	_, err = api.WalletSendToAddress(&types.ReqWalletSendToAddress{Note: "case1"})
	if err == nil {
		t.Error("WalletSendToAddress(&types.ReqWalletSendToAddress{Note:\"case1\"}) need return error.")
	}
}

func testWalletImportprivkey(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.WalletImportprivkey(&types.ReqWalletImportPrivkey{})
	if err != nil {
		t.Error("Call WalletTransactionList Failed.", err)
	}
	_, err = api.WalletImportprivkey(nil)
	if err == nil {
		t.Error("WalletImportprivkey(nil) need return error.")
	}
	_, err = api.WalletImportprivkey(&types.ReqWalletImportPrivkey{Label: "case1"})
	if err == nil {
		t.Error("WalletImportprivkey(&types.ReqWalletImportPrivKey{Label:\"case1\"}) need return error.")
	}
}

func testWalletTransactionList(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.WalletTransactionList(&types.ReqWalletTransactionList{})
	if err != nil {
		t.Error("Call WalletTransactionList Failed.", err)
	}
	_, err = api.WalletTransactionList(nil)
	if err == nil {
		t.Error("WalletTransactionList(nil) need return error.")
	}
	_, err = api.WalletTransactionList(&types.ReqWalletTransactionList{Direction: 1})
	if err == nil {
		t.Error("WalletTransactionList(&types.ReqWalletTransactionList{Direction:1}) need return error.")
	}
}

func testNewAccount(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.NewAccount(&types.ReqNewAccount{})
	if err != nil {
		t.Error("Call NewAccount Failed.", err)
	}
	_, err = api.NewAccount(nil)
	if err == nil {
		t.Error("NewAccount(nil) need return error.")
	}
	_, err = api.NewAccount(&types.ReqNewAccount{Label: "case1"})
	if err == nil {
		t.Error("NewAccount(&types.ReqNewAccount{Label:\"case1\"}) need return error.")
	}
}

func testWalletGetAccountList(t *testing.T, api client.QueueProtocolAPI) {
	req := types.ReqAccountList{WithoutBalance: true}
	_, err := api.WalletGetAccountList(&req)
	if err != nil {
		t.Error("Call WalletGetAccountList Failed.", err)
	}
}

func testGetMempool(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.GetMempool()
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
	testGetBlocksJsonRPC(t, &jrpc)
	testGetBlockOverviewJsonRPC(t, &jrpc)
	testGetBlockHashJsonRPC(t, &jrpc)
	testGetHeadersCmdJsonRPC(t, &jrpc)
	testGetLastHeaderJsonRPC(t, &jrpc)
	testGetMempoolJsonRPC(t, &jrpc)
	testGetLastMemPoolJsonRPC(t, &jrpc)
	testGetProperFeeJsonRPC(t, &jrpc)
	testGenSeedsonRPC(t, &jrpc)
	testGetPeerInfoJsonRPC(t, &jrpc)
	testIsNtpClockSyncJsonRPC(t, &jrpc)
	testIsSyncJsonRPC(t, &jrpc)
	testGetNetInfoJsonRPC(t, &jrpc)
	testGetWalletStatusJsonRPC(t, &jrpc)
	testDumpPrivkeyJsonRPC(t, &jrpc)
	testGetAccountsJsonRPC(t, &jrpc)
}

func testGetAccountsJsonRPC(t *testing.T, rpc *mockJRPCSystem) {
	var res rpctypes.WalletAccounts
	err := rpc.newRpcCtx("Chain33.GetAccounts", &types.ReqNil{}, &res)
	if err != nil {
		t.Error("testGetAccountsJsonRPC Failed.", err)
	}
}

func testDumpPrivkeyJsonRPC(t *testing.T, rpc *mockJRPCSystem) {
	var res types.ReplyString
	err := rpc.newRpcCtx("Chain33.DumpPrivkey", &types.ReqString{}, &res)
	if err != nil {
		t.Error("testDumpPrivkeyJsonRPC Failed.", err)
	}
}

func testGetWalletStatusJsonRPC(t *testing.T, rpc *mockJRPCSystem) {
	var res rpctypes.WalletStatus
	err := rpc.newRpcCtx("Chain33.GetWalletStatus", &types.ReqNil{}, &res)
	if err != nil {
		t.Error("testGetWalletStatusJsonRPC Failed.", err)
	} else {
		if res.IsTicketLock || res.IsAutoMining || !res.IsHasSeed || !res.IsWalletLock {
			t.Error("testGetWalletStatusJsonRPC return type error.")

		}
	}
}

func testGetNetInfoJsonRPC(t *testing.T, rpc *mockJRPCSystem) {
	var res rpctypes.NodeNetinfo
	err := rpc.newRpcCtx("Chain33.GetNetInfo",
		nil, &res)
	if err != nil {
		t.Error("testGetNetInfoJsonRPC failed. Error", err)
	}
}

func testIsSyncJsonRPC(t *testing.T, rpc *mockJRPCSystem) {
	var res bool
	err := rpc.newRpcCtx("Chain33.IsSync",
		nil, &res)
	if err != nil {
		t.Error("testIsSyncJsonRPC failed. Error", err)
	}
}

func testIsNtpClockSyncJsonRPC(t *testing.T, rpc *mockJRPCSystem) {
	var res bool
	err := rpc.newRpcCtx("Chain33.IsNtpClockSync",
		nil, &res)
	if err != nil {
		t.Error("testIsNtpClockSyncJsonRPC failed. Error", err)
	}
}

func testGetPeerInfoJsonRPC(t *testing.T, rpc *mockJRPCSystem) {
	var res types.PeerList
	err := rpc.newRpcCtx("Chain33.GetPeerInfo",
		nil, &res)
	if err != nil {
		t.Error("testGetPeerInfoJsonRPC failed. Error", err)
	}
}

func testGenSeedsonRPC(t *testing.T, rpc *mockJRPCSystem) {
	params := types.GenSeedLang{
		Lang: 1,
	}
	var res types.ReplySeed
	err := rpc.newRpcCtx("Chain33.GenSeed",
		params, &res)
	if err != nil {
		t.Error("testGenSeedsonRPC failed. Error", err)
	}
}

func testGetLastMemPoolJsonRPC(t *testing.T, rpc *mockJRPCSystem) {
	var res rpctypes.ReplyTxList
	err := rpc.newRpcCtx("Chain33.GetLastMemPool",
		nil, &res)
	if err != nil {
		t.Error("testGetLastMemPoolJsonRPC failed. Error", err)
	}
}

func testGetProperFeeJsonRPC(t *testing.T, rpc *mockJRPCSystem) {
	var res rpctypes.ReplyProperFee
	err := rpc.newRpcCtx("Chain33.GetProperFee",
		nil, &res)
	if err != nil {
		t.Error("testGetProperFeeJsonRPC failed. Error", err)
	}
}

func testGetMempoolJsonRPC(t *testing.T, rpc *mockJRPCSystem) {
	var res rpctypes.ReplyTxList
	err := rpc.newRpcCtx("Chain33.GetMempool",
		nil, &res)
	if err != nil {
		t.Error("testGetMempoolJsonRPC failed. Error", err)
	}
}

func testGetLastHeaderJsonRPC(t *testing.T, rpc *mockJRPCSystem) {
	var res rpctypes.Header
	err := rpc.newRpcCtx("Chain33.GetLastHeader",
		nil, &res)
	if err != nil {
		t.Error("testGetLastHeaderJsonRPC failed. Error", err)
	}
}

func testGetHeadersCmdJsonRPC(t *testing.T, rpc *mockJRPCSystem) {
	params := types.ReqBlocks{
		Start:    1,
		End:      1,
		IsDetail: true,
	}

	var res rpctypes.Headers
	err := rpc.newRpcCtx("Chain33.GetHeaders",
		params, &res)
	if err != nil {
		t.Error("testGetHeadersCmdJsonRPC failed. Error", err)
	}
}

func testGetBlockOverviewJsonRPC(t *testing.T, rpc *mockJRPCSystem) {
	params := rpctypes.QueryParm{
		Hash: "0x67c58d6ba9175313f0468ae4e0ddec946549af7748037c2fdd5d54298afd20b6",
	}

	var res rpctypes.BlockOverview
	err := rpc.newRpcCtx("Chain33.GetBlockOverview",
		params, &res)
	if err != nil {
		t.Error("testGetBlockOverviewJsonRPC failed. Error", err)
	}
}

func testGetBlocksJsonRPC(t *testing.T, rpc *mockJRPCSystem) {
	params := rpctypes.BlockParam{
		Start:    100,
		End:      1000,
		Isdetail: true,
	}

	var res rpctypes.BlockDetails
	err := rpc.newRpcCtx("Chain33.GetBlocks",
		params, &res)
	if err != nil {
		t.Error("testGetBlocksJsonRPC failed. Error", err)
	}
}

func testGetBlockHashJsonRPC(t *testing.T, rpc *mockJRPCSystem) {
	params := types.ReqInt{
		Height: 100,
	}
	var res rpctypes.ReplyHash
	err := rpc.newRpcCtx("Chain33.GetBlockHash",
		params, &res)
	if err != nil {
		t.Error("testGetBlockHashJsonRPC failed. Error", err)
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
	testVersionGRPC(t, &grpcMock)
	testIsSyncGRPC(t, &grpcMock)
	testIsNtpClockSyncGRPC(t, &grpcMock)
	testNetInfoGRPC(t, &grpcMock)

}

func testNetInfoGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.NodeNetInfo
	err := rpc.newRpcCtx("NetInfo", &types.ReqNil{}, &res)
	if err != nil {
		t.Error("Call NetInfo Failed.", err)
	}
}

func testIsNtpClockSyncGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.Reply
	err := rpc.newRpcCtx("IsNtpClockSync", &types.ReqNil{}, &res)
	if err != nil {
		t.Error("Call IsNtpClockSync Failed.", err)
	}
}

func testIsSyncGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.Reply
	err := rpc.newRpcCtx("IsSync", &types.ReqNil{}, &res)
	if err != nil {
		t.Error("Call IsSync Failed.", err)
	}
}

func testVersionGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.VersionInfo
	err := rpc.newRpcCtx("Version", &types.ReqNil{}, &res)
	if err != nil {
		t.Error("Call Version Failed.", err)
	}
	assert.Equal(t, version.GetVersion(), res.Chain33)
}

func testDumpPrivkeyGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.ReplyString
	err := rpc.newRpcCtx("DumpPrivkey", &types.ReqString{}, &res)
	if err != nil {
		t.Error("Call DumpPrivkey Failed.", err)
	}
}

func testGetHexTxByHashGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.HexTx
	err := rpc.newRpcCtx("GetHexTxByHash", &types.ReqHash{Hash: []byte("fdafdsafds")}, &res)
	if err != nil {
		t.Error("Call GetHexTxByHash Failed.", err)
	}
}

func testQueryChainGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.Reply
	err := rpc.newRpcCtx("QueryChain", &types.ChainExecutor{}, &res)
	if err != nil {
		t.Error("Call QueryChain Failed.", err)
	}
}

func testGetBalanceGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.Accounts
	err := rpc.newRpcCtx("GetBalance", &types.ReqBalance{}, &res)
	if err != nil {
		t.Error("Call GetBalance Failed.", err)
	}
}

func testSaveSeedGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.Reply
	err := rpc.newRpcCtx("SaveSeed", &types.SaveSeedByPw{}, &res)
	if err != nil {
		t.Error("Call SaveSeed Failed.", err)
	}
}

func testGetSeedGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.ReplySeed
	err := rpc.newRpcCtx("GetSeed", &types.GetSeedByPw{}, &res)
	if err != nil {
		t.Error("Call GetSeed Failed.", err)
	}
}

func testGenSeedGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.ReplySeed
	err := rpc.newRpcCtx("GenSeed", &types.GenSeedLang{}, &res)
	if err != nil {
		t.Error("Call GenSeed Failed.", err)
	}
}

func testGetBlockHashGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.ReplyHash
	err := rpc.newRpcCtx("GetBlockHash", &types.ReqInt{}, &res)
	if err != nil {
		t.Error("Call GetBlockHash Failed.", err)
	}
}

func testGetAddrOverviewGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.AddrOverview
	err := rpc.newRpcCtx("GetAddrOverview", &types.ReqAddr{Addr: "13cS5G1BDN2YfGudsxRxr7X25yu6ZdgxMU"}, &res)
	if err != nil {
		t.Error("Call GetAddrOverview Failed.", err)
	}
}

func testGetBlockOverviewGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.BlockOverview
	err := rpc.newRpcCtx("GetBlockOverview", &types.ReqHash{}, &res)
	if err != nil {
		t.Error("Call GetBlockOverview Failed.", err)
	}
}

func testGetWalletStatusGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.WalletStatus
	err := rpc.newRpcCtx("GetWalletStatus", &types.ReqNil{}, &res)
	if err != nil {
		t.Error("Call GetWalletStatus Failed.", err)
	}
}

func testGetLastMemPoolGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.ReplyTxList
	err := rpc.newRpcCtx("GetLastMemPool", &types.ReqNil{}, &res)
	if err != nil {
		t.Error("Call GetLastMemPool Failed.", err)
	}
}

func testGetProperFeeGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.ReplyProperFee
	err := rpc.newRpcCtx("GetProperFee", &types.ReqNil{}, &res)
	if err != nil {
		t.Error("Call GetProperFee Failed.", err)
	}
}

func testGetPeerInfoGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.PeerList
	err := rpc.newRpcCtx("GetPeerInfo", &types.ReqNil{}, &res)
	if err != nil {
		t.Error("Call GetPeerInfo Failed.", err)
	}
}

func testUnLockGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.Reply
	err := rpc.newRpcCtx("UnLock", &types.WalletUnLock{}, &res)
	if err != nil {
		t.Error("Call UnLock Failed.", err)
	}
}

func testLockGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.Reply
	err := rpc.newRpcCtx("Lock", &types.ReqNil{}, &res)
	if err != nil {
		t.Error("Call Lock Failed.", err)
	}
}

func testSetPasswdGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.Reply
	err := rpc.newRpcCtx("SetPasswd", &types.ReqWalletSetPasswd{}, &res)
	if err != nil {
		t.Error("Call SetPasswd Failed.", err)
	}
}

func testMergeBalanceGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.ReplyHashes
	err := rpc.newRpcCtx("MergeBalance", &types.ReqWalletMergeBalance{}, &res)
	if err != nil {
		t.Error("Call MergeBalance Failed.", err)
	}
}

func testSetLablGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.WalletAccount
	err := rpc.newRpcCtx("SetLabl", &types.ReqWalletSetLabel{}, &res)
	if err != nil {
		t.Error("Call SetLabl Failed.", err)
	}
}

func testSetTxFeeGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.Reply
	err := rpc.newRpcCtx("SetTxFee", &types.ReqWalletSetFee{}, &res)
	if err != nil {
		t.Error("Call SetTxFee Failed.", err)
	}
}

func testSendToAddressGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.ReplyHash
	err := rpc.newRpcCtx("SendToAddress", &types.ReqWalletSendToAddress{}, &res)
	if err != nil {
		t.Error("Call SendToAddress Failed.", err)
	}
}

func testImportPrivKeyGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.WalletAccount
	err := rpc.newRpcCtx("ImportPrivkey", &types.ReqWalletImportPrivkey{}, &res)
	if err != nil {
		t.Error("Call ImportPrivKey Failed.", err)
	}
}

func testWalletTransactionListGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.WalletTxDetails
	err := rpc.newRpcCtx("WalletTransactionList", &types.ReqWalletTransactionList{}, &res)
	if err != nil {
		t.Error("Call WalletTransactionList Failed.", err)
	}
}

func testNewAccountGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.WalletAccount
	err := rpc.newRpcCtx("NewAccount", &types.ReqNewAccount{}, &res)
	if err != nil {
		t.Error("Call NewAccount Failed.", err)
	}
}

func testGetAccountsGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.WalletAccounts
	err := rpc.newRpcCtx("GetAccounts", &types.ReqNil{}, &res)
	if err != nil {
		t.Error("Call GetAccounts Failed.", err)
	}
}

func testGetMemPoolGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.ReplyTxList
	err := rpc.newRpcCtx("GetMemPool", &types.ReqNil{}, &res)
	if err != nil {
		t.Error("Call GetMemPool Failed.", err)
	}
}

func testGetTransactionByHashesGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.TransactionDetails
	err := rpc.newRpcCtx("GetTransactionByHashes", &types.ReqHashes{}, &res)
	if err != nil {
		t.Error("Call GetTransactionByHashes Failed.", err)
	}
}

func testGetTransactionByAddrGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.ReplyTxInfos
	err := rpc.newRpcCtx("GetTransactionByAddr", &types.ReqAddr{}, &res)
	if err != nil {
		t.Error("Call GetTransactionByAddr Failed.", err)
	}
}

func testSendTransactionGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.Reply
	err := rpc.newRpcCtx("SendTransaction", &types.Transaction{}, &res)
	if err != nil {
		t.Error("Call SendTransaction Failed.", err)
	}
}

func testQueryTransactionGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.TransactionDetail
	err := rpc.newRpcCtx("QueryTransaction", &types.ReqHash{}, &res)
	if err != nil {
		t.Error("Call QueryTransaction Failed.", err)
	}
}

func testCreateRawTransactionGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.UnsignTx
	err := rpc.newRpcCtx("CreateRawTransaction",
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
	err := rpc.newRpcCtx("GetLastHeader", &types.ReqNil{}, &res)
	if err != nil {
		t.Error("Call GetLastHeader Failed.", err)
	}
}

func testGetBlocksGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.Reply
	err := rpc.newRpcCtx("GetBlocks", &types.ReqBlocks{}, &res)
	if err != nil {
		t.Error("Call GetBlocks Failed.", err)
	}
}

func testSendTxGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.Reply
	err := rpc.newRpcCtx("SendTransaction", &types.Transaction{}, &res)
	if err != nil {
		t.Error("Call SendTransaction Failed.", err)
	}
}

func testGetSequenceByHashGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.Int64
	err := rpc.newRpcCtx("GetSequenceByHash", &types.ReqHash{}, &res)
	if err != nil {
		t.Error("Call GetSequenceByHash Failed.", err)
	}
}

func testGetBlockBySeqGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.BlockSeq
	//just for coverage
	err := rpc.newRpcCtx("GetBlockBySeq", &types.Int64{Data: 1}, &res)
	assert.Nil(t, err)

	err = rpc.newRpcCtx("GetBlockBySeq", &types.Int64{Data: 10}, &res)
	assert.NotNil(t, err)

}

func TestGetBlockBySeq(t *testing.T) {
	q := client.QueueProtocol{}
	_, err := q.GetBlockBySeq(nil)
	assert.NotNil(t, err)

}
