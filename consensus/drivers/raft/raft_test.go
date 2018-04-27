package raft

import (
	"flag"
	"math/rand"
	"testing"
	"time"

	"encoding/binary"
	"os"

	"gitlab.33.cn/chain33/chain33/blockchain"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/config"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/limits"
	"gitlab.33.cn/chain33/chain33/common/log"
	"gitlab.33.cn/chain33/chain33/executor"
	"gitlab.33.cn/chain33/chain33/mempool"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/store"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	random    *rand.Rand
	txNumber  int = 10
	loopCount int = 10
)

func init() {
	err := limits.SetLimits()
	if err != nil {
		panic(err)
	}
	random = rand.New(rand.NewSource(time.Now().UnixNano()))
	log.SetLogLevel("info")
}

func TestRaftPerf(t *testing.T) {
	q, chain, s, mem, exec, cs := initEnvRaft()

	defer chain.Close()
	defer mem.Close()
	defer exec.Close()
	defer s.Close()
	defer cs.Close()
	defer q.Close()

	sendReplyList(q)
}

func initEnvRaft() (queue.Queue, *blockchain.BlockChain, queue.Module, *mempool.Mempool, queue.Module, *RaftClient) {
	var q = queue.New("channel")
	flag.Parse()
	cfg := config.InitCfg("chain33.test.toml")
	chain := blockchain.New(cfg.BlockChain)
	chain.SetQueueClient(q.Client())

	exec := executor.New(cfg.Exec)
	exec.SetQueueClient(q.Client())
	types.SetMinFee(0)
	s := store.New(cfg.Store)
	s.SetQueueClient(q.Client())

	cs := NewRaftCluster(cfg.Consensus)
	cs.SetQueueClient(q.Client())

	mem := mempool.New(cfg.MemPool)
	mem.SetQueueClient(q.Client())

	return q, chain, s, mem, exec, cs
}

func generateKey(i, valI int) string {
	key := make([]byte, valI)
	binary.PutUvarint(key[:10], uint64(valI))
	binary.PutUvarint(key[12:24], uint64(i))
	if _, err := rand.Read(key[24:]); err != nil {
		os.Exit(1)
	}
	return string(key)
}

func generateValue(i, valI int) string {
	value := make([]byte, valI)
	binary.PutUvarint(value[:16], uint64(i))
	binary.PutUvarint(value[32:128], uint64(i))
	if _, err := rand.Read(value[128:]); err != nil {
		os.Exit(1)
	}
	return string(value)
}

func getprivkey(key string) crypto.PrivKey {
	cr, err := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	if err != nil {
		panic(err)
	}
	bkey, err := common.FromHex(key)
	if err != nil {
		panic(err)
	}
	priv, err := cr.PrivKeyFromBytes(bkey)
	if err != nil {
		panic(err)
	}
	return priv
}

func sendReplyList(q queue.Queue) {
	client := q.Client()
	client.Sub("mempool")
	var count int
	for msg := range client.Recv() {
		if msg.Ty == types.EventTxList {
			count++
			msg.Reply(client.NewMessage("consensus", types.EventReplyTxList,
				&types.ReplyTxList{getReplyList(txNumber)}))
			if count >= loopCount {
				time.Sleep(4 * time.Second)
				break
			}
		}
	}
}

func prepareTxList() *types.Transaction {
	var key string
	var value string
	var i int

	key = generateKey(i, 32)
	value = generateValue(i, 180)

	nput := &types.NormAction_Nput{&types.NormPut{Key: key, Value: []byte(value)}}
	action := &types.NormAction{Value: nput, Ty: types.NormActionPut}
	tx := &types.Transaction{Execer: []byte("norm"), Payload: types.Encode(action), Fee: 0}
	tx.Nonce = random.Int63()
	tx.Sign(types.SECP256K1, getprivkey("CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944"))
	return tx
}

func getReplyList(n int) (txs []*types.Transaction) {

	for i := 0; i < int(n); i++ {
		txs = append(txs, prepareTxList())
	}
	return txs
}
