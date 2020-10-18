package manage

import (
	"context"
	"testing"
	"time"

	"github.com/33cn/chain33/queue"
	p2pty "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	bhost "github.com/libp2p/go-libp2p-blankhost"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/peer"
	swarmt "github.com/libp2p/go-libp2p-swarm/testing"
	"github.com/stretchr/testify/assert"
)

func prue(pid core.PeerID, beBlack bool) {

}
func Test_peerinfo(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	netw := swarmt.GenSwarm(t, ctx)
	h := bhost.NewBlankHost(netw)
	cfg := types.NewChain33Config(types.ReadFile("../../../../cmd/chain33/chain33.test.toml"))
	mcfg := &p2pty.P2PSubConfig{}
	types.MustDecode(cfg.GetSubConfig().P2P["dht"], mcfg)
	q := queue.New("channel")
	cli := q.Client()
	tcache := NewTimeCache(ctx, time.Second*10)
	infoM := NewPeerInfoManager(h, cli, tcache, prue)
	go infoM.Start()
	netw2 := swarmt.GenSwarm(t, ctx)
	h2 := bhost.NewBlankHost(netw2)
	h2info := peer.AddrInfo{ID: h2.ID(), Addrs: h2.Addrs()}
	infoM.Add(h2info.ID.String(), &types.Peer{})
	assert.Equal(t, 1, len(infoM.FetchPeerInfosInMin()))
	storePeer := infoM.GetPeerInfoInMin(h2info.ID.String())
	t.Log("storePeer", storePeer)
	assert.NotNil(t, storePeer)
	var pinfo types.P2PPeerInfo
	pinfo.Addr = "localhost"
	var tpeer types.Peer
	infoM.Copy(&tpeer, &types.P2PPeerInfo{Addr: "localhost", Port: 123})
	assert.Equal(t, "localhost", tpeer.Addr)
	infoM.Close()
}
