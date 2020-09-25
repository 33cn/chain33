package healthy

import (
	"context"
	"crypto/rand"

	"fmt"
	"testing"
	"time"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/system/p2p/dht/net"
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	types2 "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
)

func TestHandler(t *testing.T) {
	protocol.ClearEventHandler()
	q := queue.New("test")
	initBlockchain(q)
	pList := initEnv(t, q)
	p1, p2 := pList[0], pList[1]
	height, err := testLastHeader(p2, p1.Host.ID())
	assert.Nil(t, err)
	assert.Equal(t, int64(14000), height)
	var ok bool
	ok, err = testIsHealthy(p1, p2.Host.ID())
	assert.Nil(t, err)
	assert.False(t, ok)
	ok, err = testIsSync(p2, p1.Host.ID())
	assert.Nil(t, err)
	assert.True(t, ok)
	ok, err = testIsHealthy(p2, p1.Host.ID())
	assert.Nil(t, err)
	assert.True(t, ok)

	p2.updateFallBehind()
	ok, err = testIsHealthy(p1, p2.Host.ID())
	assert.Nil(t, err)
	assert.True(t, ok)
}

func testLastHeader(p *Protocol, id peer.ID) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()
	stream, err := p.Host.NewStream(ctx, id, protocol.GetLastHeader)
	if err != nil {
		return -1, err
	}
	defer protocol.CloseStream(stream)
	err = protocol.WriteStream(&types.P2PRequest{}, stream)
	if err != nil {
		return -1, err
	}
	var res types.P2PResponse
	err = protocol.ReadStream(&res, stream)
	if err != nil {
		return -1, err
	}
	if header, ok := res.Response.(*types.P2PResponse_LastHeader); ok {
		return header.LastHeader.Height, nil
	}
	return -1, types2.ErrUnknown
}

func testIsSync(p *Protocol, id peer.ID) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	stream, err := p.Host.NewStream(ctx, id, protocol.IsSync)
	if err != nil {
		return false, err
	}
	defer protocol.CloseStream(stream)
	err = protocol.WriteStream(&types.P2PRequest{}, stream)
	if err != nil {
		return false, err
	}
	var res types.P2PResponse
	err = protocol.ReadStream(&res, stream)
	if err != nil {
		return false, err
	}
	if reply, ok := res.Response.(*types.P2PResponse_Reply); ok {
		return reply.Reply.IsOk, nil
	}
	return false, types2.ErrUnknown
}

func testIsHealthy(p *Protocol, id peer.ID) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	stream, err := p.Host.NewStream(ctx, id, protocol.IsHealthy)
	if err != nil {
		return false, err
	}
	defer protocol.CloseStream(stream)
	err = protocol.WriteStream(&types.P2PRequest{
		Request: &types.P2PRequest_HealthyHeight{
			HealthyHeight: 50,
		},
	}, stream)
	if err != nil {
		return false, err
	}
	var res types.P2PResponse
	err = protocol.ReadStream(&res, stream)
	if err != nil {
		return false, err
	}
	if reply, ok := res.Response.(*types.P2PResponse_Reply); ok {
		return reply.Reply.IsOk, nil
	}
	return false, types2.ErrUnknown
}

func initEnv(t *testing.T, q queue.Queue) []*Protocol {
	r := rand.Reader
	priv1, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		panic(err)
	}
	priv2, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)

	if err != nil {
		panic(err)
	}
	client1, client2 := q.Client(), q.Client()
	m1, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", 13806))
	if err != nil {
		t.Fatal(err)
	}
	host1, err := libp2p.New(context.Background(), libp2p.ListenAddrs(m1), libp2p.Identity(priv1))
	if err != nil {
		t.Fatal(err)
	}

	m2, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", 13807))
	if err != nil {
		t.Fatal(err)
	}
	host2, err := libp2p.New(context.Background(), libp2p.ListenAddrs(m2), libp2p.Identity(priv2))
	if err != nil {
		t.Fatal(err)
	}
	t.Log("h1", host1.ID(), "h2", host2.ID())
	cfg := types.NewChain33Config(types.ReadFile("../../../../../cmd/chain33/chain33.test.toml"))
	mcfg := &types2.P2PSubConfig{}
	types.MustDecode(cfg.GetSubConfig().P2P[types2.DHTTypeName], mcfg)
	mcfg.DisableFindLANPeers = true
	discovery1 := net.InitDhtDiscovery(context.Background(), host1, nil, cfg, &types2.P2PSubConfig{Channel: 888})
	discovery1.Start()

	env1 := protocol.P2PEnv{
		Ctx:              context.Background(),
		ChainCfg:         cfg,
		QueueClient:      client1,
		Host:             host1,
		SubConfig:        mcfg,
		RoutingDiscovery: discovery1.RoutingDiscovery,
		RoutingTable:     discovery1.RoutingTable(),
	}
	InitProtocol(&env1)
	p1 := &Protocol{
		P2PEnv:     &env1,
		fallBehind: 1 << 30,
	}

	discovery2 := net.InitDhtDiscovery(context.Background(), host2, nil, cfg, &types2.P2PSubConfig{
		Seeds:   []string{fmt.Sprintf("/ip4/127.0.0.1/tcp/13806/p2p/%s", host1.ID().Pretty())},
		Channel: 888,
	})
	discovery2.Start()

	env2 := protocol.P2PEnv{
		ChainCfg:         cfg,
		QueueClient:      client2,
		Host:             host2,
		SubConfig:        mcfg,
		RoutingDiscovery: discovery2.RoutingDiscovery,
		RoutingTable:     discovery2.RoutingTable(),
		Ctx:              context.Background(),
	}
	p2 := &Protocol{
		P2PEnv:     &env2,
		fallBehind: 1 << 30,
	}
	//注册p2p通信协议，用于处理节点之间请求
	p2.Host.SetStreamHandler(protocol.IsHealthy, protocol.HandlerWithRW(p2.handleStreamIsHealthy))
	p2.Host.SetStreamHandler(protocol.IsSync, protocol.HandlerWithRW(p2.handleStreamIsSync))
	p2.Host.SetStreamHandler(protocol.GetLastHeader, protocol.HandlerWithRW(p2.handleStreamLastHeader))
	client1.Sub("p2p")
	client2.Sub("p2p2")

	return []*Protocol{p1, p2}
}

func initBlockchain(q queue.Queue) {
	client := q.Client()
	client.Sub("blockchain")
	go func() {
		for msg := range client.Recv() {
			switch msg.Ty {
			case types.EventGetLastHeader:
				msg.Reply(queue.NewMessage(0, "", 0, &types.Header{Height: 14000}))
			}
		}
	}()
}
