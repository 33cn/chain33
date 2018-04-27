package client

import (
	"testing"

	"gitlab.33.cn/chain33/chain33/types"
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

func testSendRawTransaction(t *testing.T, api QueueProtocolAPI) {
	_, err := api.CreateRawTransaction(&types.CreateTx{})
	if nil != err {
		t.Error("Call CreateRawTransaction Failed.", err)
	}
}

func testCreateRawTransaction(t *testing.T, api QueueProtocolAPI) {
	_, err := api.CreateRawTransaction(&types.CreateTx{})
	if nil != err {
		t.Error("Call CreateRawTransaction Failed.", err)
	}
}

func testGetLastHeader(t *testing.T, api QueueProtocolAPI) {
	_, err := api.GetLastHeader()
	if nil != err {
		t.Error("Call GetLastHeader Failed.", err)
	}
}

func testLocalGet(t *testing.T, api QueueProtocolAPI) {
	_, err := api.LocalGet(&types.ReqHash{})
	if nil != err {
		t.Error("Call LocalGet Failed.", err)
	}
}

func testIsNtpClockSync(t *testing.T, api QueueProtocolAPI) {
	_, err := api.IsNtpClockSync()
	if nil != err {
		t.Error("Call IsNtpClockSync Failed.", err)
	}
}

func testIsSync(t *testing.T, api QueueProtocolAPI) {
	_, err := api.IsSync()
	if nil != err {
		t.Error("Call IsSync Failed.", err)
	}
}

func testCloseTickets(t *testing.T, api QueueProtocolAPI) {
	_, err := api.CloseTickets()
	if nil != err {
		t.Error("Call CloseTickets Failed.", err)
	}
}

func testDumpPrivkey(t *testing.T, api QueueProtocolAPI) {
	_, err := api.DumpPrivkey(&types.ReqStr{})
	if nil != err {
		t.Error("Call DumpPrivkey Failed.", err)
	}
}

func testGetTicketCount(t *testing.T, api QueueProtocolAPI) {
	_, err := api.GetTicketCount()
	if nil != err {
		t.Error("Call GetTicketCount Failed.", err)
	}
}

func testWalletAutoMiner(t *testing.T, api QueueProtocolAPI) {
	_, err := api.WalletAutoMiner(&types.MinerFlag{})
	if nil != err {
		t.Error("Call WalletAutoMiner Failed.", err)
	}
}

func testGetWalletStatus(t *testing.T, api QueueProtocolAPI) {
	_, err := api.GetWalletStatus()
	if nil != err {
		t.Error("Call GetWalletStatus Failed.", err)
	}
}

func testGetSeed(t *testing.T, api QueueProtocolAPI) {
	_, err := api.GetSeed(&types.GetSeedByPw{})
	if nil != err {
		t.Error("Call GetSeed Failed.", err)
	}
}

func testSaveSeed(t *testing.T, api QueueProtocolAPI) {
	_, err := api.SaveSeed(&types.SaveSeedByPw{})
	if nil != err {
		t.Error("Call SaveSeed Failed.", err)
	}
}

func testGenSeed(t *testing.T, api QueueProtocolAPI) {
	_, err := api.GenSeed(&types.GenSeedLang{})
	if nil != err {
		t.Error("Call GenSeed Failed.", err)
	}
}

func testGetBlockHash(t *testing.T, api QueueProtocolAPI) {
	_, err := api.GetBlockHash(&types.ReqInt{})
	if nil != err {
		t.Error("Call GetBlockHash Failed.", err)
	}
}

func testGetAddrOverview(t *testing.T, api QueueProtocolAPI) {
	_, err := api.GetAddrOverview(&types.ReqAddr{})
	if nil != err {
		t.Error("Call GetAddrOverview Failed.", err)
	}
}

func testGetBlockOverview(t *testing.T, api QueueProtocolAPI) {
	_, err := api.GetBlockOverview(&types.ReqHash{})
	if nil != err {
		t.Error("Call GetBlockOverview Failed.", err)
	}
}

func testGetLastMempool(t *testing.T, api QueueProtocolAPI) {
	_, err := api.GetLastMempool()
	if nil != err {
		t.Error("Call GetLastMempool Failed.", err)
	}
}

func testGetHeaders(t *testing.T, api QueueProtocolAPI) {
	_, err := api.GetHeaders(&types.ReqBlocks{})
	if nil != err {
		t.Error("Call GetHeaders Failed.", err)
	}
}

func testPeerInfo(t *testing.T, api QueueProtocolAPI) {
	_, err := api.PeerInfo()
	if nil != err {
		t.Error("Call PeerInfo Failed.", err)
	}
}

func testWalletUnLock(t *testing.T, api QueueProtocolAPI) {
	_, err := api.WalletUnLock(&types.WalletUnLock{})
	if nil != err {
		t.Error("Call WalletUnLock Failed.", err)
	}
}

func testWalletLock(t *testing.T, api QueueProtocolAPI) {
	_, err := api.WalletLock()
	if nil != err {
		t.Error("Call WalletLock Failed.", err)
	}
}

func testWalletSetPasswd(t *testing.T, api QueueProtocolAPI) {
	_, err := api.WalletSetPasswd(&types.ReqWalletSetPasswd{})
	if nil != err {
		t.Error("Call WalletSetPasswd Failed.", err)
	}
}

func testWalletMergeBalance(t *testing.T, api QueueProtocolAPI) {
	_, err := api.WalletMergeBalance(&types.ReqWalletMergeBalance{})
	if nil != err {
		t.Error("Call WalletMergeBalance Failed.", err)
	}
}

func testWalletSetLabel(t *testing.T, api QueueProtocolAPI) {
	_, err := api.WalletSetLabel(&types.ReqWalletSetLabel{})
	if nil != err {
		t.Error("Call WalletSetLabel Failed.", err)
	}
}

func testWalletSetFee(t *testing.T, api QueueProtocolAPI) {
	_, err := api.WalletSetFee(&types.ReqWalletSetFee{})
	if nil != err {
		t.Error("Call WalletSetFee Failed.", err)
	}
}

func testWalletSendToAddress(t *testing.T, api QueueProtocolAPI) {
	_, err := api.WalletSendToAddress(&types.ReqWalletSendToAddress{})
	if nil != err {
		t.Error("Call WalletSendToAddress Failed.", err)
	}
}

func testWalletImportprivkey(t *testing.T, api QueueProtocolAPI) {
	_, err := api.WalletImportprivkey(&types.ReqWalletImportPrivKey{})
	if nil != err {
		t.Error("Call WalletTransactionList Failed.", err)
	}
}

func testWalletTransactionList(t *testing.T, api QueueProtocolAPI) {
	_, err := api.WalletTransactionList(&types.ReqWalletTransactionList{})
	if nil != err {
		t.Error("Call WalletTransactionList Failed.", err)
	}
}

func testNewAccount(t *testing.T, api QueueProtocolAPI) {
	_, err := api.NewAccount(&types.ReqNewAccount{})
	if nil != err {
		t.Error("Call NewAccount Failed.", err)
	}
}

func testWalletGetAccountList(t *testing.T, api QueueProtocolAPI) {
	_, err := api.WalletGetAccountList()
	if nil != err {
		t.Error("Call WalletGetAccountList Failed.", err)
	}
}

func testGetMempool(t *testing.T, api QueueProtocolAPI) {
	_, err := api.GetMempool()
	if nil != err {
		t.Error("Call GetMempool Failed.", err)
	}
}

func testGetTransactionByHash(t *testing.T, api QueueProtocolAPI) {
	_, err := api.GetTransactionByHash(&types.ReqHashes{})
	if nil != err {
		t.Error("Call GetTransactionByHash Failed.", err)
	}
}

func testQueryTx(t *testing.T, api QueueProtocolAPI) {
	_, err := api.QueryTx(&types.ReqHash{})
	if nil != err {
		t.Error("Call QueryTx Failed.", err)
	}
}

func testGetTransactionByAddr(t *testing.T, api QueueProtocolAPI) {
	_, err := api.GetTransactionByAddr(&types.ReqAddr{})
	if nil != err {
		t.Error("Call GetTransactionByAddr Failed.", err)
	}
}

func testGetBlocks(t *testing.T, api QueueProtocolAPI) {
	_, err := api.GetBlocks(&types.ReqBlocks{})
	if nil != err {
		t.Error("Call GetBlocks Failed.", err)
	}
}

func testGetTxList(t *testing.T, api QueueProtocolAPI) {
	_, err := api.GetTxList(&types.TxHashList{})
	if nil != err {
		t.Error("Call GetTxList Failed.", err)
	}
}

func testSendTx(t *testing.T, api QueueProtocolAPI) {
	_, err := api.SendTx(&types.Transaction{})
	if nil != err {
		t.Error("Call GetTx Failed.", err)
	}
}

func TestJsonRPC(t *testing.T) {

}

func TestGRPC(t *testing.T) {

}
