// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package dht 基于libp2p实现p2p 插件
package dht

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/33cn/chain33/p2p"

	"time"

	"github.com/33cn/chain33/client"
	logger "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/system/p2p/dht/manage"
	"github.com/33cn/chain33/system/p2p/dht/net"
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	prototypes "github.com/33cn/chain33/system/p2p/dht/protocol/types"
	p2pty "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	libp2p "github.com/libp2p/go-libp2p"
	core "github.com/libp2p/go-libp2p-core"

	connmgr "github.com/libp2p/go-libp2p-connmgr"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/metrics"
	multiaddr "github.com/multiformats/go-multiaddr"
)

var log = logger.New("module", "p2pnext")

func init() {
	p2p.RegisterP2PCreate(p2pty.DHTTypeName, New)
}

// P2P p2p struct
type P2P struct {
	chainCfg      *types.Chain33Config
	host          core.Host
	discovery     *net.Discovery
	connManag     *manage.ConnManager
	peerInfoManag *manage.PeerInfoManager
	api           client.QueueProtocolAPI
	client        queue.Client
	addrbook      *AddrBook
	taskGroup     *sync.WaitGroup

	closed  int32
	p2pCfg  *types.P2P
	subCfg  *p2pty.P2PSubConfig
	mgr     *p2p.Manager
	subChan chan interface{}
}

// New new dht p2p network
func New(mgr *p2p.Manager, subCfg []byte) p2p.IP2P {

	chainCfg := mgr.ChainCfg
	p2pCfg := chainCfg.GetModuleConfig().P2P
	mcfg := &p2pty.P2PSubConfig{}
	types.MustDecode(subCfg, mcfg)
	if mcfg.Port == 0 {
		mcfg.Port = p2pty.DefaultP2PPort
	}

	addrbook := NewAddrBook(p2pCfg)
	priv := addrbook.GetPrivkey()

	bandwidthTracker := metrics.NewBandwidthCounter()
	host := newHost(mcfg.Port, priv, bandwidthTracker, int(mcfg.MaxConnectNum))
	p2p := &P2P{
		host:          host,
		peerInfoManag: manage.NewPeerInfoManager(mgr.Client),
		chainCfg:      chainCfg,
		subCfg:        mcfg,
		p2pCfg:        p2pCfg,
		client:        mgr.Client,
		api:           mgr.SysAPI,
		addrbook:      addrbook,
		mgr:           mgr,
		taskGroup:     &sync.WaitGroup{},
	}
	p2p.subChan = p2p.mgr.PubSub.Sub(p2pty.DHTTypeName)
	p2p.discovery = net.InitDhtDiscovery(p2p.host, p2p.addrbook.AddrsInfo(), p2p.chainCfg, p2p.subCfg)
	p2p.connManag = manage.NewConnManager(p2p.host, p2p.discovery, bandwidthTracker, p2p.subCfg)
	p2p.addrbook.StoreHostID(p2p.host.ID(), p2pCfg.DbPath)
	log.Info("NewP2p", "peerId", p2p.host.ID(), "addrs", p2p.host.Addrs())

	return p2p
}

func newHost(port int32, priv p2pcrypto.PrivKey, bandwidthTracker metrics.Reporter, maxconnect int) core.Host {
	m, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port))
	if err != nil {
		return nil
	}
	log.Info("NewMulti", "addr", m.String())
	if bandwidthTracker == nil {
		bandwidthTracker = metrics.NewBandwidthCounter()
	}
	if maxconnect <= 0 {
		maxconnect = 100
	}
	host, err := libp2p.New(context.Background(),
		libp2p.ListenAddrs(m),
		libp2p.Identity(priv),
		libp2p.BandwidthReporter(bandwidthTracker),
		libp2p.NATPortMap(),

		//connmgr 默认连接100个，最少3/4*maxconnect个
		libp2p.ConnectionManager(connmgr.NewConnManager(maxconnect*3/4, maxconnect, 0)),
	)
	if err != nil {
		panic(err)
	}

	return host
}

func (p *P2P) managePeers() {
	go p.connManag.MonitorAllPeers(p.subCfg.Seeds, p.host)

	for {
		peerlist := p.discovery.RoutingTale()
		log.Debug("managePeers", "RoutingTale show peerlist>>>>>>>>>", peerlist,
			"table size", p.discovery.RoutingTableSize())
		if p.isClose() {
			log.Info("managePeers", "p2p", "closed")

			return
		}
		<-time.After(time.Minute * 10)
		//Reflesh addrbook
		peersInfo := p.discovery.FindLocalPeers(p.connManag.FetchNearestPeers())
		p.addrbook.SaveAddr(peersInfo)
	}

}

func (p *P2P) StartP2P() {

	//提供给其他插件使用的共享接口
	env := &prototypes.P2PEnv{
		ChainCfg:        p.chainCfg,
		QueueClient:     p.client,
		Host:            p.host,
		ConnManager:     p.connManag,
		Discovery:       p.discovery,
		PeerInfoManager: p.peerInfoManag,
		P2PManager:      p.mgr,
		SubConfig:       p.subCfg,
	}
	protocol.Init(env)
	go p.managePeers()
	go p.handleP2PEvent()
	go p.findLANPeers()

}

//查询本局域网内是否有节点
func (p *P2P) findLANPeers() {
	peerChan, err := p.discovery.FindLANPeers(p.host, fmt.Sprintf("/%s-mdns/%d", p.chainCfg.GetTitle(), p.subCfg.Channel))
	if err != nil {
		log.Error("findLANPeers", "err", err.Error())
		return
	}

	for neighbors := range peerChan {
		log.Info(">>>>>>>>>>>>>>>>>>>^_^! Well,Let's Play ^_^!<<<<<<<<<<<<<<<<<<<<<<<<<<")
		//发现局域网内的邻居节点
		err := p.host.Connect(context.Background(), neighbors)
		if err != nil {
			log.Error("findLANPeers", "err", err.Error())
			continue
		}

		p.connManag.AddNeighbors(&neighbors)

	}
}

func (p *P2P) handleP2PEvent() {

	//TODO, control goroutine num
	for data := range p.subChan {
		if p.isClose() {
			return
		}
		msg, ok := data.(*queue.Message)
		if !ok {
			log.Error("handleP2PEvent", "recv invalid msg, data=", data)
		}
		p.taskGroup.Add(1)
		go func(qmsg *queue.Message) {
			defer p.taskGroup.Done()
			log.Debug("handleP2PEvent", "recv msg ty", qmsg.Ty)
			protocol.HandleEvent(qmsg)

		}(msg)

	}
}

func (p *P2P) CloseP2P() {
	log.Info("p2p closed")
	p.mgr.PubSub.Unsub(p.subChan)
	atomic.StoreInt32(&p.closed, 1)
	p.waitTaskDone()
	p.connManag.Close()
	p.peerInfoManag.Close()
	p.host.Close()
	prototypes.ClearEventHandler()
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
