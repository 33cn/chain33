package testnode

import (
	"fmt"
	"math/rand"
	"time"

	"gitlab.33.cn/chain33/chain33/blockchain"
	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/common/config"
	"gitlab.33.cn/chain33/chain33/common/log"
	"gitlab.33.cn/chain33/chain33/consensus"
	"gitlab.33.cn/chain33/chain33/executor"
	"gitlab.33.cn/chain33/chain33/mempool"
	"gitlab.33.cn/chain33/chain33/p2p"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/store"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/wallet"
)

//这个包提供一个通用的测试节点，用于单元测试和集成测试。
func init() {
	log.SetLogLevel("info")
}

type Chain33Mock struct {
	random  *rand.Rand
	client  queue.Client
	api     client.QueueProtocolAPI
	chain   *blockchain.BlockChain
	mem     *mempool.Mempool
	cs      queue.Module
	exec    *executor.Executor
	wallet  queue.Module
	network queue.Module
	store   queue.Module
	rpc     *rpc.RPC
	cfg     *types.Config
}

func New(cfgpath string, mockapi client.QueueProtocolAPI) *Chain33Mock {
	q := queue.New("channel")
	cfg := config.InitCfg(cfgpath)
	types.SetTestNet(cfg.TestNet)
	types.SetTitle(cfg.Title)
	types.Debug = false
	mock := &Chain33Mock{cfg: cfg}
	mock.random = rand.New(rand.NewSource(types.Now().UnixNano()))
	mock.chain = blockchain.New(cfg.BlockChain)
	mock.chain.SetQueueClient(q.Client())

	mock.exec = executor.New(cfg.Exec)
	mock.exec.SetQueueClient(q.Client())
	types.SetMinFee(cfg.Exec.MinExecFee)

	mock.store = store.New(cfg.Store)
	mock.store.SetQueueClient(q.Client())

	mock.cs = consensus.New(cfg.Consensus)
	mock.cs.SetQueueClient(q.Client())

	mock.mem = mempool.New(cfg.MemPool)
	mock.mem.SetQueueClient(q.Client())

	if cfg.P2P.Enable {
		mock.network = p2p.New(cfg.P2P)
		mock.network.SetQueueClient(q.Client())
	} else {
		mock.network = &mockP2P{}
		mock.network.SetQueueClient(q.Client())
	}

	cli := q.Client()
	w := wallet.New(cfg.Wallet)
	mock.client = cli
	mock.wallet = w
	mock.wallet.SetQueueClient(cli)

	if mockapi == nil {
		mockapi, _ = client.New(q.Client(), nil)
		newWalletRealize(mockapi)
	}
	mock.api = mockapi

	server := rpc.New(cfg.Rpc)
	server.SetAPI(mock.api)
	server.SetQueueClientNoListen(q.Client())
	mock.rpc = server
	return mock
}

func newWalletRealize(qApi client.QueueProtocolAPI) {
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
		_, err = qApi.WalletImportprivkey(privkey)
		if err != nil {
			panic(err)
		}
	}
}

func (mock *Chain33Mock) GetAPI() client.QueueProtocolAPI {
	return mock.api
}

func (mock *Chain33Mock) GetRPC() *rpc.RPC {
	return mock.rpc
}

func (mock *Chain33Mock) GetCfg() *types.Config {
	return mock.cfg
}

func (mock *Chain33Mock) Close() {
	mock.chain.Close()
	mock.store.Close()
	mock.mem.Close()
	mock.cs.Close()
	mock.exec.Close()
	mock.wallet.Close()
	mock.network.Close()
	mock.client.Close()
	mock.rpc.Close()
}

func (mock *Chain33Mock) WaitHeight(height int64) error {
	for {
		header, err := mock.api.GetLastHeader()
		if err != nil {
			return err
		}
		if header.Height >= height {
			break
		}
		time.Sleep(time.Second / 10)
	}
	return nil
}

type mockP2P struct {
}

func (m *mockP2P) SetQueueClient(client queue.Client) {
	go func() {
		p2pKey := "p2p"
		client.Sub(p2pKey)
		for msg := range client.Recv() {
			switch msg.Ty {
			case types.EventPeerInfo:
				msg.Reply(client.NewMessage(p2pKey, types.EventPeerList, &types.PeerList{}))
			case types.EventGetNetInfo:
				msg.Reply(client.NewMessage(p2pKey, types.EventPeerList, &types.NodeNetInfo{}))
			default:
				msg.ReplyErr("Do not support", types.ErrNotSupport)
			}
		}
	}()
}

func (m *mockP2P) Close() {
}
