package ticket

import (
	"flag"
	"math/rand"
	"runtime"
	"testing"
	"time"

	"fmt"

	"gitlab.33.cn/chain33/chain33/blockchain"
	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/common/config"
	"gitlab.33.cn/chain33/chain33/common/limits"
	"gitlab.33.cn/chain33/chain33/common/log"
	"gitlab.33.cn/chain33/chain33/executor"
	"gitlab.33.cn/chain33/chain33/mempool"
	"gitlab.33.cn/chain33/chain33/p2p"
	_ "gitlab.33.cn/chain33/chain33/plugin/dapp/init"
	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/ticket/types"
	_ "gitlab.33.cn/chain33/chain33/plugin/store/init"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/store"
	_ "gitlab.33.cn/chain33/chain33/system"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/wallet"
)

var (
	random *rand.Rand
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
func TestTicket(t *testing.T) {
	q, chain, mem, s, cs, w, qApi, p2p, exec := initEnvTicket()
	defer func() {
		chain.Close()
		mem.Close()
		p2p.Close()
		exec.Close()
		s.Close()
		cs.Close()
		qApi.Close()
		w.Close()
		q.Close()
	}()
	sleepWait(qApi, 0)

	// 自动设置票列表测试 coverage: 57.2%
	setTicketListRealize(qApi, cs)

	// 创建钱包测试 coverage: 62.5%
	// newWalletRealize(qApi, w, cs)

	sleepWait(qApi, 2)
}

func sleepWait(qApi client.QueueProtocolAPI, height int64) {
	for {
		header, err := qApi.GetLastHeader()
		if err != nil {
			panic(err)
		}
		// 区块 0 产生后开始下面的运作
		if header.Height >= height {
			break
		}
		time.Sleep(time.Second)
	}
}
func initEnvTicket() (queue.Queue, *blockchain.BlockChain, *mempool.Mempool, queue.Module, *Client, *wallet.Wallet, client.QueueProtocolAPI, queue.Module, *executor.Executor) {
	q := queue.New("channel")
	flag.Parse()
	cfg := config.InitCfg("../../../cmd/chain33/chain33.test.toml")
	cfg.TestNet = true
	cfg.Consensus.Name = "ticket"

	types.SetTestNet(cfg.TestNet)
	types.SetTitle(cfg.Title)

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

	w := wallet.New(cfg.Wallet)
	w.SetQueueClient(q.Client())

	network := p2p.New(cfg.P2P)
	network.SetQueueClient(q.Client())

	qApi, _ := client.New(q.Client(), nil)

	return q, chain, mem, s, cs.(*Client), w, qApi, network, exec
}

// 获取票的列表
func getTicketList(qApi client.QueueProtocolAPI) (types.Message, error) {
	reqaddr := &ty.TicketList{"12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv", 1}
	var req types.ChainExecutor
	req.Driver = "ticket"
	req.FuncName = "TicketList"
	req.Param = types.Encode(reqaddr)
	msg, err := qApi.QueryChain(&req)
	return msg, err
}

func setTicketListRealize(qApi client.QueueProtocolAPI, cs *Client) {
	msg, err := getTicketList(qApi)
	if err != nil {
		panic(err)
	}
	reply := msg.(*ty.ReplyTicketList)
	reply.Tickets = reply.Tickets[0:5000]
	runtime.GC()
	cs.setTicket(reply, getPrivMap(types.TestPrivkeyList))
}

func newWalletRealize(qApi client.QueueProtocolAPI, w *wallet.Wallet, cs *Client) {
	seed := &types.SaveSeedByPw{"subject hamster apple parent vital can adult chapter fork business humor pen tiger void elephant", "123456"}
	_, err := qApi.SaveSeed(seed)
	if err != nil {
		panic(err)
	}

	err = w.ProcWalletUnLock(&types.WalletUnLock{"123456", 0, false})
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

	// 调用刷新票
	cs.flushTicket()
}
