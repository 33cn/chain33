package extension

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	circuit "github.com/libp2p/go-libp2p/p2p/protocol/internal/circuitv1-deprecated
	"github.com/stretchr/testify/assert"

	bhost "github.com/libp2p/go-libp2p-blankhost"
	"github.com/libp2p/go-libp2p/core/host"
	discovery "github.com/libp2p/go-libp2p/p2p/discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	swarmt "github.com/libp2p/go-libp2p/p2p/net/swarm/testing"
	"github.com/stretchr/testify/require"
)

func getNetHosts(ctx context.Context, n int, t *testing.T) []host.Host {
	var out []host.Host

	for i := 0; i < n; i++ {
		netw := swarmt.GenSwarm(t, ctx)
		h := bhost.NewBlankHost(netw)
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

func TestRelay(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	hosts := getNetHosts(ctx, 3, t)
	connect(t, hosts[0], hosts[1])
	connect(t, hosts[1], hosts[2])
	//第二个节点作为中继节点
	h1dht, err := dht.New(ctx, hosts[1])
	require.Nil(t, err)
	h2dht, err := dht.New(ctx, hosts[2])
	require.Nil(t, err)
	h1dht.Bootstrap(ctx)
	hoprelay := NewRelayDiscovery(hosts[1], discovery.NewRoutingDiscovery(h1dht), circuit.OptHop)
	hoprelay.Advertise(ctx)
	r2 := NewRelayDiscovery(hosts[2], discovery.NewRoutingDiscovery(h2dht))
	wait := make(chan bool)
	msg := []byte("relay works!")
	go func() {
		//第三个节点监听
		wait <- true
		list := r2.crelay.Listener()
		conn, err := list.Accept()
		if err != nil {
			t.Error(err)
			return
		}
		_, err = conn.Write(msg)
		if err != nil {
			t.Error(err)
			return
		}
	}()

	<-wait
	rinfo := hosts[1].Peerstore().PeerInfo(hosts[1].ID()) //中继节点的peerinfo
	require.NotNil(t, rinfo.Addrs)
	dinfo := hosts[2].Peerstore().PeerInfo(hosts[2].ID()) //目的节点
	require.NotNil(t, dinfo.Addrs)
	h0dht, err := dht.New(ctx, hosts[0])
	require.Nil(t, err)
	_, err = h0dht.RoutingTable().TryAddPeer(hosts[1].ID(), true, true)
	require.Nil(t, err)
	relayPeer := NewRelayDiscovery(hosts[0], discovery.NewRoutingDiscovery(h0dht))
	conn, err := relayPeer.DialDestPeer(rinfo, dinfo)
	if err != nil {
		t.Log("dial err:", err)
	}
	assert.Nil(t, err)
	err = conn.SetReadDeadline(time.Now().Add(time.Second))
	require.Nil(t, err)
	result := make([]byte, len(msg))
	_, err = io.ReadFull(conn, result)
	assert.Nil(t, err)
	ok := bytes.Equal(result, msg)
	assert.True(t, ok)

	testFindOpPeers(t, hoprelay)
	testCheckOp(t, relayPeer, hosts[1])
	hosts[0].Close()
	hosts[1].Close()
	hosts[2].Close()

}

func testCheckOp(t *testing.T, netRely *Relay, h host.Host) {
	//check op
	ok, err := netRely.CheckHOp(h.ID())
	require.Nil(t, err)
	require.True(t, ok)
}

func testFindOpPeers(t *testing.T, netRely *Relay) {
	peers, err := netRely.FindOpPeers()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(peers)
}
