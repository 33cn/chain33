package executor_test

import (
	//"errors"
	"math/rand"
	"testing"
	"time"

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/blockchain"
	"gitlab.33.cn/chain33/chain33/common/config"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/limits"
	"gitlab.33.cn/chain33/chain33/common/log"
	"gitlab.33.cn/chain33/chain33/common/merkle"
	"gitlab.33.cn/chain33/chain33/consensus"
	"gitlab.33.cn/chain33/chain33/executor"
	"gitlab.33.cn/chain33/chain33/mempool"
	"gitlab.33.cn/chain33/chain33/p2p"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/store"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/util"
)

var random *rand.Rand
var zeroHash [32]byte

func init() {
	err := limits.SetLimits()
	if err != nil {
		panic(err)
	}
	random = rand.New(rand.NewSource(time.Now().UnixNano()))
	log.SetLogLevel("error")
}

func initEnv() (queue.Queue, *blockchain.BlockChain, queue.Module, queue.Module, *p2p.P2p, *mempool.Mempool) {
	var q = queue.New("channel")
	cfg := config.InitCfg("../cmd/chain33/chain33.test.toml")
	chain := blockchain.New(cfg.BlockChain)
	chain.SetQueueClient(q.Client())

	exec := executor.New(cfg.Exec)
	exec.SetQueueClient(q.Client())
	types.SetMinFee(0)
	s := store.New(cfg.Store)
	s.SetQueueClient(q.Client())

	cs := consensus.New(cfg.Consensus)
	cs.SetQueueClient(q.Client())

	p2pnet := p2p.New(cfg.P2P)
	p2pnet.SetQueueClient(q.Client())

	mem := mempool.New(cfg.MemPool)
	mem.SetQueueClient(q.Client())

	return q, chain, s, cs, p2pnet, mem
}

func createTx(priv crypto.PrivKey, to string, amount int64) *types.Transaction {
	v := &types.CoinsAction_Transfer{&types.CoinsTransfer{Amount: amount}}
	transfer := &types.CoinsAction{Value: v, Ty: types.CoinsActionTransfer}
	tx := &types.Transaction{Execer: []byte("none"), Payload: types.Encode(transfer), Fee: 1e6, To: to}
	tx.Nonce = random.Int63()
	tx.To = account.ExecAddress("none").String()
	tx.Sign(types.SECP256K1, priv)
	return tx
}

func genaddress() (string, crypto.PrivKey) {
	cr, err := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	if err != nil {
		panic(err)
	}
	privto, err := cr.GenKey()
	if err != nil {
		panic(err)
	}
	addrto := account.PubKeyToAddress(privto.PubKey().Bytes())
	return addrto.String(), privto
}

func genTxs(n int64) (txs []*types.Transaction) {
	_, priv := genaddress()
	to, _ := genaddress()
	for i := 0; i < int(n); i++ {
		txs = append(txs, createTx(priv, to, types.Coin*(n+1)))
	}
	return txs
}

func createBlock(n int64) *types.Block {
	newblock := &types.Block{}
	newblock.Height = -1
	newblock.BlockTime = time.Now().Unix()
	newblock.ParentHash = zeroHash[:]
	newblock.Txs = genTxs(n)
	newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
	return newblock
}

func TestExecBlock(t *testing.T) {
	q, chain, s, cs, p2pnet, mem := initEnv()
	defer chain.Close()
	defer s.Close()
	defer q.Close()
	defer cs.Close()
	defer p2pnet.Close()
	defer mem.Close()
	block := createBlock(10)
	util.ExecBlock(q.Client(), zeroHash[:], block, false, true)
}

//gen 1万币需要 2s，主要是签名的花费
func BenchmarkGenRandBlock(b *testing.B) {
	for i := 0; i < b.N; i++ {
		createBlock(10000)
	}
}

func BenchmarkExecBlock(b *testing.B) {
	q, chain, s, cs, p2pnet, mem := initEnv()
	defer chain.Close()
	defer s.Close()
	defer q.Close()
	defer cs.Close()
	defer p2pnet.Close()
	defer mem.Close()
	block := createBlock(10000)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		util.ExecBlock(q.Client(), zeroHash[:], block, false, true)
	}
}
