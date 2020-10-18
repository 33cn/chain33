package net

import (
	"context"
	"testing"
	"time"

	p2pty "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"
)

func Test_Discovery(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hosts := getNetHosts(ctx, 4, t)
	t.Log("h0", hosts[0].ID())
	t.Log("h1", hosts[1].ID())
	t.Log("h2", hosts[2].ID())
	t.Log("h3", hosts[3].ID())
	haddrinfo1 := peer.AddrInfo{ID: hosts[1].ID(), Addrs: hosts[1].Addrs()}
	cfg := types.NewChain33Config(types.ReadFile("../../../../cmd/chain33/chain33.test.toml"))
	mcfg := &p2pty.P2PSubConfig{}
	types.MustDecode(cfg.GetSubConfig().P2P["dht"], mcfg)
	mcfg.RelayEnable = true
	dis := InitDhtDiscovery(ctx, hosts[0], []peer.AddrInfo{haddrinfo1}, cfg, mcfg)
	assert.NotNil(t, dis)
	dis.Start()
	assert.NotNil(t, dis.RoutingDiscovery)
	t.Log("listpeer", dis.ListPeers())
	dis.host.Peerstore().AddAddrs(hosts[3].ID(), hosts[3].Addrs(), time.Minute)
	dis.host.Peerstore().AddAddrs(hosts[2].ID(), hosts[2].Addrs(), time.Minute)
	dis.Update(hosts[3].ID()) //增加一个节点
	t.Log("listpeer", dis.ListPeers())
	assert.Equal(t, 1, len(dis.ListPeers()))
	dis.Update(hosts[2].ID())
	assert.Equal(t, 2, len(dis.ListPeers()))
	//查找节点
	peers := dis.FindNearestPeers(hosts[2].ID(), 2)
	t.Log("peers", peers)
	assert.Equal(t, hosts[3].ID(), peers[1])

	pinfos := dis.FindLocalPeers([]peer.ID{hosts[3].ID()})
	t.Log("pinfos", pinfos)
	dis.Remove(hosts[3].ID())
	assert.Equal(t, 1, len(dis.ListPeers()))
	t.Log("routingTable size", dis.RoutingTableSize())
}
