package client

import (
	"testing"

	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

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
			default:
				msg.ReplyErr("Do not support", types.ErrNotSupport)
			}
		}
	}()
}

func (m *mockBlockChain) Close() {
}

type mockMempool struct {
	client queue.Client
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
			default:
				msg.ReplyErr("Do not support", types.ErrNotSupport)
			}
		}
	}()
}

func (m *mockMempool) Close() {
}

type mockSystem struct {
	q     queue.Queue
	chain *mockBlockChain
	mem   *mockMempool
}

func (mock *mockSystem) startup(size int) QueueProtocolAPI {
	var q = queue.New("channel")
	chain := &mockBlockChain{}
	chain.SetQueueClient(q)

	mem := &mockMempool{}
	mem.SetQueueClient(q)

	mock.q = q
	mock.chain = chain
	mock.mem = mem
	return mock.getAPI()
}

func (mock *mockSystem) stop() {
	mock.chain.Close()
	mock.mem.Close()
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

	testGetTx(t, api)
	testGetTxList(t, api)
	testGetBlocks(t, api)
	testGetTransactionByAddr(t, api)
	testQueryTx(t, api)
	testGetTransactionByHash(t, api)
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

func testGetTx(t *testing.T, api QueueProtocolAPI) {
	_, err := api.GetTx(&types.Transaction{})
	if nil != err {
		t.Error("Call GetTx Failed.", err)
	}
}

/*

func TestQueueCoordinator_QueryTxList(t *testing.T) {
	var mock mockSystem
	api := mock.startup(0)
	defer mock.stop()

	_, err := api.GetTxList(&types.TxHashList{Count: 2, Hashes: nil})
	if nil != err {
		t.Error("QueryTxList error. ", err)
	}
}

func TestQueueCoordinator_GetMempool(t *testing.T) {
	var mock mockSystem
	api := mock.startup(0)
	defer mock.stop()

	_, err := api.GetMempool()
	if nil != err {
		t.Error("GetMempool error. ", err)
	}
}

func TestQueueCoordinator_GetLastMempool(t *testing.T) {
	var mock mockSystem
	api := mock.startup(0)
	defer mock.stop()

	_, err := api.GetLastMempool(nil)
	if nil != err {
		t.Error("GetMempool error. ", err)
	}
}

func TestQueueCoordinator_GetBlocks(t *testing.T) {
	var mock mockSystem
	api := mock.startup(0)
	defer mock.stop()

	_, err := api.GetBlocks(&types.ReqBlocks{0, 10, false, []string{""}})
	if nil != err {
		t.Error("GetBlocks error. ", err)
	}
}

func TestQueueCoordinator_QueryTx(t *testing.T) {
	var mock mockSystem
	mock.startup(0)
	defer mock.stop()
}

func TestQueueCoordinator_GetTransactionByAddr(t *testing.T) {
}

func TestQueueCoordinator_GetTransactionByHash(t *testing.T) {
}

func TestQueueCoordinator_GetHeaders(t *testing.T) {
}

func TestQueueCoordinator_GetBlockOverview(t *testing.T) {
}

func TestQueueCoordinator_GetAddrOverview(t *testing.T) {
}

func TestQueueCoordinator_GetBlockHash(t *testing.T) {
}

func TestQueueCoordinator_IsSync(t *testing.T) {
}

func TestQueueCoordinator_IsNtpClockSync(t *testing.T) {
}

func TestQueueCoordinator_LocalGet(t *testing.T) {
}

func TestQueueCoordinator_WalletGetAccountList(t *testing.T) {
}

func TestQueueCoordinator_NewAccount(t *testing.T) {
}

func TestQueueCoordinator_WalletTransactionList(t *testing.T) {
}

func TestQueueCoordinator_WalletImportprivkey(t *testing.T) {
}

func TestQueueCoordinator_WalletSendToAddress(t *testing.T) {
}

func TestQueueCoordinator_WalletSetFee(t *testing.T) {
}

func TestQueueCoordinator_WalletSetLabel(t *testing.T) {
}

func TestQueueCoordinator_WalletMergeBalance(t *testing.T) {
}

func TestQueueCoordinator_WalletSetPasswd(t *testing.T) {
}

func TestQueueCoordinator_WalletLock(t *testing.T) {
}

func TestQueueCoordinator_WalletUnLock(t *testing.T) {
}

func TestQueueCoordinator_GenSeed(t *testing.T) {
}

func TestQueueCoordinator_SaveSeed(t *testing.T) {
}

func TestQueueCoordinator_GetSeed(t *testing.T) {
}

func TestQueueCoordinator_GetWalletStatus(t *testing.T) {
}

func TestQueueCoordinator_WalletAutoMiner(t *testing.T) {
}

func TestQueueCoordinator_DumpPrivkey(t *testing.T) {
}

func TestQueueCoordinator_CloseTickets(t *testing.T) {
}

func TestQueueCoordinator_TokenPreCreate(t *testing.T) {
}

func TestQueueCoordinator_TokenFinishCreate(t *testing.T) {
}

func TestQueueCoordinator_TokenRevokeCreate(t *testing.T) {
}

func TestQueueCoordinator_SellToken(t *testing.T) {
}

func TestQueueCoordinator_BuyToken(t *testing.T) {
}

func TestQueueCoordinator_RevokeSellToken(t *testing.T) {
}

func TestQueueCoordinator_PeerInfo(t *testing.T) {
}

func TestQueueCoordinator_GetTicketCount(t *testing.T) {
}
*/
