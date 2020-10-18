package manage

import (
	"context"
	"io"
	"sort"
	"testing"
	"time"

	p2pty "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	bhost "github.com/libp2p/go-libp2p-blankhost"
	"github.com/libp2p/go-libp2p-core/metrics"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	swarmt "github.com/libp2p/go-libp2p-swarm/testing"
	"github.com/stretchr/testify/assert"
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
	assert.Equal(t, testconn[0], c3)
	assert.Equal(t, testconn[2], c1)
}

func Test_ConnManager(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	netw := swarmt.GenSwarm(t, ctx)
	h := bhost.NewBlankHost(netw)
	cfg := types.NewChain33Config(types.ReadFile("../../../../cmd/chain33/chain33.test.toml"))
	mcfg := &p2pty.P2PSubConfig{}
	types.MustDecode(cfg.GetSubConfig().P2P["dht"], mcfg)
	bandwidthTracker := metrics.NewBandwidthCounter()
	cmm := NewConnManager(h, nil, bandwidthTracker, mcfg)
	assert.NotNil(t, cmm)

	ratestr := cmm.RateCaculate(1024)
	t.Log("rate", ratestr)
	netw2 := swarmt.GenSwarm(t, ctx)
	h2 := bhost.NewBlankHost(netw2)
	h2info := peer.AddrInfo{ID: h2.ID(), Addrs: h2.Addrs()}
	cmm.AddNeighbors(&h2info)
	assert.False(t, cmm.IsNeighbors(h.ID()))
	assert.True(t, cmm.IsNeighbors(h2.ID()))
	in, out := cmm.BoundSize()
	assert.Equal(t, 0, in+out)
	h.Connect(ctx, h2info)
	assert.Equal(t, 1, cmm.OutboundSize())
	assert.Equal(t, 1, len(cmm.FetchConnPeers()))

	direction := cmm.CheckDiraction(h2info.ID)
	assert.Equal(t, network.DirOutbound, direction)
	assert.Equal(t, 0, cmm.InboundSize())
	assert.Equal(t, len(cmm.InBounds()), cmm.InboundSize())
	assert.Equal(t, len(cmm.OutBounds()), cmm.OutboundSize())
	cmm.GetNetRate()
	cmm.BandTrackerByProtocol()
	cmm.Close()
}
