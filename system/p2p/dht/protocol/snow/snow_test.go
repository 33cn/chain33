package snow

import (
	"context"
	"testing"
	"time"

	"github.com/33cn/chain33/client/mocks"
	commlog "github.com/33cn/chain33/common/log"
	"github.com/33cn/chain33/queue"
	prototypes "github.com/33cn/chain33/system/p2p/dht/protocol"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func init() {
	commlog.SetLogLevel("error")
}

func newHost(t *testing.T) core.Host {

	host, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"), libp2p.Transport(tcp.NewTCPTransport))
	require.Nil(t, err)

	return host
}

func newTestEnv(q queue.Queue, t *testing.T) (*prototypes.P2PEnv, context.CancelFunc) {
	cfg := types.NewChain33Config(types.ReadFile("../../../../../cmd/chain33/chain33.test.toml"))
	env := &prototypes.P2PEnv{
		ChainCfg:    cfg,
		QueueClient: q.Client(),
		Host:        newHost(t),
		API:         new(mocks.QueueProtocolAPI),
		Ctx:         context.Background(),
	}
	ctx, cancel := context.WithCancel(context.Background())
	env.Ctx = ctx
	return env, cancel
}

func newSnowWithQueue(q queue.Queue, t *testing.T) (*snowman, context.CancelFunc) {
	env, cancel := newTestEnv(q, t)
	prototypes.ClearEventHandler()
	p := &snowman{}
	p.init(env)
	return p, cancel
}

func handleSub(q queue.Queue, topic string, handler func(*queue.Message)) {
	cli := q.Client()
	cli.Sub(topic)
	for msg := range cli.Recv() {
		handler(msg)
	}
}

func checkDone(done chan struct{}, t *testing.T) {
	select {
	case <-done:
	case <-time.After(time.Second * 3):
		t.Error("test timeout")
	}
}

func TestSnowmanChits(t *testing.T) {

	q := queue.New("test")
	sm, cancel := newSnowWithQueue(q, t)
	sm2, cancel2 := newSnowWithQueue(q, t)
	q.SetConfig(sm.ChainCfg)
	defer func() {
		q.Close()
		cancel2()
		cancel()
		sm.Host.Close()
		sm2.Host.Close()
	}()

	addrInfo := peer.AddrInfo{
		ID:    sm2.Host.ID(),
		Addrs: sm2.Host.Addrs(),
	}

	hash := []byte("testhash")
	chits := &types.SnowChits{
		RequestID:        1,
		PeerName:         sm2.Host.ID().String(),
		PreferredBlkHash: hash,
	}

	done := make(chan struct{})
	go handleSub(q, "consensus", func(msg *queue.Message) {

		if msg.ID == types.EventSnowmanQueryFailed {
			rep, ok := msg.Data.(*types.SnowFailedQuery)
			require.True(t, ok)
			require.Equal(t, chits.RequestID, rep.RequestID)
			return
		}
		defer close(done)
		req, ok := msg.Data.(*types.SnowChits)
		require.True(t, ok)
		require.Equal(t, chits.RequestID, req.RequestID)
		require.Equal(t, sm.Host.ID().String(), req.PeerName)
		require.Equal(t, string(hash), string(req.GetPreferredBlkHash()))

	})

	sm.handleEventChits(sm.QueueClient.NewMessage("", 0, chits))
	err := sm.Host.Connect(sm.Ctx, addrInfo)
	require.Nil(t, err)
	sm.handleEventChits(sm.QueueClient.NewMessage("", 0, chits))
	checkDone(done, t)
}

func TestSnowmanGetBlock(t *testing.T) {

	q := queue.New("test")
	sm, cancel := newSnowWithQueue(q, t)
	sm2, cancel2 := newSnowWithQueue(q, t)
	q.SetConfig(sm.ChainCfg)
	go q.Start()
	defer func() {
		q.Close()
		cancel2()
		cancel()
		sm.Host.Close()
		sm2.Host.Close()
	}()

	addrInfo := peer.AddrInfo{
		ID:    sm2.Host.ID(),
		Addrs: sm2.Host.Addrs(),
	}
	err := sm.Host.Connect(sm.Ctx, addrInfo)
	require.Nil(t, err)
	blk := &types.Block{Height: 1}
	req := &types.SnowGetBlock{
		RequestID: 1,
		PeerName:  sm2.Host.ID().String(),
	}
	api := sm2.API.(*mocks.QueueProtocolAPI)
	api.On("GetBlockByHashes", mock.Anything).Return(&types.BlockDetails{Items: []*types.BlockDetail{{Block: blk}}}, nil)
	done := make(chan struct{})
	go handleSub(q, "consensus", func(msg *queue.Message) {
		defer close(done)
		require.Equal(t, types.EventSnowmanPutBlock, int(msg.ID))
		require.Equal(t, types.EventForFinalizer, int(msg.Ty))
		reply, ok := msg.Data.(*types.SnowPutBlock)
		require.True(t, ok)
		require.Equal(t, req.RequestID, reply.RequestID)
		require.Equal(t, req.PeerName, reply.PeerName)
	})
	sm.handleEventGetBlock(sm.QueueClient.NewMessage("", 0, req))
	checkDone(done, t)
}

func TestSnowmanPullQuery(t *testing.T) {

	q := queue.New("test")
	sm, cancel := newSnowWithQueue(q, t)
	sm2, cancel2 := newSnowWithQueue(q, t)
	q.SetConfig(sm.ChainCfg)
	go q.Start()
	defer func() {
		q.Close()
		cancel2()
		cancel()
		sm.Host.Close()
		sm2.Host.Close()
	}()

	addrInfo := peer.AddrInfo{
		ID:    sm2.Host.ID(),
		Addrs: sm2.Host.Addrs(),
	}
	err := sm.Host.Connect(sm.Ctx, addrInfo)
	require.Nil(t, err)
	hash := []byte("testhash")
	req := &types.SnowPullQuery{
		RequestID: 1,
		PeerName:  sm2.Host.ID().String(),
		BlockHash: hash,
	}
	done := make(chan struct{})
	go handleSub(q, "consensus", func(msg *queue.Message) {
		defer close(done)
		reply, ok := msg.Data.(*types.SnowPullQuery)
		require.True(t, ok)
		require.Equal(t, req.RequestID, req.RequestID)
		require.Equal(t, sm.Host.ID().String(), reply.PeerName)
		require.Equal(t, string(hash), string(reply.GetBlockHash()))

	})
	sm.handleEventPullQuery(sm.QueueClient.NewMessage("", 0, req))
	checkDone(done, t)
}

func TestSnowmanPushQuery(t *testing.T) {

	q := queue.New("test")
	sm, cancel := newSnowWithQueue(q, t)
	sm2, cancel2 := newSnowWithQueue(q, t)
	q.SetConfig(sm.ChainCfg)
	go q.Start()
	defer func() {
		q.Close()
		cancel2()
		cancel()
		sm.Host.Close()
		sm2.Host.Close()
	}()

	addrInfo := peer.AddrInfo{
		ID:    sm2.Host.ID(),
		Addrs: sm2.Host.Addrs(),
	}
	err := sm.Host.Connect(sm.Ctx, addrInfo)
	require.Nil(t, err)
	req := &types.SnowPushQuery{
		RequestID: 1,
		PeerName:  sm2.Host.ID().String(),
	}
	done := make(chan struct{})
	go handleSub(q, "consensus", func(msg *queue.Message) {
		defer close(done)
		reply, ok := msg.Data.(*types.SnowPushQuery)
		require.True(t, ok)
		require.Equal(t, req.RequestID, req.RequestID)
		require.Equal(t, sm.Host.ID().String(), reply.PeerName)

	})
	sm.handleEventPushQuery(sm.QueueClient.NewMessage("", 0, req))
	checkDone(done, t)
}
