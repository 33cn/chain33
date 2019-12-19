package p2pnext

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/33cn/chain33/client"
	l "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/metrics"
	"github.com/libp2p/go-libp2p-core/peer"
	host "github.com/libp2p/go-libp2p-host"
	multiaddr "github.com/multiformats/go-multiaddr"
)

var logger = l.New("module", "p2pnext")

type P2p struct {
	Host       host.Host
	discovery  *Discovery
	streamMang *StreamMange
	api        client.QueueProtocolAPI
	client     queue.Client
	Done       chan struct{}
	Node       *Node
	Processer  map[string]Driver
}

func New(cfg *types.Chain33Config) *P2p {

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
		//libp2p.EnableAutoRelay(),
		libp2p.BandwidthReporter(bandwidthTracker),
		libp2p.NATPortMap(),
	)

	if err != nil {
		panic(err)
	}
	p2p := &P2p{Host: host}
	p2p.streamMang = NewStreamManage(host)
	p2p.Processer = make(map[string]Driver)
	p2p.discovery = new(Discovery)
	p2p.Node = NewNode(p2p, cfg)

	logger.Info("NewP2p", "peerId", p2p.Host.ID(), "addrs", p2p.Host.Addrs())
	return p2p

}

func (p *P2p) managePeers() {
	for _, seed := range p.Node.p2pCfg.Seeds {
		addr, _ := multiaddr.NewMultiaddr(seed)
		peerinfo, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			panic(err)
		}

		err = p.Host.Connect(context.Background(), *peerinfo)
		if err != nil {
			logger.Error("Host Connect", "err", err)
			return
		}

		logger.Info("xxx", "pid:", peerinfo.ID, "addr", peerinfo.Addrs)

		_, err = p.streamMang.newStream(context.Background(), *peerinfo)
		if err != nil {
			logger.Error("xxxnewStream", "err", err)
			return
		}
	}
	peerChan, err := p.discovery.FindPeers(context.Background(), p.Host)
	if err != nil {
		panic("PeerFind Err")
	}

	for {
		select {
		case <-p.Done:
			return
		case peer := <-peerChan:
			if len(peer.Addrs) == 0 {
				continue
			}
			if peer.ID == p.Host.ID() {
				break
			}

			if peer.ID == p.Host.ID() {
				logger.Info("Find self...")
				continue
			}
			logger.Info("p2p.FindPeers", "addrs", peer.Addrs, "id", peer.ID.String(),
				"peer", peer.String())
			p.streamMang.newStream(context.Background(), peer)
		Recheck:
			if p.streamMang.Size() >= 25 {
				//达到连接节点数最大要求
				time.Sleep(time.Second * 10)
				goto Recheck
			}

		case <-p.Done:
			return

		}
	}

}
func (p *P2p) Wait() {

}

func (p *P2p) Close() {
	p.client.Close()
	close(p.Done)
	logger.Info("p2p closed")

	return
}

func (p *P2p) initProcesser() {
	for name, driver := range drivers {
		process := driver.New(p.Node, p.client, p.Done)
		p.Processer[name] = process

	}

}

// SetQueueClient
func (p *P2p) SetQueueClient(cli queue.Client) {
	var err error
	p.api, err = client.New(cli, nil)
	if err != nil {
		panic("SetQueueClient client.New err")
	}
	if p.client == nil {
		p.client = cli
	}
	p.initProcesser()
	go p.managePeers()
	go p.subP2PMsg()

}

func (p *P2p) subP2PMsg() {

	if p.client == nil {
		return
	}
	p.client.Sub("p2p")

	for msg := range p.client.Recv() {
		switch msg.Ty {
		case types.EventTxBroadcast, types.EventBlockBroadcast:
			//go p.Processer[BroadCast].DoProcess(msg)
		case types.EventPeerInfo:
			logger.Info("subP2PMsg", "Receive PeerInfo Request")
			go p.Processer[PeerInfo].DoProcess(msg)
		case types.EventFetchBlockHeaders:
			go p.Processer[Header].DoProcess(msg)
		case types.EventGetNetInfo:
			go p.Processer[NetInfo].DoProcess(msg)
		case types.EventFetchBlocks:
			go p.Processer[Download].DoProcess(msg)

		}
	}
}
