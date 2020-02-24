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
	chainCfg      *types.Chain33Config
	host          core.Host
	discovery     *Discovery
	connManag     *manage.ConnManager
	peerInfoManag *manage.PeerInfoManager
	api           client.QueueProtocolAPI
	client        queue.Client
	Done          chan struct{}
	Node          *Node
}

func New(cfg *types.Chain33Config) *P2P {

	m, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", cfg.GetModuleConfig().P2P.Port))
	if err != nil {
		return nil
	}
	localAddr := getNodeLocalAddr()
	lm, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%v/tcp/%d", localAddr, cfg.GetModuleConfig().P2P.Port))
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
	p2p.connManag = manage.NewConnManager()
	p2p.peerInfoManag = manage.NewPeerInfoManager()
	p2p.chainCfg = cfg
	p2p.discovery = new(Discovery)
	p2p.Node = NewNode(p2p, cfg)

	logger.Info("NewP2p", "peerId", p2p.host.ID(), "addrs", p2p.host.Addrs())
	return p2p

}

func (p *P2P) managePeers() {

	peerChan, err := p.discovery.FindPeers(context.Background(), p.host, p.Node.p2pCfg.Seeds)
	if err != nil {
		panic("PeerFind Err")
	}

	for peer := range peerChan {
		logger.Info("find peer", "peer", peer)
		if peer.ID == p.host.ID() {
			logger.Info("Find self...", p.host.ID(), "")
			continue
		}
		logger.Info("+++++++++++++++++++++++++++++p2p.FindPeers", "addrs", peer.Addrs, "id", peer.ID.String(),
			"peer", peer.String())

		p.host.Peerstore().AddAddrs(peer.ID, peer.Addrs, peerstore.AddressTTL)

		peerstore := p.host.Peerstore()

		logger.Info("xxxxxxxxxxxxxxxxxxxAll Peers", peerstore.Peers(), "")
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

}

func (p *P2P) processP2P() {

	p.client.Sub("p2p")

	//TODO, control goroutine num
	for msg := range p.client.Recv() {
		go protocol.HandleEvent(msg)
	}
}

func (p *P2P) newConn(ctx context.Context, pr peer.AddrInfo) error {

	//可以后续添加 block.ID,mempool.ID,header.ID

	err = p.host.Connect(context.Background(), *peerinfo)

	//logger.Info("newStream", "MsgIds size", len(protocol.MsgIDs), "msgIds", protocol.MsgIDs)
	//stream, err := p.host.NewStream(ctx, pr.ID, protocol.MsgIDs...)
	if err != nil {
		return err
	}
	//defer stream.Close()
	p.host.ConnManager().TagPeer(pr.ID, "chain33", 1)
	p.host.ConnManager().Protect(pr.ID, "chain33")
	logger.Info("NewStream", "Pid", pr.ID)
	p.connManag.Add(pr.ID.String(), nil)
	return nil

}
