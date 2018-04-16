package client

import (
	"flag"
	_ "gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/blockchain"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/config"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	_ "gitlab.33.cn/chain33/chain33/common/limits"
	"gitlab.33.cn/chain33/chain33/consensus"
	"gitlab.33.cn/chain33/chain33/executor"
	"gitlab.33.cn/chain33/chain33/mempool"
	_ "gitlab.33.cn/chain33/chain33/p2p"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/store"
	"gitlab.33.cn/chain33/chain33/types"
	"testing"
)

//----------------------------- data for testing ---------------------------------
var (
	amount   = int64(1e8)
	v        = &types.CoinsAction_Transfer{&types.CoinsTransfer{Amount: amount}}
	transfer = &types.CoinsAction{Value: v, Ty: types.CoinsActionTransfer}
	tx1      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 1000000, Expire: 1}
	tx2      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 100000000, Expire: 0}
	tx3      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 200000000, Expire: 0}
	tx4      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 300000000, Expire: 0}
	tx5      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 400000000, Expire: 0}
	tx6      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 500000000, Expire: 0}
	tx7      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 600000000, Expire: 0}
	tx8      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 700000000, Expire: 0}
	tx9      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 800000000, Expire: 0}
	tx10     = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 900000000, Expire: 0}
	tx11     = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 450000000, Expire: 0}
	tx12     = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 460000000, Expire: 0}

	c, _       = crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	hex_val    = "CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944"
	a, _       = common.FromHex(hex_val)
	privKey, _ = c.PrivKeyFromBytes(a)
	//tx_hash	   = "611dde807369601588ed1b901f116c9b647b960e55f0798603ce0f47c271f985"
	//tx_hash_b, _  = common.FromHex(tx_hash)
	//random     *rand.Rand
	//mainPriv   crypto.PrivKey
)

/*
var blk = &types.Block{
	Version:    1,
	ParentHash: []byte("parent hash"),
	TxHash:     []byte("tx hash"),
	Height:     1,
	BlockTime:  1,
	Txs:        []*types.Transaction{tx3, tx5},
}
*/

type mockSystem struct {
	q     queue.Queue
	mem   *mempool.Mempool
	exec  *executor.Executor
	chain *blockchain.BlockChain
	store queue.Module
	cons  queue.Module
}

func (mock *mockSystem) startup(size int) QueueProtocolAPI {
	var q = queue.New("channel")
	flag.Parse()
	cfg := config.InitCfg("chain33.toml")
	chain := blockchain.New(cfg.BlockChain)
	chain.SetQueueClient(q.Client())

	exec := executor.New(cfg.Exec)
	exec.SetQueueClient(q.Client())

	s := store.New(cfg.Store)
	s.SetQueueClient(q.Client())

	cs := consensus.New(cfg.Consensus)
	cs.SetQueueClient(q.Client())

	mem := mempool.New(cfg.MemPool)
	mem.SetQueueClient(q.Client())
	//mem.sync = true
	if size > 0 {
		mem.Resize(size)
	}
	mem.SetMinFee(types.MinFee)

	tx1.Sign(types.SECP256K1, privKey)
	tx2.Sign(types.SECP256K1, privKey)
	tx3.Sign(types.SECP256K1, privKey)
	tx4.Sign(types.SECP256K1, privKey)
	tx5.Sign(types.SECP256K1, privKey)
	tx6.Sign(types.SECP256K1, privKey)
	tx7.Sign(types.SECP256K1, privKey)
	tx8.Sign(types.SECP256K1, privKey)
	tx9.Sign(types.SECP256K1, privKey)
	tx10.Sign(types.SECP256K1, privKey)
	tx11.Sign(types.SECP256K1, privKey)
	tx12.Sign(types.SECP256K1, privKey)

	mock.q = q
	mock.exec = exec
	mock.chain = chain
	mock.store = s
	mock.cons = cs
	mock.mem = mem

	return mock.getAPI()
}

func (mock *mockSystem) stop() {
	mock.cons.Close()
	mock.store.Close()
	mock.chain.Close()
	mock.exec.Close()
	mock.mem.Close()
	mock.q.Close()
}

func (mock *mockSystem) getAPI() QueueProtocolAPI {
	api, _ := NewQueueAPI(mock.q.Client())
	return api
}

func TestQueueCoordinator_GetTx(t *testing.T) {
	var mock mockSystem
	api := mock.startup(0)
	defer mock.stop()

	_, err := api.GetTx(tx1)
	if nil != err {
		t.Error("QueryTx error. ", err)
	}
}

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

	// 需要获取正确的交易hash值
	/*
		_, err := api.QueryTx(&types.ReqHash{tx_hash_b})
		if nil != err {
			t.Error("QueryTx error. ", err)
		}
	*/
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
	var mock mockSystem
	api := mock.startup(0)
	defer mock.stop()

	_, err := api.GetTicketCount()
	if nil != err {
		t.Error("Call GetTickedCount failed. error: ", err)
	}
}
