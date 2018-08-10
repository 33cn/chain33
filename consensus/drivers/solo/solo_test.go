package solo

import (
	"flag"
	"math/rand"
	"testing"
	"time"

	"gitlab.33.cn/chain33/chain33/blockchain"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/common/config"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/limits"
	"gitlab.33.cn/chain33/chain33/common/log"
	"gitlab.33.cn/chain33/chain33/executor"
	"gitlab.33.cn/chain33/chain33/mempool"
	"gitlab.33.cn/chain33/chain33/p2p"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/store"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	endLoop        = 5
	txNumber int64 = 100
	random   *rand.Rand
)

func init() {
	err := limits.SetLimits()
	if err != nil {
		panic(err)
	}
	log.SetLogLevel("info")
	random = rand.New(rand.NewSource(types.Now().UnixNano()))
}

// 执行： go test -cover
func TestSolo(t *testing.T) {
	q, chain, mem, s, p2p := initEnvSolo()

	defer chain.Close()
	defer s.Close()
	defer mem.Close()
	defer q.Close()
	defer p2p.Close()
	sendReplyList(q)
}

func initEnvSolo() (queue.Queue, *blockchain.BlockChain, *mempool.Mempool, queue.Module, queue.Module) {
	var q = queue.New("channel")
	flag.Parse()
	cfg := config.InitCfg("../../../cmd/chain33/chain33.test.toml")
	chain := blockchain.New(cfg.BlockChain)
	chain.SetQueueClient(q.Client())

	exec := executor.New(cfg.Exec)
	exec.SetQueueClient(q.Client())
	types.SetMinFee(0)

	s := store.New(cfg.Store)
	s.SetQueueClient(q.Client())

	cs := New(cfg.Consensus)
	cs.SetQueueClient(q.Client())

	mem := mempool.New(cfg.MemPool)
	mem.SetQueueClient(q.Client())

	network := p2p.New(cfg.P2P)
	network.SetQueueClient(q.Client())

	return q, chain, mem, s, network
}

// 向共识发送交易列表
func sendReplyList(q queue.Queue) {
	client := q.Client()
	client.Sub("mempool")
	var count int
	for msg := range client.Recv() {
		if msg.Ty == types.EventTxList {
			msg.Reply(client.NewMessage("consensus", types.EventReplyTxList,
				&types.ReplyTxList{genTxs(txNumber)}))
			count++
			if count == endLoop {
				break
			}
		}
	}

	time.Sleep(4 * time.Second)
}

func genTxs(n int64) (txs []*types.Transaction) {
	_, priv := genaddress()
	to, _ := genaddress()
	for i := 0; i < int(n); i++ {
		txs = append(txs, createTx(priv, to, types.Coin*(n+1)))
	}
	return txs
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
	addrto := address.PubKeyToAddress(privto.PubKey().Bytes())
	return addrto.String(), privto
}

func createTx(priv crypto.PrivKey, to string, amount int64) *types.Transaction {
	v := &types.CoinsAction_Transfer{&types.CoinsTransfer{Amount: amount}}
	transfer := &types.CoinsAction{Value: v, Ty: types.CoinsActionTransfer}
	tx := &types.Transaction{Execer: []byte("none"), Payload: types.Encode(transfer), Fee: 1e6, To: to}
	tx.Nonce = random.Int63()
	tx.To = address.ExecAddress("none")
	tx.Sign(types.SECP256K1, priv)
	return tx
}
