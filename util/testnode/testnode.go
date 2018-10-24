package testnode

import (
	"bytes"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/blockchain"
	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/common/address"
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
	"gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/store"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/util"
	"gitlab.33.cn/chain33/chain33/wallet"

	cty "gitlab.33.cn/chain33/chain33/system/dapp/coins/types"
)

var chainlog = log15.New("module", "testnode")

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

func (m *Chain33Mock) CreateCoinsTx(priv crypto.PrivKey, to string, amount int64) *types.Transaction {
	exec := types.LoadExecutorType("coins")
	if exec == nil {
		panic("unknow driver coins")
	}
	tx, err := exec.AssertCreate(&types.CreateTx{
		To:     to,
		Amount: amount,
	})
	if err != nil {
		panic(err)
	}
	tx.Nonce = m.random.Int63()
	tx.To = to
	tx.Sign(types.SECP256K1, priv)
	return tx
}

var zeroHash [32]byte

func (m *Chain33Mock) CreateNoneBlock(n int64) *types.Block {
	newblock := &types.Block{}
	newblock.Height = -1
	newblock.BlockTime = types.Now().Unix()
	newblock.ParentHash = zeroHash[:]
	newblock.Txs = m.GenNoneTxs(n)
	newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
	return newblock
}

func (m *Chain33Mock) CreateGenesisBlock() *types.Block {
	var tx types.Transaction
	tx.Execer = []byte("coins")
	tx.To = m.cfg.Consensus.Genesis
	//gen payload
	g := &cty.CoinsAction_Genesis{}
	g.Genesis = &types.AssetsGenesis{}
	g.Genesis.Amount = 1e8 * types.Coin
	tx.Payload = types.Encode(&cty.CoinsAction{Value: g, Ty: cty.CoinsActionGenesis})
	newblock := &types.Block{}
	newblock.Height = 0
	newblock.BlockTime = types.Now().Unix()
	newblock.ParentHash = zeroHash[:]
	newblock.Txs = append(newblock.Txs, &tx)
	newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
	return newblock
}

func (m *Chain33Mock) ExecBlock(prevStateRoot []byte, block *types.Block, errReturn bool, sync bool) (*types.BlockDetail, []*types.Transaction, error) {
	//发送执行交易给execs模块
	//通过consensus module 再次检查
	client := m.client
	chainlog.Debug("ExecBlock", "height------->", block.Height, "ntx", len(block.Txs))
	beg := types.Now()
	defer func() {
		chainlog.Info("ExecBlock", "height", block.Height, "ntx", len(block.Txs), "writebatchsync", sync, "cost", types.Since(beg))
	}()

	if errReturn && block.Height > 0 && !block.CheckSign() {
		//block的来源不是自己的mempool，而是别人的区块
		return nil, nil, types.ErrSign
	}
	//tx交易去重处理, 这个地方要查询数据库，需要一个更快的办法
	cacheTxs := types.TxsToCache(block.Txs)
	oldtxscount := len(cacheTxs)
	var err error
	cacheTxs, err = util.CheckTxDup(client, cacheTxs, block.Height)
	if err != nil {
		return nil, nil, err
	}
	newtxscount := len(cacheTxs)
	if oldtxscount != newtxscount && errReturn {
		return nil, nil, types.ErrTxDup
	}
	chainlog.Debug("ExecBlock", "prevtx", oldtxscount, "newtx", newtxscount)
	block.TxHash = merkle.CalcMerkleRootCache(cacheTxs)
	block.Txs = types.CacheToTxs(cacheTxs)

	receipts := util.ExecTx(client, prevStateRoot, block)
	var maplist = make(map[string]*types.KeyValue)
	var kvset []*types.KeyValue
	var deltxlist = make(map[int]bool)
	var rdata []*types.ReceiptData //save to db receipt log
	for i := 0; i < len(receipts.Receipts); i++ {
		receipt := receipts.Receipts[i]
		if receipt.Ty == types.ExecErr {
			chainlog.Error("exec tx err", "err", receipt)
			if errReturn { //认为这个是一个错误的区块
				return nil, nil, types.ErrBlockExec
			}
			deltxlist[i] = true
			continue
		}

		rdata = append(rdata, &types.ReceiptData{receipt.Ty, receipt.Logs})
		//处理KV
		kvs := receipt.KV
		for _, kv := range kvs {
			if item, ok := maplist[string(kv.Key)]; ok {
				item.Value = kv.Value //更新item 的value
			} else {
				maplist[string(kv.Key)] = kv
				kvset = append(kvset, kv)
			}
		}
	}
	//check TxHash
	calcHash := merkle.CalcMerkleRoot(block.Txs)
	if errReturn && !bytes.Equal(calcHash, block.TxHash) {
		return nil, nil, types.ErrCheckTxHash
	}
	block.TxHash = calcHash
	//删除无效的交易
	var deltx []*types.Transaction
	if len(deltxlist) > 0 {
		var newtx []*types.Transaction
		for i := 0; i < len(block.Txs); i++ {
			if deltxlist[i] {
				deltx = append(deltx, block.Txs[i])
			} else {
				newtx = append(newtx, block.Txs[i])
			}
		}
		block.Txs = newtx
		block.TxHash = merkle.CalcMerkleRoot(block.Txs)
	}

	var detail types.BlockDetail

	calcHash = util.ExecKVMemSet(client, prevStateRoot, block.Height, kvset, sync)

	if errReturn && !bytes.Equal(block.StateHash, calcHash) {
		util.ExecKVSetRollback(client, calcHash)
		if len(rdata) > 0 {
			for i, rd := range rdata {
				rd.OutputReceiptDetails(block.Txs[i].Execer, chainlog)
			}
		}
		return nil, nil, types.ErrCheckStateHash
	}
	block.StateHash = calcHash
	detail.Block = block
	detail.Receipts = rdata
	if detail.Block.Height > 0 {
		err := util.CheckBlock(client, &detail)
		if err != nil {
			chainlog.Debug("CheckBlock-->", "err=", err)
			return nil, deltx, err
		}
	}
	//save to db
	util.ExecKVSetCommit(client, block.StateHash)
	detail.KV = kvset
	detail.PrevStatusHash = prevStateRoot
	//get receipts
	//save kvset and get state hash
	//ulog.Debug("blockdetail-->", "detail=", detail)
	return &detail, deltx, nil
}
