// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package client_test

import (
	"flag"
	"time"

	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/common/log"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/rpc"
	"github.com/33cn/chain33/types"
)

var (
	configPath = flag.String("f", "../cmd/chain33/chain33.test.toml", "configfile")

	jrpcaddr = "localhost:8801"
	jrpcsite = "http://localhost:8801"
	grpcaddr = "localhost:8802"
	grpcsite = "localhost:8802"
)

func init() {
	cfg, sub := types.InitCfg(*configPath)
	println(sub)
	// change rpc bind address
	cfg.RPC.JrpcBindAddr = jrpcaddr
	cfg.RPC.GrpcBindAddr = grpcaddr
	rpc.InitCfg(cfg.RPC)
	log.SetLogLevel("crit")
}

type MockLive interface {
	OnStartup(m *mockSystem)
	OnStop()
}

type mockSystem struct {
	grpcMock  MockLive
	jrpcMock  MockLive
	q         *mockQueue
	chain     *mockBlockChain
	mem       *mockMempool
	wallet    *mockWallet
	p2p       *mockP2P
	consensus *mockConsensus
	store     *mockStore
	execs     *mockExecs
}

type mockClient struct {
	c queue.Client
}

func (mock *mockClient) Send(msg *queue.Message, waitReply bool) error {
	if msg.Topic == "error" {
		return types.ErrInvalidParam
	}
	return mock.c.Send(msg, waitReply)
}

func (mock *mockClient) SendTimeout(msg *queue.Message, waitReply bool, timeout time.Duration) error {
	return mock.c.SendTimeout(msg, waitReply, timeout)
}

func (mock *mockClient) Wait(msg *queue.Message) (*queue.Message, error) {
	return mock.c.Wait(msg)
}

func (mock *mockClient) Reply(msg *queue.Message) {
	mock.c.Reply(msg)
}

func (mock *mockClient) WaitTimeout(msg *queue.Message, timeout time.Duration) (*queue.Message, error) {
	return mock.c.WaitTimeout(msg, timeout)
}

func (mock *mockClient) Recv() chan *queue.Message {
	return mock.c.Recv()
}

func (mock *mockClient) Sub(topic string) {
	mock.c.Sub(topic)
}

func (mock *mockClient) Close() {
	mock.c.Close()
}

func (mock *mockClient) CloseQueue() (*types.Reply, error) {
	mock.c.CloseQueue()
	return &types.Reply{IsOk: true, Msg: []byte("Ok")}, nil
}

func (mock *mockClient) NewMessage(topic string, ty int64, data interface{}) *queue.Message {
	return mock.c.NewMessage(topic, ty, data)
}

func (mock *mockClient) Clone() queue.Client {
	clone := mockClient{}
	clone.c = mock.c
	return &clone
}

type mockQueue struct {
	q queue.Queue
}

func (mock *mockQueue) Close() {
	mock.q.Close()
}

func (mock *mockQueue) Start() {
	mock.q.Start()
}

func (mock *mockQueue) Client() queue.Client {
	client := mockClient{}
	client.c = mock.q.Client()
	return &client
}

func (mock *mockQueue) Name() string {
	return mock.q.Name()
}

func (mock *mockSystem) startup(size int) client.QueueProtocolAPI {

	var q = queue.New("channel")
	queue := &mockQueue{q: q}
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
	execs := new(mockExecs)
	execs.SetQueueClient(q)

	mock.q = queue
	mock.chain = chain
	mock.mem = mem
	mock.wallet = wallet
	mock.p2p = p2p
	mock.consensus = consensus
	mock.store = store
	mock.execs = execs
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
	mock.execs.Close()
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
	mock.japi = rpc.NewJSONRPCServer(m.q.Client(), nil)
	ch := make(chan struct{}, 1)
	go func() {
		ch <- struct{}{}
		_, err := mock.japi.Listen()
		if err != nil {
			panic(err)
		}
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
	mock.gapi = rpc.NewGRpcServer(m.q.Client(), nil)
	ch := make(chan struct{}, 1)
	go func() {
		ch <- struct{}{}
		_, err := mock.gapi.Listen()
		if err != nil {
			panic(err)
		}
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
