package client_test

import (
	"testing"

	"fmt"

	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/types"
	lt "gitlab.33.cn/chain33/chain33/types/local"
)

func TestCoordinator(t *testing.T) {
	var mock mockSystem
	api := mock.startup(0)
	defer mock.stop()

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
	testGetBlockOverview(t, api)
	testGetAddrOverview(t, api)
	testGetBlockHash(t, api)
	testGenSeed(t, api)
	testSaveSeed(t, api)
	testGetSeed(t, api)
	testGetWalletStatus(t, api)
	testWalletAutoMiner(t, api)
	testGetTicketCount(t, api)
	testDumpPrivkey(t, api)
	testCloseTickets(t, api)
	testIsSync(t, api)
	testIsNtpClockSync(t, api)
	testLocalGet(t, api)
	testGetLastHeader(t, api)
	testCreateRawTransaction(t, api)
	testSendRawTransaction(t, api)
}

func testSendRawTransaction(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.CreateRawTransaction(&types.CreateTx{})
	if nil != err {
		t.Error("Call CreateRawTransaction Failed.", err)
	}
}

func testCreateRawTransaction(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.CreateRawTransaction(&types.CreateTx{})
	if nil != err {
		t.Error("Call CreateRawTransaction Failed.", err)
	}
}

func testGetLastHeader(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.GetLastHeader()
	if nil != err {
		t.Error("Call GetLastHeader Failed.", err)
	}
}

func testLocalGet(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.LocalGet(&types.ReqHash{})
	if nil != err {
		t.Error("Call LocalGet Failed.", err)
	}
}

func testIsNtpClockSync(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.IsNtpClockSync()
	if nil != err {
		t.Error("Call IsNtpClockSync Failed.", err)
	}
}

func testIsSync(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.IsSync()
	if nil != err {
		t.Error("Call IsSync Failed.", err)
	}
}

func testCloseTickets(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.CloseTickets()
	if nil != err {
		t.Error("Call CloseTickets Failed.", err)
	}
}

func testDumpPrivkey(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.DumpPrivkey(&types.ReqStr{})
	if nil != err {
		t.Error("Call DumpPrivkey Failed.", err)
	}
}

func testGetTicketCount(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.GetTicketCount()
	if nil != err {
		t.Error("Call GetTicketCount Failed.", err)
	}
}

func testWalletAutoMiner(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.WalletAutoMiner(&types.MinerFlag{})
	if nil != err {
		t.Error("Call WalletAutoMiner Failed.", err)
	}
}

func testGetWalletStatus(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.GetWalletStatus()
	if nil != err {
		t.Error("Call GetWalletStatus Failed.", err)
	}
}

func testGetSeed(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.GetSeed(&types.GetSeedByPw{})
	if nil != err {
		t.Error("Call GetSeed Failed.", err)
	}
}

func testSaveSeed(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.SaveSeed(&types.SaveSeedByPw{})
	if nil != err {
		t.Error("Call SaveSeed Failed.", err)
	}
}

func testGenSeed(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.GenSeed(&types.GenSeedLang{})
	if nil != err {
		t.Error("Call GenSeed Failed.", err)
	}
}

func testGetBlockHash(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.GetBlockHash(&types.ReqInt{})
	if nil != err {
		t.Error("Call GetBlockHash Failed.", err)
	}
}

func testGetAddrOverview(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.GetAddrOverview(&types.ReqAddr{})
	if nil != err {
		t.Error("Call GetAddrOverview Failed.", err)
	}
}

func testGetBlockOverview(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.GetBlockOverview(&types.ReqHash{})
	if nil != err {
		t.Error("Call GetBlockOverview Failed.", err)
	}
}

func testGetLastMempool(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.GetLastMempool()
	if nil != err {
		t.Error("Call GetLastMempool Failed.", err)
	}
}

func testGetHeaders(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.GetHeaders(&types.ReqBlocks{})
	if nil != err {
		t.Error("Call GetHeaders Failed.", err)
	}
}

func testPeerInfo(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.PeerInfo()
	if nil != err {
		t.Error("Call PeerInfo Failed.", err)
	}
}

func testWalletUnLock(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.WalletUnLock(&types.WalletUnLock{})
	if nil != err {
		t.Error("Call WalletUnLock Failed.", err)
	}
}

func testWalletLock(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.WalletLock()
	if nil != err {
		t.Error("Call WalletLock Failed.", err)
	}
}

