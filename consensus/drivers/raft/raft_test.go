package raft

import (
	"flag"
	"strconv"
	"strings"
	"testing"
	"time"

	"gitlab.33.cn/chain33/chain33/blockchain"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/config"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/store"
	"gitlab.33.cn/chain33/chain33/types"
	"github.com/coreos/etcd/raft/raftpb"
	log "github.com/inconshreveable/log15"
)

var (
	transactions []*types.Transaction
	txSize       int = 10000
	endLoop      int = 10
	configpath       = flag.String("f", "chain33.toml", "configfile")
)
var amount = int64(1e8)
var v = &types.CoinsAction_Transfer{&types.CoinsTransfer{Amount: amount}}
var transfer = &types.CoinsAction{Value: v, Ty: types.CoinsActionTransfer}
var tx = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 1000000, Expire: 0}

var c, _ = crypto.New(types.GetSignatureTypeName(types.SECP256K1))
var hex = "CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944"
var data, _ = common.FromHex(hex)
var privKey, _ = c.PrivKeyFromBytes(data)

// 执行： go test -cover
// 多次执行，需要删除之前的数据库文件，不然交易都是重复的
func TestRaft(t *testing.T) {
	flag.PrintDefaults()
	flag.Parse()
	cfg := config.InitCfg(*configpath)

	q := queue.New("channel")

	chain := blockchain.New(cfg.BlockChain)
	chain.SetQueueClient(q.Client())

	log.Info("loading store module")
	s := store.New(cfg.Store)
	s.SetQueueClient(q.Client())

	urls := "http://127.0.0.1:9021"
	nodeId := 1
	//isValidator = true

	// propose channel
	proposeC := make(chan *types.Block)
	confChangeC := make(chan raftpb.ConfChange)
	//var cfg *types.Consensus
	var b *RaftClient
	getSnapshot := func() ([]byte, error) { return b.getSnapshot() }
	var peers []string
	commitC, errorC, snapshotterReady, validatorC := NewRaftNode(nodeId, false, strings.Split(urls, ","), peers, peers, getSnapshot, proposeC, confChangeC)

	b = NewBlockstore(cfg.Consensus, <-snapshotterReady, proposeC, commitC, errorC, validatorC)

	time.Sleep(5 * time.Second)

	b.SetQueueClient(q.Client())

	go sendReplyList(q.Client())

	log.Info("start")
	q.Start()
}

// 向共识发送交易列表
func sendReplyList(client queue.Client) {
	client.Sub("mempool")
	var accountNum int
	for msg := range client.Recv() {
		if msg.Ty == types.EventTxList {
			accountNum++
			if accountNum < endLoop/5 {
				// 无交易
				msg.Reply(client.NewMessage("consensus", types.EventReplyTxList,
					&types.ReplyTxList{}))
			} else {
				// 有交易
				createReplyList("test" + strconv.Itoa(accountNum))
				msg.Reply(client.NewMessage("consensus", types.EventReplyTxList,
					&types.ReplyTxList{transactions}))
			}
			if accountNum == endLoop+1 {
				log.Info("Test finished!!")
				client.Close()
				break
			}
		}
	}
}

// 准备交易列表
func createReplyList(account string) {
	var result []*types.Transaction
	for j := 0; j < txSize; j++ {
		tx := &types.Transaction{}
		if j > 1000 && j%1000 == 0 {
			// 重复交易
			tx = &types.Transaction{Execer: []byte("coin"), Payload: []byte("duplicate"), Fee: 1000, Expire: 0}
		} else {
			tx = &types.Transaction{Execer: []byte(account + "This is a payload" + strconv.Itoa(j)), Payload: []byte(account + "This is a account" + strconv.Itoa(j)), Fee: 1000, Expire: 0}
		}

		result = append(result, tx)
	}
	//result = append(result, tx)
	transactions = result
}
