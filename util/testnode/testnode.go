package testnode

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"gitlab.33.cn/chain33/chain33/blockchain"
	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/common/config"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/limits"
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
	err := limits.SetLimits()
	if err != nil {
		panic(err)
	}
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

func GetDefaultConfig() (*types.Config, *types.ConfigSubModule) {
	return config.InitCfgString(cfgstring)
}

func NewWithConfig(cfg *types.Config, sub *types.ConfigSubModule, mockapi client.QueueProtocolAPI) *Chain33Mock {
	q := queue.New("channel")
	types.SetTestNet(cfg.TestNet)
	types.SetTitle(cfg.Title)
	types.Debug = false
	mock := &Chain33Mock{cfg: cfg}
	mock.random = rand.New(rand.NewSource(types.Now().UnixNano()))
	mock.chain = blockchain.New(cfg.BlockChain)
	mock.chain.SetQueueClient(q.Client())

	mock.exec = executor.New(cfg.Exec, sub.Exec)
	mock.exec.SetQueueClient(q.Client())
	types.SetMinFee(cfg.Exec.MinExecFee)

	mock.store = store.New(cfg.Store, sub.Store)
	mock.store.SetQueueClient(q.Client())

	mock.cs = consensus.New(cfg.Consensus, sub.Consensus)
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
	w := wallet.New(cfg.Wallet, sub.Wallet)
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

func New(cfgpath string, mockapi client.QueueProtocolAPI) *Chain33Mock {
	var cfg *types.Config
	var sub *types.ConfigSubModule
	if cfgpath == "" {
		cfg, sub = config.InitCfgString(cfgstring)
	} else {
		cfg, sub = config.InitCfg(cfgpath)
	}
	return NewWithConfig(cfg, sub, mockapi)
}

func (m *Chain33Mock) Listen() {
	portgrpc, portjsonrpc := m.rpc.Listen()
	if strings.HasSuffix(m.cfg.Rpc.JrpcBindAddr, ":0") {
		l := len(m.cfg.Rpc.JrpcBindAddr)
		m.cfg.Rpc.JrpcBindAddr = m.cfg.Rpc.JrpcBindAddr[0:l-2] + ":" + fmt.Sprint(portjsonrpc)
	}
	if strings.HasSuffix(m.cfg.Rpc.GrpcBindAddr, ":0") {
		l := len(m.cfg.Rpc.GrpcBindAddr)
		m.cfg.Rpc.GrpcBindAddr = m.cfg.Rpc.GrpcBindAddr[0:l-2] + ":" + fmt.Sprint(portgrpc)
	}
}

func newWalletRealize(qApi client.QueueProtocolAPI) {
	seed := &types.SaveSeedByPw{"subject hamster apple parent vital can adult chapter fork business humor pen tiger void elephant", "123456"}
	reply, err := qApi.SaveSeed(seed)
	if !reply.IsOk && err != nil {
		panic(err)
	}
	reply, err = qApi.WalletUnLock(&types.WalletUnLock{"123456", 0, false})
	if !reply.IsOk && err != nil {
		panic(err)
	}
	for i, priv := range TestPrivkeyHex {
		privkey := &types.ReqWalletImportPrivkey{priv, fmt.Sprintf("label%d", i)}
		_, err = qApi.WalletImportprivkey(privkey)
		if err != nil {
			panic(err)
		}
	}
	req := &types.ReqAccountList{WithoutBalance: true}
	_, err = qApi.WalletGetAccountList(req)
	if err != nil {
		panic(err)
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

func (m *Chain33Mock) GenNoneTxs(n int64) (txs []*types.Transaction) {
	_, priv := m.Genaddress()
	to, _ := m.Genaddress()
	for i := 0; i < int(n); i++ {
		txs = append(txs, m.CreateNoneTx(priv, to, types.Coin*(n+1)))
	}
	return txs
}

func (m *Chain33Mock) Genaddress() (string, crypto.PrivKey) {
	cr, err := crypto.New(types.GetSignName("", types.SECP256K1))
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

func (m *Chain33Mock) CreateNoneTx(priv crypto.PrivKey, to string, amount int64) *types.Transaction {
	tx := &types.Transaction{Execer: []byte("none"), Payload: []byte("none"), Fee: 1e6, To: to}
	tx.Nonce = m.random.Int63()
	tx.To = address.ExecAddress("none")
	tx.Sign(types.SECP256K1, priv)
	return tx
}