func testWalletSetPasswd(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.WalletSetPasswd(&types.ReqWalletSetPasswd{})
	if nil != err {
		t.Error("Call WalletSetPasswd Failed.", err)
	}
}

func testWalletMergeBalance(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.WalletMergeBalance(&types.ReqWalletMergeBalance{})
	if nil != err {
		t.Error("Call WalletMergeBalance Failed.", err)
	}
}

func testWalletSetLabel(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.WalletSetLabel(&types.ReqWalletSetLabel{})
	if nil != err {
		t.Error("Call WalletSetLabel Failed.", err)
	}
}

func testWalletSetFee(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.WalletSetFee(&types.ReqWalletSetFee{})
	if nil != err {
		t.Error("Call WalletSetFee Failed.", err)
	}
}

func testWalletSendToAddress(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.WalletSendToAddress(&types.ReqWalletSendToAddress{})
	if nil != err {
		t.Error("Call WalletSendToAddress Failed.", err)
	}
}

func testWalletImportprivkey(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.WalletImportprivkey(&types.ReqWalletImportPrivKey{})
	if nil != err {
		t.Error("Call WalletTransactionList Failed.", err)
	}
}

func testWalletTransactionList(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.WalletTransactionList(&types.ReqWalletTransactionList{})
	if nil != err {
		t.Error("Call WalletTransactionList Failed.", err)
	}
}

func testNewAccount(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.NewAccount(&types.ReqNewAccount{})
	if nil != err {
		t.Error("Call NewAccount Failed.", err)
	}
}

func testWalletGetAccountList(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.WalletGetAccountList()
	if nil != err {
		t.Error("Call WalletGetAccountList Failed.", err)
	}
}

func testGetMempool(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.GetMempool()
	if nil != err {
		t.Error("Call GetMempool Failed.", err)
	}
}

func testGetTransactionByHash(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.GetTransactionByHash(&types.ReqHashes{})
	if nil != err {
		t.Error("Call GetTransactionByHash Failed.", err)
	}
}

func testQueryTx(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.QueryTx(&types.ReqHash{})
	if nil != err {
		t.Error("Call QueryTx Failed.", err)
	}
}

func testGetTransactionByAddr(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.GetTransactionByAddr(&types.ReqAddr{})
	if nil != err {
		t.Error("Call GetTransactionByAddr Failed.", err)
	}
}

func testGetBlocks(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.GetBlocks(&types.ReqBlocks{})
	if nil != err {
		t.Error("Call GetBlocks Failed.", err)
	}
}

func testGetTxList(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.GetTxList(&types.TxHashList{})
	if nil != err {
		t.Error("Call GetTxList Failed.", err)
	}
}

func testSendTx(t *testing.T, api client.QueueProtocolAPI) {
	_, err := api.SendTx(&types.Transaction{})
	if nil != err {
		t.Error("Call GetTx Failed.", err)
	}
}

func TestJsonRPC(t *testing.T) {
	var mock mockSystem
	var jrpc mockJRPCSystem
	mock.MockLive = &jrpc
	mock.startup(0)
	defer mock.stop()

	testGetBlocksJsonRPC(t, &jrpc)
	testGetBlockOverviewJsonRPC(t, &jrpc)
	testGetBlockHashJsonRPC(t, &jrpc)
	testGetHeadersCmdJsonRPC(t, &jrpc)
	testGetLastHeaderJsonRPC(t, &jrpc)
	testGetMempoolJsonRPC(t, &jrpc)
	testGetLastMemPoolJsonRPC(t, &jrpc)
	testGenSeedsonRPC(t, &jrpc)
	testGetPeerInfoJsonRPC(t, &jrpc)
	testIsNtpClockSyncJsonRPC(t, &jrpc)
	testIsSyncJsonRPC(t, &jrpc)
	testGetNetInfoJsonRPC(t, &jrpc)
}

func testGetNetInfoJsonRPC(t *testing.T, rpc *mockJRPCSystem) {
	var res lt.NodeNetinfo
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
	var res lt.ReplyTxList
	err := rpc.newRpcCtx("Chain33.GetLastMemPool",
		nil, &res)
	if err != nil {
		t.Error("testGetLastMemPoolJsonRPC failed. Error", err)
	}
}

func testGetMempoolJsonRPC(t *testing.T, rpc *mockJRPCSystem) {
	var res lt.ReplyTxList
	err := rpc.newRpcCtx("Chain33.GetMempool",
		nil, &res)
	if err != nil {
		t.Error("testGetMempoolJsonRPC failed. Error", err)
	}
}

