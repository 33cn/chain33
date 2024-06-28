package extension

import (
	"context"
	"testing"
	"time"

	ma "github.com/multiformats/go-multiaddr"

	"github.com/libp2p/go-libp2p/core/network"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"

	"github.com/libp2p/go-libp2p"

	"github.com/libp2p/go-libp2p/core/host"
	bhost "github.com/libp2p/go-libp2p/p2p/host/blank"
	swarmt "github.com/libp2p/go-libp2p/p2p/net/swarm/testing"
)

func getNetHosts(n int, t *testing.T) []host.Host {
	var out []host.Host

	for i := 0; i < n; i++ {
		h := bhost.NewBlankHost(swarmt.GenSwarm(t))
		out = append(out, h)
	}

	return out
}

func connect(t *testing.T, a, b host.Host) {
	pinfo := b.Peerstore().PeerInfo(b.ID())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := a.Connect(ctx, pinfo)
	if err != nil {
		t.Fatal(err)
	}
}

func relayCliStreamHander(h host.Host, t *testing.T) {
	// Now, to test the communication, let's set up a protocol handler on unreachable2
	h.SetStreamHandler("/chain33/test/customprotocol", func(s network.Stream) {
		t.Log("Awesome! We're now communicating via the relay!,i am:", h.ID())
		var buf [1024]byte
		rlen, _ := s.Read(buf[:])
		t.Log("read from:", s.Conn().RemotePeer())
		t.Log("content:", string(buf[:rlen]))

		s.Write([]byte("yeap,i hear u"))
		s.Close()
	})
}

func TestRelayV2(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx = network.WithUseTransient(ctx, "test")

	unreachable1, err := libp2p.New(
		libp2p.NoListenAddrs,
		libp2p.EnableRelay(),
	)
	if err != nil {
		t.Log("new unreachable1 err:", err)
		return
	}

	unreachable2, err := libp2p.New(
		libp2p.NoListenAddrs,
		libp2p.EnableRelay(),
	)
	if err != nil {
		t.Log("new unreachable2 err:", err)
		return
	}
	unreachable2info := peer.AddrInfo{
		ID:    unreachable2.ID(),
		Addrs: unreachable2.Addrs(),
	}
	err = unreachable1.Connect(ctx, unreachable2info)
	assert.NotNil(t, err)

	hosts := getNetHosts(1, t)
	t.Log("host id:", hosts[0].ID())
	// 把host[0] 当作中间（中继）节点
	r := MakeNodeRelayService(hosts[0], nil)
	defer r.Close()
	relayInfo := peer.AddrInfo{
		ID:    hosts[0].ID(),
		Addrs: hosts[0].Addrs(),
	}

	relayCliStreamHander(unreachable2, t)
	err = unreachable1.Connect(ctx, relayInfo)
	assert.Nil(t, err)

	err = unreachable2.Connect(ctx, relayInfo)
	assert.Nil(t, err)

	go ReserveRelaySlot(ctx, unreachable1, unreachable2info, time.Millisecond)

	//unreachable2 向中继节点申请一个中继槽slot
	go ReserveRelaySlot(ctx, unreachable2, relayInfo, time.Millisecond*100)
	time.Sleep(time.Second)
	relayaddr, err := MakeRelayAddrs(relayInfo.ID.String(), unreachable2.ID().String())
	assert.Nil(t, err)
	t.Log("relayaddr:", relayaddr)
	unreachable2relayinfo := peer.AddrInfo{
		ID:    unreachable2.ID(),
		Addrs: []ma.Multiaddr{relayaddr},
	}
	err = unreachable1.Connect(ctx, unreachable2relayinfo)
	assert.Nil(t, err)

	s1, err := unreachable1.NewStream(ctx, unreachable2.ID(), "/chain33/test/customprotocol")
	assert.Nil(t, err)

	wlen, err := s1.Write([]byte("hello, unreachable2,i am coming..."))
	assert.Nil(t, err)
	t.Log("wring len:", wlen)
	var read [1024]byte
	rlen, err := s1.Read(read[:])
	assert.Nil(t, err)
	t.Log("read:", string(read[:rlen]))

	err = unreachable1.Connect(ctx, unreachable2info)
	assert.Nil(t, err)

}
