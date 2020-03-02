package p2pnext

import (
	"context"
	"fmt"
	p2pty "github.com/33cn/chain33/p2pnext/types"
	"time"

	"github.com/33cn/chain33/client"
	logger "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/p2pnext/dht"
	"github.com/33cn/chain33/p2pnext/manage"

	"github.com/33cn/chain33/p2pnext/protocol"
	prototypes "github.com/33cn/chain33/p2pnext/protocol/types"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	libp2p "github.com/libp2p/go-libp2p"
	core "github.com/libp2p/go-libp2p-core"

	//"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/metrics"

	//"github.com/libp2p/go-libp2p-core/peer"
	//"github.com/libp2p/go-libp2p-core/peerstore"
	multiaddr "github.com/multiformats/go-multiaddr"
	p2pmgr "github.com/33cn/chain33/p2p/manage"
)

var log = logger.New("module", "p2pnext")

func init() {
	p2pmgr.RegisterP2PCreate(p2pmgr.DHTTypeName, New)
}



type P2P struct {
	chainCfg      *types.Chain33Config
	host          core.Host
	discovery     *dht.Discovery
	connManag     *manage.ConnManager
	peerInfoManag *manage.PeerInfoManager
	api           client.QueueProtocolAPI
	client        queue.Client
	Done          chan struct{}
	addrbook      *AddrBook
	Node          *Node
	p2pCfg   *types.P2P
	subCfg   *p2pty.P2PSubConfig
	mgr      *p2pmgr.P2PMgr
	subChan  chan interface{}
}

func New(mgr *p2pmgr.P2PMgr, subCfg []byte) p2pmgr.IP2P {

	chainCfg := mgr.ChainCfg
	p2pCfg := chainCfg.GetModuleConfig().P2P
	mcfg := &p2pty.P2PSubConfig{}
	types.MustDecode(subCfg, mcfg)
	if mcfg.Port == 0 {
		mcfg.Port = 13803
	}

	m, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", mcfg.Port))
	if err != nil {
		return nil
	}

	log.Info("NewMulti", "addr", m.String())
	addrbook := NewAddrBook(p2pCfg)
	priv := addrbook.GetPrivkey()

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
	p2p := &P2P{
		host : host,
		peerInfoManag: manage.NewPeerInfoManager(),
		chainCfg:chainCfg,
		subCfg: mcfg,
		p2pCfg:p2pCfg,
		client:mgr.Client,
		api: mgr.SysApi,
		discovery: &dht.Discovery{},
		addrbook:addrbook,
	}
	p2p.connManag = manage.NewConnManager(p2p.host, p2p.discovery, bandwidthTracker)
	p2p.Node = NewNode(p2p, mcfg)
	log.Info("NewP2p", "peerId", p2p.host.ID(), "addrs", p2p.host.Addrs())
	return p2p
}

func (p *P2P) managePeers() {
	p.discovery.InitDht(p.host, p.Node.subCfg.Seeds, p.addrbook.AddrsInfo())
	go p.connManag.MonitorAllPeers(p.Node.subCfg.Seeds, p.host)

	for {
		peerlist := p.discovery.RoutingTale()
		log.Info("managePeers", "RoutingTale show peerlist>>>>>>>>>", peerlist,
			"table size", p.discovery.RoutingTableSize())

		select {
		case <-time.After(time.Minute * 10):
			//Reflesh addrbook
			peersInfo := p.discovery.FindLocalPeers(p.connManag.Fetch())
			p.addrbook.SaveAddr(peersInfo)

		case <-p.Done:
			return
		}
	}

}

// SetQueueClient
func (p *P2P) StartP2P() {

	//提供给其他插件使用的共享接口
	globalData := &prototypes.GlobalData{
		ChainCfg:        p.chainCfg,
		QueueClient:     p.client,
		Host:            p.host,
		ConnManager:     p.connManag,
		Discovery:       p.discovery,
		PeerInfoManager: p.peerInfoManag,
		P2PManager:p.mgr,
		SubConfig:p.subCfg,
	}
	protocol.Init(globalData)
	go p.managePeers()
	go p.processP2P()

}

func (p *P2P) processP2P() {

	p.subChan = p.mgr.PubSub.Sub(p2pmgr.DHTTypeName)

	//TODO, control goroutine num
	for data := range p.subChan {
		msg, ok := data.(*queue.Message)

		if ok {
			log.Debug("processP2P", "recv msg ty", msg.Ty)
			go protocol.HandleEvent(msg)
		}else {
			log.Error("processP2P", "recv invalid msg, data=", data)
		}

	}
}

func (p *P2P) Wait() {

}

func (p *P2P) CloseP2P() {
	close(p.Done)
	p.mgr.PubSub.Unsub(p.subChan)
	log.Info("p2p closed")
	return
}