func testGetLastHeaderJsonRPC(t *testing.T, rpc *mockJRPCSystem) {
	var res lt.Header
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
		Isdetail: true,
	}

	var res lt.Headers
	err := rpc.newRpcCtx("Chain33.GetHeaders",
		params, &res)
	if err != nil {
		t.Error("testGetHeadersCmdJsonRPC failed. Error", err)
	}
}

func testGetBlockOverviewJsonRPC(t *testing.T, rpc *mockJRPCSystem) {
	params := lt.QueryParm{
		Hash: "0x67c58d6ba9175313f0468ae4e0ddec946549af7748037c2fdd5d54298afd20b6",
	}

	var res lt.BlockOverview
	err := rpc.newRpcCtx("Chain33.GetBlockOverview",
		params, &res)
	if err != nil {
		t.Error("testGetBlockOverviewJsonRPC failed. Error", err)
	}
}

func testGetBlocksJsonRPC(t *testing.T, rpc *mockJRPCSystem) {
	params := lt.BlockParam{
		Start:    1,
		End:      1,
		Isdetail: true,
	}

	var res lt.BlockDetails
	err := rpc.newRpcCtx("Chain33.GetBlocks",
		params, &res)
	if err != nil {
		t.Error("testGetBlocksJsonRPC failed. Error", err)
	}
}

func testGetBlockHashJsonRPC(t *testing.T, rpc *mockJRPCSystem) {
	params := types.ReqInt{
		Height: 10,
	}
	var res lt.ReplyHash
	err := rpc.newRpcCtx("Chain33.GetBlockHash",
		params, &res)
	if err != nil {
		t.Error("testGetBlockHashJsonRPC failed. Error", err)
	}
}

func TestGRPC(t *testing.T) {
	var mock mockSystem
	var grpc mockGRPCSystem
	mock.MockLive = &grpc
	mock.startup(0)

	defer mock.stop()

	testSendTxGRPC(t, &grpc)
	testGetBlocksGRPC(t, &grpc)
	testGetLastHeaderGRPC(t, &grpc)
	testCreateRawTransactionGRPC(t, &grpc)
	testSendRawTransactionGRPC(t, &grpc)
	testQueryTransactionGRPC(t, &grpc)
	testSendTransactionGRPC(t, &grpc)
	testGetTransactionByAddrGRPC(t, &grpc)
	testGetTransactionByHashesGRPC(t, &grpc)
	testGetMemPoolGRPC(t, &grpc)
	testGetAccountsGRPC(t, &grpc)
	testNewAccountGRPC(t, &grpc)
	testWalletTransactionListGRPC(t, &grpc)
	testImportPrivKeyGRPC(t, &grpc)
	testSendToAddressGRPC(t, &grpc)
	testSetTxFeeGRPC(t, &grpc)
	testSetLablGRPC(t, &grpc)
	testMergeBalanceGRPC(t, &grpc)
	testSetPasswdGRPC(t, &grpc)
	testLockGRPC(t, &grpc)
	testUnLockGRPC(t, &grpc)
}

func testUnLockGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.Reply
	err := rpc.newRpcCtx("UnLock", &types.WalletUnLock{}, &res)
	if nil != err {
		t.Error("Call UnLock Failed.", err)
	} else {
		fmt.Println(res)
	}
}

func testLockGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.Reply
	err := rpc.newRpcCtx("Lock", &types.ReqNil{}, &res)
	if nil != err {
		t.Error("Call Lock Failed.", err)
	} else {
		fmt.Println(res)
	}
}

func testSetPasswdGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.Reply
	err := rpc.newRpcCtx("SetPasswd", &types.ReqWalletSetPasswd{}, &res)
	if nil != err {
		t.Error("Call SetPasswd Failed.", err)
	} else {
		fmt.Println(res)
	}
}

func testMergeBalanceGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.ReplyHashes
	err := rpc.newRpcCtx("MergeBalance", &types.ReqWalletMergeBalance{}, &res)
	if nil != err {
		t.Error("Call MergeBalance Failed.", err)
	} else {
		fmt.Println(res)
	}
}

func testSetLablGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.WalletAccount
	err := rpc.newRpcCtx("SetLabl", &types.ReqWalletSetLabel{}, &res)
	if nil != err {
		t.Error("Call SetLabl Failed.", err)
	} else {
		fmt.Println(res)
	}
}

func testSetTxFeeGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.Reply
	err := rpc.newRpcCtx("SetTxFee", &types.ReqWalletSetFee{}, &res)
	if nil != err {
		t.Error("Call SetTxFee Failed.", err)
	} else {
		fmt.Println(res)
	}
}

func testSendToAddressGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.ReplyHash
	err := rpc.newRpcCtx("SendToAddress", &types.ReqWalletSendToAddress{}, &res)
	if nil != err {
		t.Error("Call SendToAddress Failed.", err)
	} else {
		fmt.Println(res)
	}
}

func testImportPrivKeyGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.WalletAccount
	err := rpc.newRpcCtx("ImportPrivKey", &types.ReqWalletImportPrivKey{}, &res)
	if nil != err {
		t.Error("Call ImportPrivKey Failed.", err)
	} else {
		fmt.Println(res)
	}
}

func testWalletTransactionListGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.WalletTxDetails
	err := rpc.newRpcCtx("WalletTransactionList", &types.ReqWalletTransactionList{}, &res)
	if nil != err {
		t.Error("Call WalletTransactionList Failed.", err)
	} else {
		fmt.Println(res)
	}
}

func testNewAccountGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.WalletAccount
	err := rpc.newRpcCtx("NewAccount", &types.ReqNewAccount{}, &res)
	if nil != err {
		t.Error("Call NewAccount Failed.", err)
	} else {
		fmt.Println(res)
	}
}

func testGetAccountsGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.WalletAccounts
	err := rpc.newRpcCtx("GetAccounts", &types.ReqNil{}, &res)
	if nil != err {
		t.Error("Call GetAccounts Failed.", err)
	} else {
		fmt.Println(res)
	}
}

func testGetMemPoolGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.ReplyTxList
	err := rpc.newRpcCtx("GetMemPool", &types.ReqNil{}, &res)
	if nil != err {
		t.Error("Call GetMemPool Failed.", err)
	} else {
		fmt.Println(res)
	}
}

func testGetTransactionByHashesGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.TransactionDetails
	err := rpc.newRpcCtx("GetTransactionByHashes", &types.ReqHashes{}, &res)
	if nil != err {
		t.Error("Call GetTransactionByHashes Failed.", err)
	} else {
		fmt.Println(res)
	}
}

func testGetTransactionByAddrGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.ReplyTxInfos
	err := rpc.newRpcCtx("GetTransactionByAddr", &types.ReqAddr{}, &res)
	if nil != err {
		t.Error("Call GetTransactionByAddr Failed.", err)
	} else {
		fmt.Println(res)
	}
}

func testSendTransactionGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.Reply
	err := rpc.newRpcCtx("SendTransaction", &types.Transaction{}, &res)
	if nil != err {
		t.Error("Call SendTransaction Failed.", err)
	} else {
		fmt.Println(res)
	}
}

func testQueryTransactionGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.TransactionDetail
	err := rpc.newRpcCtx("QueryTransaction", &types.ReqHash{}, &res)
	if nil != err {
		t.Error("Call QueryTransaction Failed.", err)
	} else {
		fmt.Println(res)
	}
}

func testSendRawTransactionGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.Reply
	err := rpc.newRpcCtx("SendRawTransaction", &types.SignedTx{}, &res)
	if nil != err {
		t.Error("Call SendRawTransaction Failed.", err)
	} else {
		fmt.Println(res)
	}
}

func testCreateRawTransactionGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.UnsignTx
	err := rpc.newRpcCtx("CreateRawTransaction", &types.CreateTx{}, &res)
	if nil != err {
		t.Error("Call CreateRawTransaction Failed.", err)
	} else {
		fmt.Println(res)
	}
}

func testGetLastHeaderGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.Header
	err := rpc.newRpcCtx("GetLastHeader", &types.ReqNil{}, &res)
	if nil != err {
		t.Error("Call GetLastHeader Failed.", err)
	} else {
		fmt.Println(res)
	}
}

func testGetBlocksGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.Reply
	err := rpc.newRpcCtx("GetBlocks", &types.ReqBlocks{}, &res)
	if nil != err {
		t.Error("Call GetBlocks Failed.", err)
	} else {
		fmt.Println(res)
	}
}

func testSendTxGRPC(t *testing.T, rpc *mockGRPCSystem) {
	var res types.Reply
	err := rpc.newRpcCtx("SendTransaction", &types.Transaction{}, &res)
	if nil != err {
		t.Error("Call SendTransaction Failed.", err)
	} else {
		fmt.Println(res)
	}
}
