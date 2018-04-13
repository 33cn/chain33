package client_test

import (
	"testing"
	"flag"
	"math/rand"

	_ "gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/blockchain"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/config"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	_ "gitlab.33.cn/chain33/chain33/common/limits"
	"gitlab.33.cn/chain33/chain33/consensus"
	"gitlab.33.cn/chain33/chain33/executor"
	_ "gitlab.33.cn/chain33/chain33/p2p"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/store"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/mempool"
	"gitlab.33.cn/chain33/chain33/client"
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
	hex        = "CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944"
	a, _       = common.FromHex(hex)
	privKey, _ = c.PrivKeyFromBytes(a)
	random     *rand.Rand
	mainPriv   crypto.PrivKey
)

var blk = &types.Block{
	Version:    1,
	ParentHash: []byte("parent hash"),
	TxHash:     []byte("tx hash"),
	Height:     1,
	BlockTime:  1,
	Txs:        []*types.Transaction{tx3, tx5},
}

type mockSystem struct {
	q		queue.Queue
	mem		*mempool.Mempool
	exec	*executor.Executor
	chain	*blockchain.BlockChain
	store	queue.Module
	cons	queue.Module
	coord	*client.QueueCoordinator
}

func (mock *mockSystem) startup(size int) client.QueueProtocolAPI {
	var q = queue.New("channel")
	flag.Parse()
	cfg := config.InitCfg("../chain33.toml")
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

	coord := client.New()
	coord.SetQueueClient(q.Client())

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
	mock.coord = coord

	return mock.getAPI()
}

func (mock *mockSystem) stop() {
	mock.cons.Close()
	mock.store.Close()
	mock.chain.Close()
	mock.exec.Close()
	mock.coord.Close()
	mock.q.Close()
}

func (mock *mockSystem) getAPI() client.QueueProtocolAPI  {
	return mock.coord
}

func TestQueueCoordinator_QueryTx(t *testing.T) {
	var mock mockSystem
	api := mock.startup(0)
	defer mock.stop()

	_, err := api.QueryTx(tx1)
	if nil != err {
		t.Error("QueryTx error. ", err)
	}
}

func TestQueueCoordinator_QueryTxList(t *testing.T) {
	var mock mockSystem
	api := mock.startup(0)
	defer mock.stop()

	_, err := api.QueryTxList(2, nil)
	if nil != err {
		t.Error("QueryTxList error. ", err)
	}
}