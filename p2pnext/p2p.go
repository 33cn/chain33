package p2pnext

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
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

	"github.com/libp2p/go-libp2p-core/metrics"

	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"

	multiaddr "github.com/multiformats/go-multiaddr"
)

var log = l.New("module", "p2pnext")

type P2P struct {
	chainCfg      *types.Chain33Config
	host          core.Host
	discovery     *dht.Discovery
	connManag     *manage.ConnManager
	peerInfoManag *manage.PeerInfoManager
	api           client.QueueProtocolAPI
	client        queue.Client
	addrbook      *AddrBook
	taskGroup     *sync.WaitGroup

	closed int32
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

	log.Info("NewMulti", "addr", m.String())
	addrbook := NewAddrBook(cfg.GetModuleConfig().P2P)
	priv := addrbook.GetPrivkey()

	bandwidthTracker := metrics.NewBandwidthCounter()
	host := newHost(cfg.GetModuleConfig().P2P, priv, bandwidthTracker)
	p2p := &P2P{host: host}
	p2p.peerInfoManag = manage.NewPeerInfoManager()
	p2p.chainCfg = cfg
	p2p.addrbook = addrbook
	p2p.discovery = new(dht.Discovery)
	p2p.discovery.InitDht(p2p.host, cfg.GetModuleConfig().P2P.Seeds, p2p.addrbook.AddrsInfo())
	p2p.connManag = manage.NewConnManager(p2p.host, p2p.discovery, bandwidthTracker)
	p2p.taskGroup = &sync.WaitGroup{}

	log.Info("NewP2p", "peerId", p2p.host.ID(), "addrs", p2p.host.Addrs())
	return p2p
}

func newHost(cfg *types.P2P, priv p2pcrypto.PrivKey, bandwidthTracker *metrics.BandwidthCounter) core.Host {
	m, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", cfg.Port))
	if err != nil {
		return nil
	}
	log.Info("NewMulti", "addr", m.String())
	if bandwidthTracker == nil {
		bandwidthTracker = metrics.NewBandwidthCounter()
	}

	host, err := libp2p.New(context.Background(),
		libp2p.ListenAddrs(m),
		libp2p.Identity(priv),
		libp2p.BandwidthReporter(bandwidthTracker),
		libp2p.NATPortMap(),
	)

	if err != nil {
		panic(err)
	}

	return host
}

func (p *P2P) managePeers() {
	go p.connManag.MonitorAllPeers(p.chainCfg.GetModuleConfig().P2P.Seeds, p.host)

	for {
		peerlist := p.discovery.RoutingTale()
		log.Info("managePeers", "RoutingTale show peerlist>>>>>>>>>", peerlist,
			"table size", p.discovery.RoutingTableSize())
		if p.isClose() {
			log.Info("managePeers", "p2p", "closed")

			return
		}
		select {
		case <-time.After(time.Minute * 10):
			//Reflesh addrbook
			peersInfo := p.discovery.FindLocalPeers(p.connManag.Fetch())
			p.addrbook.SaveAddr(peersInfo)

		}
	}

}

// SetQueueClient
func (p *P2P) SetQueueClient(cli queue.Client) {
	var err error
	p.api, err = client.New(cli, nil)
	if err != nil {
		//panic("SetQueueClient client.New err")
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
		if p.isClose() {
			return
		}
		p.taskGroup.Add(1)
		go func() {
			defer p.taskGroup.Done()
			protocol.HandleEvent(msg)

		}()
	}
}

func (p *P2P) Wait() {

}

func (p *P2P) ReStart() {
	client := p.client

	p.Close()
	p = New(p.chainCfg)
	p.SetQueueClient(client)

}

func (p *P2P) Close() {
	log.Info("p2p closed")
	atomic.StoreInt32(&p.closed, 1)
	p.waitTaskDone()
	p.connManag.Close()
	p.host.Close()
	prototypes.ClearEventHandler()

	return
}

func (p *P2P) isClose() bool {
	return atomic.LoadInt32(&p.closed) == 1
}

func (p *P2P) waitTaskDone() {

	waitDone := make(chan struct{})
	go func() {
		defer close(waitDone)
		p.taskGroup.Wait()
	}()
	select {
	case <-waitDone:
	case <-time.After(time.Second * 20):
		log.Error("waitTaskDone", "err", "20s timeout")
	}
}
