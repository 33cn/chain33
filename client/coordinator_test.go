package client

import (
	"testing"

	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

type mockConsensus struct {
}

func (m *mockConsensus) SetQueueClient(q queue.Queue) {
	go func() {
		client := q.Client()
		client.Sub(consensusKey)
		for msg := range client.Recv() {
			switch msg.Ty {
			case types.EventGetTicketCount:
				msg.Reply(client.NewMessage(consensusKey, msg.Ty, &types.Int64{}))
			default:
				msg.ReplyErr("Do not support", types.ErrNotSupport)
			}
		}
	}()
}

func (m *mockConsensus) Close() {
}

type mockP2P struct {
}

func (m *mockP2P) SetQueueClient(q queue.Queue) {
	go func() {
		client := q.Client()
		client.Sub(p2pKey)
		for msg := range client.Recv() {
			switch msg.Ty {
			case types.EventPeerInfo:
				msg.Reply(client.NewMessage(mempoolKey, msg.Ty, &types.PeerList{}))
			default:
				msg.ReplyErr("Do not support", types.ErrNotSupport)
			}
		}
	}()
}

func (m *mockP2P) Close() {
}

type mockBlockChain struct {
}

func (m *mockBlockChain) SetQueueClient(q queue.Queue) {
	go func() {
		client := q.Client()
		client.Sub(blockchainKey)
		for msg := range client.Recv() {
			switch msg.Ty {
			case types.EventGetBlocks:
				msg.Reply(client.NewMessage(blockchainKey, msg.Ty, &types.BlockDetails{}))
			case types.EventGetTransactionByAddr:
				msg.Reply(client.NewMessage(blockchainKey, msg.Ty, &types.ReplyTxInfos{}))
			case types.EventQueryTx:
				msg.Reply(client.NewMessage(blockchainKey, msg.Ty, &types.TransactionDetail{}))
			case types.EventGetTransactionByHash:
				msg.Reply(client.NewMessage(blockchainKey, msg.Ty, &types.TransactionDetails{}))
			case types.EventGetHeaders:
				msg.Reply(client.NewMessage(blockchainKey, msg.Ty, &types.Headers{}))
			case types.EventGetBlockOverview:
				msg.Reply(client.NewMessage(blockchainKey, msg.Ty, &types.BlockOverview{}))
			case types.EventGetAddrOverview:
				msg.Reply(client.NewMessage(blockchainKey, msg.Ty, &types.AddrOverview{}))
			case types.EventGetBlockHash:
				msg.Reply(client.NewMessage(blockchainKey, msg.Ty, &types.ReplyHash{}))
			case types.EventIsSync:
				msg.Reply(client.NewMessage(blockchainKey, msg.Ty, &types.IsCaughtUp{}))
			case types.EventIsNtpClockSync:
				msg.Reply(client.NewMessage(walletKey, msg.Ty, &types.IsNtpClockSync{}))
			case types.EventGetLastHeader:
				msg.Reply(client.NewMessage(walletKey, msg.Ty, &types.Header{}))
			default:
				msg.ReplyErr("Do not support", types.ErrNotSupport)
			}
		}
	}()
}

func (m *mockBlockChain) Close() {
}

type mockMempool struct {
}

func (m *mockMempool) SetQueueClient(q queue.Queue) {
	go func() {
		client := q.Client()
		client.Sub(mempoolKey)
		for msg := range client.Recv() {
			switch msg.Ty {
			case types.EventTx:
				msg.Reply(client.NewMessage(mempoolKey, msg.Ty, &types.Reply{IsOk: true, Msg: []byte("word")}))
			case types.EventTxList:
				msg.Reply(client.NewMessage(mempoolKey, msg.Ty, &types.ReplyTxList{}))
			case types.EventGetMempool:
				msg.Reply(client.NewMessage(mempoolKey, msg.Ty, &types.ReplyTxList{}))
			case types.EventGetLastMempool:
				msg.Reply(client.NewMessage(mempoolKey, msg.Ty, &types.ReplyTxList{}))
			default:
				msg.ReplyErr("Do not support", types.ErrNotSupport)
			}
		}
	}()
}

func (m *mockMempool) Close() {
}

type mockWallet struct {
}

func (m *mockWallet) SetQueueClient(q queue.Queue) {
	go func() {
		client := q.Client()
		client.Sub(walletKey)
		for msg := range client.Recv() {
			switch msg.Ty {
			case types.EventWalletGetAccountList:
				msg.Reply(client.NewMessage(walletKey, msg.Ty, &types.WalletAccounts{}))
			case types.EventNewAccount:
				msg.Reply(client.NewMessage(walletKey, msg.Ty, &types.WalletAccount{}))
			case types.EventWalletTransactionList:
				msg.Reply(client.NewMessage(walletKey, msg.Ty, &types.WalletTxDetails{}))
			case types.EventWalletImportprivkey:
				msg.Reply(client.NewMessage(walletKey, msg.Ty, &types.WalletAccount{}))
			case types.EventWalletSendToAddress:
				msg.Reply(client.NewMessage(walletKey, msg.Ty, &types.ReplyHash{}))
			case types.EventWalletSetFee:
				msg.Reply(client.NewMessage(walletKey, msg.Ty, &types.Reply{IsOk: true}))
			case types.EventWalletSetLabel:
				msg.Reply(client.NewMessage(walletKey, msg.Ty, &types.WalletAccount{}))
			case types.EventWalletMergeBalance:
				msg.Reply(client.NewMessage(walletKey, msg.Ty, &types.ReplyHashes{}))
			case types.EventWalletSetPasswd:
				msg.Reply(client.NewMessage(walletKey, msg.Ty, &types.Reply{}))
			case types.EventGenSeed:
				msg.Reply(client.NewMessage(walletKey, msg.Ty, &types.ReplySeed{}))
			case types.EventSaveSeed:
				msg.Reply(client.NewMessage(walletKey, msg.Ty, &types.Reply{}))
			case types.EventGetSeed:
				msg.Reply(client.NewMessage(walletKey, msg.Ty, &types.ReplySeed{}))
			case types.EventGetWalletStatus:
				msg.Reply(client.NewMessage(walletKey, msg.Ty, &types.WalletStatus{}))
			case types.EventWalletAutoMiner:
				msg.Reply(client.NewMessage(walletKey, msg.Ty, &types.Reply{}))
			case types.EventDumpPrivkey:
				msg.Reply(client.NewMessage(walletKey, msg.Ty, &types.ReplyStr{}))
			case types.EventCloseTickets:
				msg.Reply(client.NewMessage(walletKey, msg.Ty, &types.ReplyHashes{}))
			case types.EventTokenPreCreate:
				msg.Reply(client.NewMessage(walletKey, msg.Ty, &types.ReplyHash{}))
			case types.EventTokenFinishCreate:
				msg.Reply(client.NewMessage(walletKey, msg.Ty, &types.ReplyHash{}))
			case types.EventTokenRevokeCreate:
				msg.Reply(client.NewMessage(walletKey, msg.Ty, &types.ReplyHash{}))
			case types.EventSellToken:
				msg.Reply(client.NewMessage(walletKey, msg.Ty, &types.Reply{}))
			case types.EventBuyToken:
				msg.Reply(client.NewMessage(walletKey, msg.Ty, &types.Reply{}))
			case types.EventRevokeSellToken:
				msg.Reply(client.NewMessage(walletKey, msg.Ty, &types.Reply{}))
			case types.EventLocalGet:
				msg.Reply(client.NewMessage(walletKey, msg.Ty, &types.LocalReplyValue{}))
			default:
				msg.ReplyErr("Do not support", types.ErrNotSupport)
			}
		}
	}()
}

func (m *mockWallet) Close() {
}

type mockSystem struct {
	q         queue.Queue
	chain     *mockBlockChain
	mem       *mockMempool
	wallet    *mockWallet
	p2p       *mockP2P
	consensus *mockConsensus
}

func (mock *mockSystem) startup(size int) QueueProtocolAPI {
	var q = queue.New("channel")
	chain := &mockBlockChain{}
	chain.SetQueueClient(q)
	mem := &mockMempool{}
	mem.SetQueueClient(q)
	wallet := &mockWallet{}
	wallet.SetQueueClient(q)
	p2p := &mockP2P{}
	p2p.SetQueueClient(q)
	consensus := &mockConsensus{}
	consensus.SetQueueClient(q)

	mock.q = q
	mock.chain = chain
	mock.mem = mem
	mock.wallet = wallet
	mock.p2p = p2p
	mock.consensus = consensus
	return mock.getAPI()
}

func (mock *mockSystem) stop() {
	mock.chain.Close()
	mock.mem.Close()
	mock.wallet.Close()
	mock.p2p.Close()
	mock.consensus.Close()
	mock.q.Close()
}

func (mock *mockSystem) getAPI() QueueProtocolAPI {
	api, _ := NewQueueAPI(mock.q.Client())
	return api
}

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
	testTokenPreCreate(t, api)
	testTokenFinishCreate(t, api)
	testTokenRevokeCreate(t, api)
	testSellToken(t, api)
	testBuyToken(t, api)
	testRevokeSellToken(t, api)
	testIsNtpClockSync(t, api)
	testLocalGet(t, api)
	testGetLastHeader(t, api)
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

func testRevokeSellToken(t *testing.T, api QueueProtocolAPI) {
	_, err := api.RevokeSellToken(&types.ReqRevokeSell{})
	if nil != err {
		t.Error("Call RevokeSellToken Failed.", err)
	}
}

func testBuyToken(t *testing.T, api QueueProtocolAPI) {
	_, err := api.BuyToken(&types.ReqBuyToken{})
	if nil != err {
		t.Error("Call BuyToken Failed.", err)
	}
}

func testSellToken(t *testing.T, api QueueProtocolAPI) {
	_, err := api.SellToken(&types.ReqSellToken{})
	if nil != err {
		t.Error("Call SellToken Failed.", err)
	}
}

func testTokenRevokeCreate(t *testing.T, api QueueProtocolAPI) {
	_, err := api.TokenRevokeCreate(&types.ReqTokenRevokeCreate{})
	if nil != err {
		t.Error("Call TokenRevokeCreate Failed.", err)
	}
}

func testTokenFinishCreate(t *testing.T, api QueueProtocolAPI) {
	_, err := api.TokenFinishCreate(&types.ReqTokenFinishCreate{})
	if nil != err {
		t.Error("Call TokenFinishCreate Failed.", err)
	}
}

func testTokenPreCreate(t *testing.T, api QueueProtocolAPI) {
	_, err := api.TokenPreCreate(&types.ReqTokenPreCreate{})
	if nil != err {
		t.Error("Call TokenPreCreate Failed.", err)
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
	_, err := api.GetLastMempool(&types.ReqNil{})
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
	_, err := api.WalletUnLock()
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
