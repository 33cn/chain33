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
	"time"

	core "github.com/libp2p/go-libp2p-core"

	"github.com/33cn/chain33/client"
	logger "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/p2p"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/system/p2p/dht/manage"
	"github.com/33cn/chain33/system/p2p/dht/net"
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	prototypes "github.com/33cn/chain33/system/p2p/dht/protocol/types"
	"github.com/33cn/chain33/system/p2p/dht/store"
	p2pty "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	ds "github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p"
	circuit "github.com/libp2p/go-libp2p-circuit"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	p2pcrypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/metrics"
	"github.com/multiformats/go-multiaddr"
)

var log = logger.New("module", p2pty.DHTTypeName)

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

	pubsub  *net.PubSub
	closed  int32
	p2pCfg  *types.P2P
	subCfg  *p2pty.P2PSubConfig
	mgr     *p2p.Manager
	subChan chan interface{}
	ctx     context.Context
	cancel  context.CancelFunc

	db  ds.Datastore
	env *protocol.P2PEnv
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

	maddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", mcfg.Port))
	if err != nil {
		panic(err)
	}
	log.Info("NewMulti", "addr", maddr.String())
	host := newHost(mcfg, priv, bandwidthTracker, maddr)

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
		db:            store.NewDataStore(mcfg),
	}
	p2p.ctx, p2p.cancel = context.WithCancel(context.Background())

	p2p.subChan = p2p.mgr.PubSub.Sub(p2pty.DHTTypeName)
	p2p.discovery = net.InitDhtDiscovery(p2p.ctx, p2p.host, p2p.addrbook.AddrsInfo(), p2p.chainCfg, p2p.subCfg)
	p2p.connManag = manage.NewConnManager(p2p.host, p2p.discovery, bandwidthTracker, p2p.subCfg)

	pubsub, err := net.NewPubSub(p2p.ctx, p2p.host)
	if err != nil {
		return nil
	}

	p2p.pubsub = pubsub
	p2p.addrbook.StoreHostID(p2p.host.ID(), p2pCfg.DbPath)
	log.Info("NewP2p", "peerId", p2p.host.ID(), "addrs", p2p.host.Addrs())
	return p2p
}

func newHost(cfg *p2pty.P2PSubConfig, priv p2pcrypto.PrivKey, bandwidthTracker metrics.Reporter, maddr multiaddr.Multiaddr) core.Host {
	if bandwidthTracker == nil {
		bandwidthTracker = metrics.NewBandwidthCounter()
	}

	var relayOpt = make([]circuit.RelayOpt, 3)
	if cfg.RelayActive {
		relayOpt = append(relayOpt, circuit.OptActive)
	}
	if cfg.RelayHop {
		relayOpt = append(relayOpt, circuit.OptHop)
	}
	if cfg.RelayDiscovery {
		relayOpt = append(relayOpt, circuit.OptDiscovery)
	}

	var options []libp2p.Option
	if len(relayOpt) != 0 {
		options = append(options, libp2p.EnableRelay(relayOpt...))

	}

	options = append(options, libp2p.NATPortMap())
	if maddr != nil {
		options = append(options, libp2p.ListenAddrs(maddr))
	}
	if priv != nil {
		options = append(options, libp2p.Identity(priv))
	}
	if bandwidthTracker != nil {
		options = append(options, libp2p.BandwidthReporter(bandwidthTracker))
	}

	if cfg.MaxConnectNum > 0 { //如果不设置最大连接数量，默认允许dht自由连接并填充路由表
		var maxconnect = int(cfg.MaxConnectNum)
		options = append(options, libp2p.ConnectionManager(connmgr.NewConnManager(maxconnect*3/4, maxconnect, 0)))
	}

	host, err := libp2p.New(context.Background(), options...)
	if err != nil {
		panic(err)
	}

	return host
}

func (p *P2P) managePeers() {
	go p.connManag.MonitorAllPeers(p.subCfg.Seeds, p.host)

	for {
		log.Debug("managePeers", "table size", p.discovery.RoutingTableSize())
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

// StartP2P start p2p
func (p *P2P) StartP2P() {

	//提供给其他插件使用的共享接口
	env := &prototypes.P2PEnv{
		ChainCfg:        p.chainCfg,
		QueueClient:     p.client,
		Host:            p.host,
		ConnManager:     p.connManag,
		PeerInfoManager: p.peerInfoManag,
		P2PManager:      p.mgr,
		SubConfig:       p.subCfg,
		Discovery:       p.discovery,
		Pubsub:          p.pubsub,
		Ctx:             p.ctx,
		Cancel:          p.cancel,
	}
	protocol.Init(env)

	go p.managePeers()
	go p.handleP2PEvent()
	go p.findLANPeers()

	//debug new
	env2 := &protocol.P2PEnv{
		ChainCfg:         p.chainCfg,
		QueueClient:      p.client,
		Host:             p.host,
		P2PManager:       p.mgr,
		SubConfig:        p.subCfg,
		DB:               p.db,
		RoutingDiscovery: p.discovery.RoutingDiscovery,
		RoutingTable:     p.discovery.RoutingTable(),
	}
	p.env = env2
	protocol.InitAllProtocol(env2)
}

//查询本局域网内是否有节点
func (p *P2P) findLANPeers() {
	if p.subCfg.DisableFindLANPeers {
		return
	}
	peerChan, err := p.discovery.FindLANPeers(p.host, fmt.Sprintf("/%s-mdns/%d", p.chainCfg.GetTitle(), p.subCfg.Channel))
	if err != nil {
		log.Error("findLANPeers", "err", err.Error())
		return
	}

	for {
		select {
		case neighbors := <-peerChan:
			log.Info("^_^! Well,findLANPeers Let's Play ^_^!<<<<<<<<<<<<<<<<<<<", "peerName", neighbors.ID, "addrs:", neighbors.Addrs, "paddr", p.host.Peerstore().Addrs(neighbors.ID))
			//发现局域网内的邻居节点
			err := p.host.Connect(context.Background(), neighbors)
			if err != nil {
				log.Error("findLANPeers", "err", err.Error())
				continue
			}
			log.Info("findLANPeers", "connect neighbors success", neighbors.ID.Pretty())
			p.connManag.AddNeighbors(&neighbors)

		case <-p.ctx.Done():
			log.Warn("findLANPeers", "process", "done")
			return
		}
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
			//log.Debug("handleP2PEvent", "recv msg ty", qmsg.Ty)
			protocol.HandleEvent(qmsg)

		}(msg)

	}
}

// CloseP2P close p2p
func (p *P2P) CloseP2P() {
	log.Info("p2p closing")
	p.mgr.PubSub.Unsub(p.subChan)
	atomic.StoreInt32(&p.closed, 1)
	p.cancel()
	p.connManag.Close()
	p.peerInfoManag.Close()
	p.host.Close()
	p.waitTaskDone()
	protocol.ClearEventHandler()
	prototypes.ClearEventHandler()
	log.Info("p2p closed")
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
