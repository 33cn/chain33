package client_test

import (
	"flag"

	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/common/config"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/rpc"
)

var (
	configPath = flag.String("f", "chain33.toml", "configfile")
)

type MockLive interface {
	OnStartup(m *mockSystem)
	OnStop()
}

type mockSystem struct {
	MockLive
	q         queue.Queue
	chain     *mockBlockChain
	mem       *mockMempool
	wallet    *mockWallet
	p2p       *mockP2P
	consensus *mockConsensus
	store     *mockStore
}

func (mock *mockSystem) startup(size int) client.QueueProtocolAPI {
	cfg := config.InitCfg(*configPath)
	rpc.Init(cfg.Rpc)

	var q = queue.New("channel")
	chain := &mockBlockChain{}
	chain.SetQueueClient(q)
	mem := &mockMempool{}
	mem.SetQueueClient(q)
	wallet := &mockWallet{}
	wallet.SetQueueClient(q)
	p2p := &mockP2P{}
	p2p.SetQueueClient(q)
	consensus := &mockConsensus{}
	consensus.SetQueueClient(q)
	store := &mockStore{}
	store.SetQueueClient(q)

	mock.q = q
	mock.chain = chain
	mock.mem = mem
	mock.wallet = wallet
	mock.p2p = p2p
	mock.consensus = consensus
	mock.store = store
	if mock.MockLive != nil {
		mock.MockLive.OnStartup(mock)
	}
	return mock.getAPI()
}

func (mock *mockSystem) stop() {
	if mock.MockLive != nil {
		mock.MockLive.OnStop()
	}
	mock.chain.Close()
	mock.mem.Close()
	mock.wallet.Close()
	mock.p2p.Close()
	mock.consensus.Close()
	mock.store.Close()
	mock.q.Close()
}

func (mock *mockSystem) getAPI() client.QueueProtocolAPI {
	api, _ := client.New(mock.q.Client(), nil)
	return api
}

type mockJRPCSystem struct {
	japi *rpc.JSONRPCServer
}

func (mock *mockJRPCSystem) OnStartup(m *mockSystem) {
	mock.japi = rpc.NewJSONRPCServer(m.q.Client())
	go mock.japi.Listen()
}

func (mock *mockJRPCSystem) OnStop() {
	mock.japi.Close()
}

func (mock *mockJRPCSystem) newRpcCtx(methed string, params, res interface{}) error {
	ctx := NewRpcCtx(methed, params, res)
	return ctx.Run()
}

type mockGRPCSystem struct {
	mockSystem
}

func (mock *mockGRPCSystem) OnStartup(m *mockSystem) {
}

func (mock *mockGRPCSystem) OnStop() {
}
