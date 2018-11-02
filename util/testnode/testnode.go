package testnode

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/blockchain"
	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/config"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/limits"
	"gitlab.33.cn/chain33/chain33/common/log"
	"gitlab.33.cn/chain33/chain33/consensus"
	"gitlab.33.cn/chain33/chain33/executor"
	"gitlab.33.cn/chain33/chain33/mempool"
	"gitlab.33.cn/chain33/chain33/p2p"
	"gitlab.33.cn/chain33/chain33/pluginmgr"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/rpc/jsonclient"
	rpctypes "gitlab.33.cn/chain33/chain33/rpc/types"
	"gitlab.33.cn/chain33/chain33/store"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/util"
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

//保证只有一个chain33 会运行
var lognode = log15.New("module", "lognode")
var chain33globalLock sync.Mutex

type Chain33Mock struct {
	random  *rand.Rand
	q       queue.Queue
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
	return newWithConfig(cfg, sub, mockapi)
}

func newWithConfig(cfg *types.Config, sub *types.ConfigSubModule, mockapi client.QueueProtocolAPI) *Chain33Mock {
	chain33globalLock.Lock()
	types.Init(cfg.Title, cfg)
	q := queue.New("channel")
	types.Debug = false
	mock := &Chain33Mock{cfg: cfg, q: q}
	mock.random = rand.New(rand.NewSource(types.Now().UnixNano()))

	mock.exec = executor.New(cfg.Exec, sub.Exec)
	mock.exec.SetQueueClient(q.Client())
	types.SetMinFee(cfg.Exec.MinExecFee)
	lognode.Info("init exec")

	mock.store = store.New(cfg.Store, sub.Store)
	mock.store.SetQueueClient(q.Client())
	lognode.Info("init store")

	mock.chain = blockchain.New(cfg.BlockChain)
	mock.chain.SetQueueClient(q.Client())
	lognode.Info("init blockchain")

	mock.cs = consensus.New(cfg.Consensus, sub.Consensus)
	mock.cs.SetQueueClient(q.Client())
	lognode.Info("init consensus " + cfg.Consensus.Name)

	mock.mem = mempool.New(cfg.MemPool)
	mock.mem.SetQueueClient(q.Client())
	lognode.Info("init mempool")

	if cfg.P2P.Enable {
		mock.network = p2p.New(cfg.P2P)
		mock.network.SetQueueClient(q.Client())
	} else {
		mock.network = &mockP2P{}
		mock.network.SetQueueClient(q.Client())
	}
	lognode.Info("init P2P")
	cli := q.Client()
	w := wallet.New(cfg.Wallet, sub.Wallet)
	mock.client = cli
	mock.wallet = w
	mock.wallet.SetQueueClient(cli)
	lognode.Info("init wallet")
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
	if cfgpath == "" || cfgpath == "--notset--" || cfgpath == "--free--" {
		cfg, sub = config.InitCfgString(cfgstring)
		if cfgpath == "--free--" {
			setFee(cfg, 0)
		}
	} else {
		cfg, sub = config.InitCfg(cfgpath)
	}
	return newWithConfig(cfg, sub, mockapi)
}

func (m *Chain33Mock) Listen() {
	pluginmgr.AddRPC(m.rpc)
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

func (m *Chain33Mock) GetBlockChain() *blockchain.BlockChain {
	return m.chain
}

func setFee(cfg *types.Config, fee int64) {
	cfg.Exec.MinExecFee = fee
	cfg.MemPool.MinTxFee = fee
	cfg.Wallet.MinFee = fee
	if fee == 0 {
		cfg.Exec.IsFree = true
	}
}

func (m *Chain33Mock) GetJsonC() *jsonclient.JSONClient {
	jsonc, _ := jsonclient.NewJSONClient("http://" + m.cfg.Rpc.JrpcBindAddr + "/")
	return jsonc
}

func (m *Chain33Mock) SendAndSign(priv crypto.PrivKey, hextx string) ([]byte, error) {
	txbytes, err := hex.DecodeString(hextx)
	if err != nil {
		return nil, err
	}
	tx := &types.Transaction{}
	err = types.Decode(txbytes, tx)
	if err != nil {
		return nil, err
	}
	tx.Fee = 1e6
	tx.Sign(types.SECP256K1, priv)
	reply, err := m.api.SendTx(tx)
	if err != nil {
		return nil, err
	}
	return reply.GetMsg(), nil
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
	for i, priv := range util.TestPrivkeyHex {
		privkey := &types.ReqWalletImportPrivkey{priv, fmt.Sprintf("label%d", i)}
		acc, err := qApi.WalletImportprivkey(privkey)
		if err != nil {
			panic(err)
		}
		lognode.Info("import", "index", i, "addr", acc.Acc.Addr)
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
	chain33globalLock.Unlock()
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

func (mock *Chain33Mock) WaitTx(hash []byte) (*rpctypes.TransactionDetail, error) {
	if hash == nil {
		return nil, nil
	}
	for {
		param := &types.ReqHash{Hash: hash}
		_, err := mock.api.QueryTx(param)
		if err != nil {
			println(err)
			time.Sleep(time.Second / 10)
			continue
		}
		var testResult rpctypes.TransactionDetail
		data := rpctypes.QueryParm{
			Hash: common.ToHex(hash),
		}
		err = mock.GetJsonC().Call("Chain33.QueryTransaction", data, &testResult)
		return &testResult, err
	}
}

func (mock *Chain33Mock) SendHot() error {
	header, err := mock.api.GetLastHeader()
	if err != nil {
		return err
	}
	tx := util.CreateCoinsTx(mock.GetGenesisKey(), mock.GetHotAddress(), 10000*types.Coin)
	_, err = mock.GetAPI().SendTx(tx)
	if err != nil {
		panic(err)
	}
	return mock.WaitHeight(header.Height + 1)
}

func (mock *Chain33Mock) GetAccount(stateHash []byte, addr string) *types.Account {
	statedb := executor.NewStateDB(mock.client, stateHash, nil, nil)
	acc := account.NewCoinsAccount()
	acc.SetDB(statedb)
	return acc.LoadAccount(addr)
}

func (mock *Chain33Mock) GetBlock(height int64) *types.Block {
	blocks, err := mock.api.GetBlocks(&types.ReqBlocks{Start: height, End: height})
	if err != nil {
		panic(err)
	}
	return blocks.Items[0].Block
}

func (mock *Chain33Mock) GetLastBlock() *types.Block {
	header, err := mock.api.GetLastHeader()
	if err != nil {
		panic(err)
	}
	return mock.GetBlock(header.Height)
}

func (m *Chain33Mock) GetClient() queue.Client {
	return m.client
}

func (m *Chain33Mock) GetHotKey() crypto.PrivKey {
	return util.TestPrivkeyList[0]
}

func (m *Chain33Mock) GetHotAddress() string {
	return m.cfg.Consensus.HotkeyAddr
}

func (m *Chain33Mock) GetGenesisKey() crypto.PrivKey {
	return util.TestPrivkeyList[1]
}

func (m *Chain33Mock) GetGenesisAddress() string {
	return m.cfg.Consensus.Genesis
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
			case types.EventTxBroadcast, types.EventBlockBroadcast:
			default:
				msg.ReplyErr("p2p->Do not support "+types.GetEventName(int(msg.Ty)), types.ErrNotSupport)
			}
		}
	}()
}

func (m *mockP2P) Close() {
}
