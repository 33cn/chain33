package manage

import (
	"context"
	"fmt"
	"io"
	"sort"
	"testing"
	"time"

	p2pty "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/metrics"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/multiformats/go-multiaddr"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/stretchr/testify/require"
)

type testConn struct {
	io.Closer
	network.ConnSecurity
	network.ConnMultiaddrs
	stat network.Stat
}

func (t testConn) ID() string {
	return ""
}

// NewStream constructs a new Stream over this conn.
func (t testConn) NewStream() (network.Stream, error) { return nil, nil }

// GetStreams returns all open streams over this conn.
func (t testConn) GetStreams() []network.Stream { return nil }

// Stat stores metadata pertaining to this conn.
func (t testConn) Stat() network.Stat {
	return t.stat
}

func newtestConn(stat network.Stat) network.Conn {
	return testConn{stat: stat}

}
func Test_SortConn(t *testing.T) {
	var testconn conns

	var s1, s2, s3 network.Stat
	s1.Opened = time.Now().Add(time.Second * 10)
	s2.Opened = time.Now().Add(time.Second * 15)
	s3.Opened = time.Now().Add(time.Minute)
	c1 := newtestConn(s1)
	c2 := newtestConn(s2)
	c3 := newtestConn(s3)

	testconn = append(testconn, c1, c2, c3)
	sort.Sort(testconn)
	require.Equal(t, testconn[0], c3)
	require.Equal(t, testconn[2], c1)
}

func TestConnManager(t *testing.T) {
	m1, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", 13666))
	require.Nil(t, err)
	h1, err := libp2p.New(context.Background(), libp2p.ListenAddrs(m1))
	require.Nil(t, err)
	m2, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", 13777))
	require.Nil(t, err)
	h2, err := libp2p.New(context.Background(), libp2p.ListenAddrs(m2))
	require.Nil(t, err)

	addr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/13666/p2p/%s", h1.ID().Pretty()))
	peerInfo, _ := peer.AddrInfoFromP2pAddr(addr)
	err = h2.Connect(context.Background(), *peerInfo)
	require.Nil(t, err)

	kademliaDHT, err := dht.New(context.Background(), h1)
	require.Nil(t, err)
	bandwidthTracker := metrics.NewBandwidthCounter()
	protocolID := protocol.ID("test")
	bandwidthTracker.LogSentMessageStream(1024, protocolID, h2.ID())
	bandwidthTracker.LogRecvMessageStream(2048, protocolID, h2.ID())
	subCfg := &p2pty.P2PSubConfig{}
	mgr := NewConnManager(context.Background(), h1, kademliaDHT.RoutingTable(), bandwidthTracker, subCfg)
	info := mgr.BandTrackerByProtocol()
	require.NotNil(t, info)
	h1.Peerstore().RecordLatency(h2.ID(), time.Second/100)
	mgr.printMonitorInfo()
	mgr.procConnections()
	kademliaDHT.RoutingTable().Update(h2.ID())
	peers := mgr.FetchNearestPeers(1)
	require.NotNil(t, peers)
	require.Equal(t, h2.ID(), peers[0])

	require.Equal(t, 1, int(mgr.CheckDirection(h2.ID())))
	require.Equal(t, 1, len(mgr.InBounds()))
	require.Equal(t, 0, len(mgr.OutBounds()))

	require.False(t, mgr.IsNeighbors(h2.ID()))
	mgr.AddNeighbors(&peer.AddrInfo{ID: h2.ID()})
	require.True(t, mgr.IsNeighbors(h2.ID()))
}
