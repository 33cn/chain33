package p2pnext

import (
	"context"
	"encoding/hex"
	"fmt"

	"time"

	"github.com/33cn/chain33/client"
	l "github.com/33cn/chain33/common/log/log15"
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
	"github.com/libp2p/go-libp2p-core/peerstore"
	multiaddr "github.com/multiformats/go-multiaddr"
)

var logger = l.New("module", "p2pnext")

type P2P struct {
	chainCfg         *types.Chain33Config
	host             core.Host
	discovery        *Discovery
	connManag        *manage.ConnManager
	peerInfoManag    *manage.PeerInfoManager
	api              client.QueueProtocolAPI
	client           queue.Client
	Done             chan struct{}
	bandwidthTracker *metrics.BandwidthCounter
	Node             *Node
}

func New(cfg *types.Chain33Config) *P2P {

	mcfg := cfg.GetModuleConfig().P2P
	//TODO 增加P2P channel
	if mcfg.InnerBounds == 0 {
		mcfg.InnerBounds = 500
	}
	logger.Info("p2p", "InnerBounds", mcfg.InnerBounds)

	if mcfg.Port == 0 {
		mcfg.Port = 13803
	}

	m, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", mcfg.Port))
	if err != nil {
		return nil
	}

	localAddr := getNodeLocalAddr()
	lm, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%v/tcp/%d", localAddr, mcfg.Port))
	logger.Info("NewMulti", "addr", m.String(), "laddr", lm.String())
	var addrlist []multiaddr.Multiaddr
	addrlist = append(addrlist, m)
	addrlist = append(addrlist, lm)
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
		libp2p.ListenAddrs(addrlist...),
		libp2p.Identity(priv),
		libp2p.BandwidthReporter(bandwidthTracker),
		libp2p.NATPortMap(),
	)

	if err != nil {
		panic(err)
	}
	p2p := &P2P{host: host}
	p2p.connManag = manage.NewConnManager(p2p.host)
	p2p.peerInfoManag = manage.NewPeerInfoManager()
	p2p.chainCfg = cfg
	p2p.discovery = new(Discovery)
	p2p.Node = NewNode(p2p, cfg)
	p2p.bandwidthTracker = bandwidthTracker
	logger.Info("NewP2p", "peerId", p2p.host.ID(), "addrs", p2p.host.Addrs())
	return p2p
}

func (p *P2P) managePeers() {

	go p.connManag.MonitorAllPeers(p.Node.p2pCfg.Seeds, p.host)
	peerChan, err := p.discovery.FindPeers(context.Background(), p.host, p.Node.p2pCfg.Seeds)
	if err != nil {
		panic("PeerFind Err")
	}

	for peer := range peerChan {
		logger.Info("find peer", "peer", peer)
		if peer.ID.Pretty() == p.host.ID().Pretty() {
			logger.Info("Find self...", "ID", p.host.ID())
			continue
		}
		logger.Info("+++++++++++++++++++++++++++++p2p.FindPeers", "addrs", peer.Addrs, "id", peer.ID.String(),
			"peer", peer.String())

		logger.Info("All Peers", "PeersWithAddrs", p.host.Peerstore().PeersWithAddrs())
		p.newConn(context.Background(), peer)
	Recheck:
		if p.connManag.Size() >= 25 {
			//达到连接节点数最大要求
			time.Sleep(time.Second * 10)
			goto Recheck
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
	globalData := &prototypes.GlobalData{
		ChainCfg:        p.chainCfg,
		QueueClient:     p.client,
		Host:            p.host,
		ConnManager:     p.connManag,
		PeerInfoManager: p.peerInfoManag,
	}
	protocol.Init(globalData)
	go p.managePeers()
	go p.processP2P()
	go p.showBandwidthTracker()

}
func (p *P2P) showBandwidthTracker() {
	for {
		logger.Info("------------BandTracker--------------")
		bandByPeer := p.bandwidthTracker.GetBandwidthByPeer()
		for pid, stat := range bandByPeer {
			logger.Info("BandwidthTracker", "pid", pid, "RateIn bytes/seconds", stat.RateIn, "RateOut  bytes/seconds", stat.RateOut,
				"TotalIn", stat.TotalIn, "TotalOut", stat.TotalOut)
		}
		logger.Info("-------------------------------------")
		time.Sleep(time.Second * 10)

	}
}
func (p *P2P) processP2P() {

	p.client.Sub("p2p")

	//TODO, control goroutine num
	for msg := range p.client.Recv() {
		go protocol.HandleEvent(msg)
	}
}

func (p *P2P) newConn(ctx context.Context, pr peer.AddrInfo) error {

	err := p.host.Connect(context.Background(), pr)
	if err != nil {
		logger.Error("newConn", "Connect err", err, "remoteID", pr.ID)
		return err
	}
	p.connManag.Add(pr, peerstore.RecentlyConnectedAddrTTL)

	return nil

}
