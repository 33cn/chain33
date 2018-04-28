package client_test

import (
	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/queue"
)

type mockSystem struct {
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
	mock.onStartup()
	return mock.getAPI()
}
func (mock *mockSystem) onStartup() {

}

func (mock *mockSystem) stop() {
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
	mockSystem
}

func (mock *mockJRPCSystem) onStartup() {
	mock.mockSystem.onStartup()
}

type mockGRPCSystem struct {
	mockSystem
}

func (mock *mockGRPCSystem) onStartup() {
	mock.mockSystem.onStartup()
}
