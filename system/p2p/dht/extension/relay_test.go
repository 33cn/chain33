package extension

import (
	"bytes"
	"context"
	"io"
	"net"
	"testing"
	"time"

	bhost "github.com/libp2p/go-libp2p-blankhost"
	circuit "github.com/libp2p/go-libp2p-circuit"
	"github.com/libp2p/go-libp2p-core/host"
	discovery "github.com/libp2p/go-libp2p-discovery"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	swarmt "github.com/libp2p/go-libp2p-swarm/testing"
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
	pinfo := a.Peerstore().PeerInfo(a.ID())
	err := b.Connect(context.Background(), pinfo)
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
	newRelay(ctx, hosts[1], circuit.OptHop)
	r2, err := newRelay(ctx, hosts[2])
	require.Nil(t, err)

	var (
		conn1, conn2 net.Conn
		done         = make(chan struct{})
	)

	defer func() {
		<-done
		if conn1 != nil {
			conn1.Close()
		}
		if conn2 != nil {
			conn2.Close()
		}
	}()
	msg := []byte("relay works!")
	go func() {
		defer close(done)
		//第三个节点监听
		list := r2.Listener()

		var err error
		conn1, err = list.Accept()
		if err != nil {
			t.Error(err)
			return
		}

		_, err = conn1.Write(msg)
		if err != nil {
			t.Error(err)
			return
		}
	}()

	rinfo := hosts[1].Peerstore().PeerInfo(hosts[1].ID()) //中继节点的peerinfo
	require.NotNil(t, rinfo.Addrs)
	dinfo := hosts[2].Peerstore().PeerInfo(hosts[2].ID()) //目的节点
	require.NotNil(t, dinfo.Addrs)
	kademliaDHT, err := dht.New(context.Background(), hosts[0])
	require.Nil(t, err)
	_, err = kademliaDHT.RoutingTable().Update(hosts[1].ID())
	require.Nil(t, err)
	netRely := NewRelayDiscovery(hosts[0], discovery.NewRoutingDiscovery(kademliaDHT))
	netRely.Advertise(ctx)
	conn2, err = netRely.DialDestPeer(rinfo, dinfo)
	if err != nil {
		t.Fatal(err)
	}

	err = conn2.SetReadDeadline(time.Now().Add(time.Second))
	require.Nil(t, err)
	result := make([]byte, len(msg))
	_, err = io.ReadFull(conn2, result)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(result, msg) {
		t.Fatal("message was incorrect:", string(result))
	}

	testCheckOp(t, netRely, hosts[1])
	testFindOpPeers(t, netRely)

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
