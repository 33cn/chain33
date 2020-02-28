package p2pnext

import (
	"context"
	"encoding/hex"
	"fmt"

	"time"

	"github.com/33cn/chain33/client"
	l "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/p2pnext/dht"
	"github.com/33cn/chain33/p2pnext/manage"

	"github.com/33cn/chain33/p2pnext/protocol"
	prototypes "github.com/33cn/chain33/p2pnext/protocol/types"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	libp2p "github.com/libp2p/go-libp2p"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/metrics"
	"github.com/libp2p/go-libp2p-core/peer"

	//"github.com/libp2p/go-libp2p-core/peerstore"
	multiaddr "github.com/multiformats/go-multiaddr"
)

var logger = l.New("module", "p2pnext")

type P2P struct {
	chainCfg      *types.Chain33Config
	host          core.Host
	discovery     *dht.Discovery
	connManag     *manage.ConnManager
	peerInfoManag *manage.PeerInfoManager
	api           client.QueueProtocolAPI
	client        queue.Client
	Done          chan struct{}
	Node          *Node
}

func New(cfg *types.Chain33Config) *P2P {

	mcfg := cfg.GetModuleConfig().P2P
	if mcfg.Port == 0 {
		mcfg.Port = 13803
	}

	m, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", mcfg.Port))
	if err != nil {
		return nil
	}

	logger.Info("NewMulti", "addr", m.String())
	keystr, _ := NewAddrBook(cfg.GetModuleConfig().P2P).GetPrivPubKey()
	logger.Info("loadPrivkey:", keystr)

	//key string convert to crpyto.Privkey
	key, _ := hex.DecodeString(keystr)
	priv, err := crypto.UnmarshalSecp256k1PrivateKey(key)
	if err != nil {
		panic(err)
	}

	bandwidthTracker := metrics.NewBandwidthCounter()
	host, err := libp2p.New(context.Background(),
		libp2p.ListenAddrs(m),
		libp2p.Identity(priv),
		libp2p.BandwidthReporter(bandwidthTracker),
		libp2p.NATPortMap(),
	)

	if err != nil {
		panic(err)
	}
	p2p := &P2P{host: host}
	p2p.connManag = manage.NewConnManager(p2p.host, bandwidthTracker)
	p2p.peerInfoManag = manage.NewPeerInfoManager()
	p2p.chainCfg = cfg
	p2p.discovery = new(dht.Discovery)
	p2p.Node = NewNode(p2p, cfg)
	logger.Info("NewP2p", "peerId", p2p.host.ID(), "addrs", p2p.host.Addrs())
	return p2p
}

func (p *P2P) managePeers() {

	go p.connManag.MonitorAllPeers(p.Node.p2pCfg.Seeds, p.host)
	p.discovery.InitDht(context.Background(), p.host, p.Node.p2pCfg.Seeds)
	for {
		time.Sleep(time.Second * 5)
		tables := p.discovery.RoutingTale()
		logger.Info("managePeers", "RoutingTale show", tables)
		for _, peer := range tables {
			logger.Info("find peer", "peer", peer)
			if peer.Pretty() == p.host.ID().Pretty() {
				logger.Info("Find self...", "ID", p.host.ID())
				continue
			}
			logger.Info("+++++++++++++++++++++++++++++p2p.FindPeers",
				"peer", peer.String())
			logger.Info("All Peers", "Peers", p.host.Peerstore().Peers(),
				"peersWithAddress size", len(p.host.Peerstore().PeersWithAddrs()))
			p.newConn(context.Background(), peer)
		Recheck:
			if p.connManag.Size() >= 25 {
				//达到连接节点数最大要求
				time.Sleep(time.Second * 10)
				goto Recheck
			}
		}
	}

}

func (p *P2P) Wait() {

}

func (p *P2P) Close() {
	close(p.Done)
	logger.Info("p2p closed")

	return
}

// SetQueueClient
func (p *P2P) SetQueueClient(cli queue.Client) {
	var err error
	p.api, err = client.New(cli, nil)
	if err != nil {
		panic("SetQueueClient client.New err")
	}
	if p.client == nil {
		p.client = cli
	}
	//提供给其他插件使用的共享接口
	globalData := &prototypes.GlobalData{
		ChainCfg:        p.chainCfg,
		QueueClient:     p.client,
		Host:            p.host,
		ConnManager:     p.connManag,
		Discovery:       p.discovery,
		PeerInfoManager: p.peerInfoManag,
	}
	protocol.Init(globalData)
	go p.managePeers()
	go p.processP2P()

}

func (p *P2P) processP2P() {

	p.client.Sub("p2p")

	//TODO, control goroutine num
	for msg := range p.client.Recv() {
		go protocol.HandleEvent(msg)
	}
}

func (p *P2P) newConn(ctx context.Context, pid peer.ID) error {

	_, err := p.host.NewStream(context.Background(), pid, protocol.MsgIDs...)
	if err != nil {
		logger.Error("NewStream", "Connect err", err, "remoteID", pid)
		return err
	}
	pinfo, err := p.discovery.FindSpecialPeer(pid)
	p.connManag.Add(pinfo)

	return nil

}
