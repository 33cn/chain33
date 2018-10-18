package wallet_test

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gitlab.33.cn/chain33/chain33/blockchain"
	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/common/config"
	"gitlab.33.cn/chain33/chain33/common/log"
	"gitlab.33.cn/chain33/chain33/consensus"
	"gitlab.33.cn/chain33/chain33/executor"
	"gitlab.33.cn/chain33/chain33/mempool"
	"gitlab.33.cn/chain33/chain33/p2p"
	tickettypes "gitlab.33.cn/chain33/chain33/plugin/dapp/ticket/types"
	ticketwallet "gitlab.33.cn/chain33/chain33/plugin/dapp/ticket/wallet"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/store"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/wallet"

	_ "gitlab.33.cn/chain33/chain33/plugin"
	_ "gitlab.33.cn/chain33/chain33/system"
)

func init() {
	log.SetLogLevel("info")
}

type chain33Mock struct {
	random *rand.Rand
	client queue.Client

	api     client.QueueProtocolAPI
	chain   *blockchain.BlockChain
	mem     *mempool.Mempool
	cs      queue.Module
	exec    *executor.Executor
	wallet  queue.Module
	network *p2p.P2p
	store   queue.Module
}

func (mock *chain33Mock) start() {
	q := queue.New("channel")
	cfg := config.InitCfg("testdata/chain33.test.toml")
	cfg.Title = "local"
	cfg.TestNet = true
	cfg.Consensus.Name = "ticket"
	types.SetTestNet(cfg.TestNet)
	types.SetTitle(cfg.Title)
	types.Debug = false
	mock.random = rand.New(rand.NewSource(types.Now().UnixNano()))

	mock.chain = blockchain.New(cfg.BlockChain)
	mock.chain.SetQueueClient(q.Client())
	mock.exec = executor.New(cfg.Exec)
	mock.exec.SetQueueClient(q.Client())
	types.SetMinFee(0)
	mock.store = store.New(cfg.Store)
	mock.store.SetQueueClient(q.Client())
	mock.cs = consensus.New(cfg.Consensus)
	mock.cs.SetQueueClient(q.Client())
	mock.mem = mempool.New(cfg.MemPool)
	mock.mem.SetQueueClient(q.Client())
	mock.network = p2p.New(cfg.P2P)
	mock.network.SetQueueClient(q.Client())
	mock.api, _ = client.New(q.Client(), nil)
	cli := q.Client()
	w := wallet.New(cfg.Wallet)
	mock.client = cli
	mock.wallet = w
	mock.wallet.SetQueueClient(cli)
	api, _ := client.New(q.Client(), nil)
	newWalletRealize(api, w)
}

func newWalletRealize(qApi client.QueueProtocolAPI, w *wallet.Wallet) {
	seed := &types.SaveSeedByPw{"subject hamster apple parent vital can adult chapter fork business humor pen tiger void elephant", "123456"}
	_, err := qApi.SaveSeed(seed)
	if err != nil {
		panic(err)
	}
	_, err = qApi.WalletUnLock(&types.WalletUnLock{"123456", 0, false})
	if err != nil {
		panic(err)
	}
	for i, priv := range types.TestPrivkeyHex {
		privkey := &types.ReqWalletImportPrivkey{priv, fmt.Sprintf("label%d", i)}
		_, err = w.ProcImportPrivKey(privkey)
		if err != nil {
			panic(err)
		}
	}
}

func (mock *chain33Mock) stop() {
	mock.chain.Close()
	mock.store.Close()
	mock.mem.Close()
	mock.cs.Close()
	mock.exec.Close()
	mock.wallet.Close()
	mock.network.Close()
	mock.client.Close()
}

func (mock *chain33Mock) waitHeight(height int64, t *testing.T) {
	for {
		header, err := mock.api.GetLastHeader()
		require.Equal(t, err, nil)
		// 区块出块以后再执行下面的动作
		if header.Height >= height {
			break
		}
		time.Sleep(time.Second)
	}
}

func (mock *chain33Mock) getTicketList(addr string, t *testing.T) *tickettypes.ReplyTicketList {
	msg, err := mock.api.Query(types.TicketX, "TicketList", &tickettypes.TicketList{addr, 1})
	require.Equal(t, err, nil)
	return msg.(*tickettypes.ReplyTicketList)
}

func Test_WalletTicket(t *testing.T) {
	t.Log("Begin wallet ticket test")
	// 启动虚拟即
	mock := &chain33Mock{}
	mock.start()
	defer mock.stop()

	mock.waitHeight(0, t)
	ticketList := mock.getTicketList("12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv", t)
	if ticketList == nil {
		t.Error("getTicketList return nil")
		return
	}
	ticketwallet.FlushTicket(mock.api)
	mock.waitHeight(2, t)
	header, err := mock.api.GetLastHeader()
	require.Equal(t, err, nil)
	require.Equal(t, header.Height >= 2, true)
	t.Log("End wallet ticket test")
}
