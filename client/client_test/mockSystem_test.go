package client_test

import (
	"flag"

	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/common/config"
	"gitlab.33.cn/chain33/chain33/common/log"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/rpc"
)

var (
	configPath = flag.String("f", "../../cmd/chain33/chain33.test.toml", "configfile")
)

func init() {
	cfg := config.InitCfg(*configPath)
	cfg.GetRpc().GrpcBindAddr = "localhost:8803"
	rpc.Init(cfg.Rpc)
	log.SetFileLog(cfg.Log)
}

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
	ctx := NewJsonRpcCtx(methed, params, res)
	return ctx.Run()
}

type mockGRPCSystem struct {
	gapi *rpc.Grpcserver
}

func (mock *mockGRPCSystem) OnStartup(m *mockSystem) {
	mock.gapi = rpc.NewGRpcServer(m.q.Client())
	go mock.gapi.Listen()
}

func (mock *mockGRPCSystem) OnStop() {
	mock.gapi.Close()
}

func (mock *mockGRPCSystem) newRpcCtx(method string, param, res interface{}) error {
	ctx := NewGRpcCtx(method, param, res)
	return ctx.Run()
}
