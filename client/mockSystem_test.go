package client

import (
	"flag"
	"time"

	"gitlab.33.cn/chain33/chain33/common/config"
	"gitlab.33.cn/chain33/chain33/common/log"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/rpc"
)

var (
	configPath = flag.String("f", "../cmd/chain33/chain33.test.toml", "configfile")
)

func init() {
	cfg := config.InitCfg(*configPath)
	rpc.Init(cfg.Rpc)
	log.SetLogLevel("crit")
}

type MockLive interface {
	OnStartup(m *mockSystem)
	OnStop()
}

type mockSystem struct {
	grpcMock  MockLive
	jrpcMock  MockLive
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
	if mock.grpcMock != nil {
		mock.grpcMock.OnStartup(mock)
	}
	if mock.jrpcMock != nil {
		mock.jrpcMock.OnStartup(mock)
	}
	return mock.getAPI()
}

func (mock *mockSystem) stop() {
	if mock.jrpcMock != nil {
		mock.jrpcMock.OnStop()
	}
	if mock.grpcMock != nil {
		mock.grpcMock.OnStop()
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
	ctx  *JsonRpcCtx
}

func (mock *mockJRPCSystem) OnStartup(m *mockSystem) {
	println("=============jrpc====")
	mock.japi = rpc.NewJSONRPCServer(m.q.Client())
	ch := make(chan struct{}, 1)
	go func() {
		ch <- struct{}{}
		mock.japi.Listen()
	}()
	<-ch
	time.Sleep(time.Millisecond)
}

func (mock *mockJRPCSystem) OnStop() {
	mock.japi.Close()
}

func (mock *mockJRPCSystem) newRpcCtx(methed string, params, res interface{}) error {
	if mock.ctx == nil {
		mock.ctx = NewJsonRpcCtx(methed, params, res)
	} else {
		mock.ctx.Method = methed
		mock.ctx.Params = params
		mock.ctx.Res = res
	}
	return mock.ctx.Run()
}

type mockGRPCSystem struct {
	gapi *rpc.Grpcserver
	ctx  *GrpcCtx
}

func (mock *mockGRPCSystem) OnStartup(m *mockSystem) {
	println("=============grpc====")
	mock.gapi = rpc.NewGRpcServer(m.q.Client())
	ch := make(chan struct{}, 1)
	go func() {
		ch <- struct{}{}
		mock.gapi.Listen()
	}()
	<-ch
	time.Sleep(time.Millisecond)
}

func (mock *mockGRPCSystem) OnStop() {
	mock.gapi.Close()
}

func (mock *mockGRPCSystem) newRpcCtx(method string, param, res interface{}) error {
	if mock.ctx == nil {
		mock.ctx = NewGRpcCtx(method, param, res)
	} else {
		mock.ctx.Method = method
		mock.ctx.Params = param
		mock.ctx.Res = res
	}
	return mock.ctx.Run()
}
