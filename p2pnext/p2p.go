package p2pnext

import (
	"context"
	"encoding/hex"
	"github.com/33cn/chain33/p2pnext/manage"
	"github.com/33cn/chain33/p2pnext/protocol"
	prototypes "github.com/33cn/chain33/p2pnext/protocol/types"
	core "github.com/libp2p/go-libp2p-core"
	"time"

	"github.com/33cn/chain33/client"
	l "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/metrics"
	"github.com/libp2p/go-libp2p-core/peer"
	multiaddr "github.com/multiformats/go-multiaddr"
)

var logger = l.New("module", "p2pnext")

type P2P struct {
	chainCfg      *types.Chain33Config
	host          core.Host
	discovery     *Discovery
	streamManag   *manage.StreamManager
	peerInfoManag *manage.PeerInfoManager
	api           client.QueueProtocolAPI
	client        queue.Client
	Done          chan struct{}
	Node          *Node
	Processer     map[string]Driver
}

func New(cfg *types.Chain33Config) *P2P {

	m, err := multiaddr.NewMultiaddr("/ip4/0.0.0.0/tcp/13803")
	if err != nil {
		return nil
	}
	var addrlist []multiaddr.Multiaddr
	addrlist = append(addrlist, m)
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
		//libp2p.EnableAutoRelay(),
		libp2p.BandwidthReporter(bandwidthTracker),
		libp2p.NATPortMap(),
	)

	if err != nil {
		panic(err)
	}
	p2p := &P2P{host: host}
	p2p.streamManag = manage.NewStreamManager()
	p2p.peerInfoManag = manage.NewPeerInfoManager()
	p2p.chainCfg = cfg
	p2p.Processer = make(map[string]Driver)
	p2p.discovery = new(Discovery)
	p2p.Node = NewNode(p2p, cfg)


	logger.Info("NewP2p", "peerId", p2p.host.ID(), "addrs", p2p.host.Addrs())
	return p2p

}

func (p *P2P) managePeers() {
	for _, seed := range p.Node.p2pCfg.Seeds {
		addr, _ := multiaddr.NewMultiaddr(seed)

		peerinfo, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			panic(err)
		}
		logger.Info("xxx", "pid:", peerinfo.ID, "addr", peerinfo.Addrs)
		_, err = p.newStream(context.Background(), *peerinfo)
		logger.Error(err.Error())
		//p.Host.Connect(context.Background(), peerinfo)
	}
	peerChan, err := p.discovery.FindPeers(context.Background(), p.host)
	if err != nil {
		panic("PeerFind Err")
	}
	for {
		select {
		case peer := <-peerChan:
			if len(peer.Addrs) == 0 {
				continue
			}
			if peer.ID == p.host.ID() {
				break
			}

			if peer.ID == p.host.ID() {
				logger.Info("Find self...")
				continue
			}
			logger.Info("p2p.managePeers", "addrs", peer.Addrs, "id", peer.ID.String(),
				"peer", peer.String())
			p.newStream(context.Background(), peer)
		Recheck:
			if p.streamManag.Size() >= 25 {
				//达到连接节点数最大要求
				time.Sleep(time.Second * 10)
				goto Recheck
			}

		case <-p.Done:
			return

		}
	}

}
func (p *P2P) Wait() {

}

func (p *P2P) Close() {
	close(p.Done)
}

func (p *P2P) initProcesser() {

	for _, name := range ProcessName {
		driver, err := NewDriver(name)
		if err != nil {
			return
		}
		process := driver.New(p.Node, p.client, p.Done)
		p.Processer[name] = process
	}

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
		ChainCfg:        nil,
		QueueClient:     nil,
		Host:            nil,
		StreamManager:   nil,
		PeerInfoManager: nil,
	}
	protocol.Init(globalData)
	p.initProcesser()
	go p.managePeers()
	//go p.subP2PMsg()
	go p.processP2P()

}

//func (p *P2P) subP2PMsg() {
//	if p.client == nil {
//		return
//	}
//
//	for msg := range p.client.Recv() {
//		switch msg.Ty {
//		case types.EventTxBroadcast, types.EventBlockBroadcast:
//			go p.Processer[BroadCast].DoProcess(msg)
//		case types.EventPeerInfo:
//			go p.Processer[PeerInfo].DoProcess(msg)
//		case types.EventFetchBlockHeaders:
//			go p.Processer[Header].DoProcess(msg)
//		case types.EventGetNetInfo:
//			go p.Processer[NetInfo].DoProcess(msg)
//		case types.EventFetchBlocks:
//			go p.Processer[Download].DoProcess(msg)
//
//		}
//	}
//}

func (p *P2P) processP2P() {

	if p.client == nil {
		return
	}
	//TODO, control goroutine num
	for msg := range p.client.Recv() {

	   go protocol.HandleEvent(msg)
	}
}



func (p *P2P) newStream(ctx context.Context, pr peer.AddrInfo) (core.Stream, error) {

	//可以后续添加 block.ID,mempool.ID,header.ID

	stream, err := p.host.NewStream(ctx, pr.ID)
	if err != nil {
		return nil, err
	}

	p.streamManag.AddStream(string(pr.ID), stream)
	return stream, nil

}
