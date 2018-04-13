package pbft

import (
	"gitlab.33.cn/chain33/chain33/blockchain"
	"gitlab.33.cn/chain33/chain33/queue"
	"testing"
	//"gitlab.33.cn/chain33/chain33/store"
	//"gitlab.33.cn/chain33/chain33/p2p"
	"flag"
	"gitlab.33.cn/chain33/chain33/common/config"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/log"
	"gitlab.33.cn/chain33/chain33/mempool"
	"gitlab.33.cn/chain33/chain33/types"
	"math/rand"
	//"time"
	//"gitlab.33.cn/chain33/chain33/execs"
	//"os/exec"
	//"github.com/piotrnar/gocoin/lib/chain"
	"gitlab.33.cn/chain33/chain33/store"
	//"gitlab.33.cn/chain33/chain33/execs"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/limits"
	"gitlab.33.cn/chain33/chain33/executor"
	"gitlab.33.cn/chain33/chain33/p2p"
	"gitlab.33.cn/chain33/chain33/wallet"
	"strconv"
	"time"
)

var (
	random       *rand.Rand
	transactions []*types.Transaction
	txSize       = 1000
)

func init() {
	err := limits.SetLimits()
	if err != nil {
		panic(err)
	}
	random = rand.New(rand.NewSource(time.Now().UnixNano()))
	//common.SetLogLevel("info")
}
func TestPbft(t *testing.T) {
	q, chain, _, s, mem, exec, cs, wallet := initEnvPbft()
	defer chain.Close()
	defer mem.Close()
	//defer p2pnet.Close()
	defer exec.Close()
	defer s.Close()
	defer cs.Close()
	defer q.Close()
	defer wallet.Close()
	time.Sleep(5 * time.Second)

	sendReplyList(q)

}

func initEnvPbft() (queue.Queue, *blockchain.BlockChain, *p2p.P2p, queue.Module, *mempool.Mempool, queue.Module, *PbftClient, queue.Module) {
	var q = queue.New("channel")
	flag.Parse()
	cfg := config.InitCfg("chain33.test.toml")
	log.SetFileLog(cfg.Log)
	chain := blockchain.New(cfg.BlockChain)
	chain.SetQueueClient(q.Client())
	mem := mempool.New(cfg.MemPool)
	mem.SetQueueClient(q.Client())
	exec := executor.New(cfg.Exec)
	exec.SetQueueClient(q.Client())
	types.SetMinFee(0)
	s := store.New(cfg.Store)
	s.SetQueueClient(q.Client())
	cs := NewPbft(cfg.Consensus)
	cs.SetQueueClient(q.Client())
	p2pnet := p2p.New(cfg.P2P)
	p2pnet.SetQueueClient(q.Client())
	walletm := wallet.New(cfg.Wallet)
	walletm.SetQueueClient(q.Client())

	return q, chain, p2pnet, s, mem, exec, cs, walletm

}

func sendReplyList(q queue.Queue) {
	client := q.Client()
	client.Sub("mempool")
	var count int
	for msg := range client.Recv() {
		if msg.Ty == types.EventTxList {
			count++
			createReplyList("test" + strconv.Itoa(count))
			msg.Reply(client.NewMessage("consensus", types.EventReplyTxList,
				&types.ReplyTxList{transactions}))
			if count == 5 {
				time.Sleep(5 * time.Second)
				break
			}
		}
	}
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

func createReplyList(account string) {
	var result []*types.Transaction
	for j := 0; j < txSize; j++ {
		//tx := &types.Transaction{}
		val := &types.CoinsAction_Transfer{&types.CoinsTransfer{Amount: 10}}
		action := &types.CoinsAction{Value: val, Ty: types.CoinsActionTransfer}
		tx := &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(action), Fee: 0}
		tx.To = "14qViLJfdGaP4EeHnDyJbEGQysnCpwn1gZ"

		tx.Nonce = random.Int63()

		tx.Sign(types.SECP256K1, getprivkey("CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944"))
		result = append(result, tx)
	}
	//result = append(result, tx)
	transactions = result
}
