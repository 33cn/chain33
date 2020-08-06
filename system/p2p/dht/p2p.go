// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package dht 基于libp2p实现p2p 插件
package dht

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/rand"
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
	restart int32
	p2pCfg  *types.P2P
	subCfg  *p2pty.P2PSubConfig
	mgr     *p2p.Manager
	subChan chan interface{}
	ctx     context.Context
	cancel  context.CancelFunc

	db ds.Datastore
	//env *protocol.P2PEnv
	env *prototypes.P2PEnv
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
	client := mgr.Client
	return &P2P{
		client:   client,
		chainCfg: chainCfg,
		subCfg:   mcfg,
		p2pCfg:   p2pCfg,
		mgr:      mgr,
		api:      mgr.SysAPI,
		addrbook: NewAddrBook(p2pCfg),
		subChan:  mgr.PubSub.Sub(p2pty.DHTTypeName),
	}

}

// StartP2P start p2p
func (p *P2P) StartP2P() {
	priv := p.addrbook.GetPrivkey()
	bandwidthTracker := metrics.NewBandwidthCounter()
	maddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", p.subCfg.Port))
	if err != nil {
		panic(err)
	}
	log.Info("NewMulti", "addr", maddr.String())
	host := newHost(p.subCfg, priv, bandwidthTracker, maddr)
	p.host = host
	p.peerInfoManag = manage.NewPeerInfoManager(p.client)
	p.taskGroup = &sync.WaitGroup{}
	p.db = store.NewDataStore(p.subCfg)

	p.ctx, p.cancel = context.WithCancel(context.Background())
	atomic.StoreInt32(&p.restart, 0)
	p.discovery = net.InitDhtDiscovery(p.ctx, p.host, p.addrbook.AddrsInfo(), p.chainCfg, p.subCfg)
	p.connManag = manage.NewConnManager(p.host, p.discovery, bandwidthTracker, p.subCfg)
	pubsub, err := net.NewPubSub(p.ctx, p.host)
	if err != nil {
		return
	}

	p.pubsub = pubsub
	p.addrbook.StoreHostID(p.host.ID(), p.p2pCfg.DbPath)

	log.Info("NewP2p", "peerId", p.host.ID(), "addrs", p.host.Addrs())

	//提供给其他插件使用的共享接口
	env := &prototypes.P2PEnv{
		ChainCfg:         p.chainCfg,
		QueueClient:      p.client,
		Host:             p.host,
		ConnManager:      p.connManag,
		PeerInfoManager:  p.peerInfoManag,
		P2PManager:       p.mgr,
		SubConfig:        p.subCfg,
		Discovery:        p.discovery,
		Pubsub:           p.pubsub,
		Ctx:              p.ctx,
		DB:               p.db,
		RoutingTable:     p.discovery.RoutingTable(),
		RoutingDiscovery: p.discovery.RoutingDiscovery,
	}
	protocol.Init(env)
	protocol.InitAllProtocol(env)
	p.env = env
	go p.managePeers()
	go p.handleP2PEvent()
	go p.findLANPeers()
	//TODO 考虑增加配置，选择是否创建空投地址
	go p.genAirDropKey()

}

// CloseP2P close p2p
func (p *P2P) CloseP2P() {
	log.Info("p2p closing")
	//p.mgr.PubSub.Unsub(p.subChan)
	p.cancel()
	p.connManag.Close()
	p.peerInfoManag.Close()
	p.discovery.CloseDht()
	p.waitTaskDone()
	p.db.Close()
	p.host.Close()
	protocol.ClearEventHandler()
	prototypes.ClearEventHandler()
	if !p.isRestart() {
		p.mgr.PubSub.Unsub(p.subChan)

	}
	log.Info("p2p closed")

}

func (p *P2P) reStart() {
	atomic.StoreInt32(&p.restart, 1)
	log.Info("p2p will restart")
	p.CloseP2P()
	p.StartP2P()

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
		select {
		case <-p.ctx.Done():
			log.Info("managePeers", "p2p", "closed")
			return
		case <-time.After(time.Minute * 10):
			//Reflesh addrbook
			peersInfo := p.discovery.FindLocalPeers(p.connManag.FetchNearestPeers())
			if len(peersInfo) != 0 {
				p.addrbook.SaveAddr(peersInfo)
			}

		}

	}

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
			log.Info("findLANPeers", "process", "done")
			p.discovery.CloseFindLANPeers()
			return
		}
	}

}

func (p *P2P) handleP2PEvent() {

	//TODO, control goroutine num
	for {
		select {
		case <-p.ctx.Done():
			return
		case data := <-p.subChan:
			msg, ok := data.(*queue.Message)
			if !ok || data == nil {
				log.Error("handleP2PEvent", "recv invalid msg, data=", data)
				continue
			}

			p.taskGroup.Add(1)
			go func(qmsg *queue.Message) {
				defer p.taskGroup.Done()
				//log.Debug("handleP2PEvent", "recv msg ty", qmsg.Ty)
				protocol.HandleEvent(qmsg)

			}(msg)
		}

	}

}

func (p *P2P) isRestart() bool {
	return atomic.LoadInt32(&p.restart) == 1
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

//创建空投地址
func (p *P2P) genAirDropKey() {

	for { //等待钱包创建，解锁
		select {
		case <-p.ctx.Done():
			log.Info("genAirDropKey", "p2p closed")
			return
		case <-time.After(time.Second):

			resp, err := p.api.ExecWalletFunc("wallet", "GetWalletStatus", &types.ReqNil{})
			if err != nil {
				time.Sleep(time.Second)
				continue
			}
			if resp.(*types.WalletStatus).GetIsWalletLock() { //上锁状态，无法用助记词创建空投地址,等待...
				continue
			}

			if !resp.(*types.WalletStatus).GetIsHasSeed() {
				continue
			}

		}
		break
	}

	//用助记词和随机索引创建空投地址
	r := rand.New(rand.NewSource(types.Now().Unix()))
	var minIndex int32 = 100000000
	randIndex := minIndex + r.Int31n(1000000)
	reqIndex := &types.Int32{Data: randIndex}
	msg, err := p.api.ExecWalletFunc("wallet", "NewAccountByIndex", reqIndex)
	if err != nil {
		log.Error("genAirDropKey", "NewAccountByIndex err", err)
		return
	}

	var walletPrivkey string
	if reply, ok := msg.(*types.ReplyString); !ok {
		log.Error("genAirDropKey", "wrong format data", "")
		panic(err)

	} else {
		walletPrivkey = reply.GetData()
	}
	if walletPrivkey[:2] == "0x" {
		walletPrivkey = walletPrivkey[2:]
	}

	walletPubkey, err := GenPubkey(walletPrivkey)
	if err != nil {
		return
	}
	//如果addrbook之前保存的savePrivkey相同，则意味着节点启动之前已经创建了airdrop 空投地址
	savePrivkey, _ := p.addrbook.GetPrivPubKey()
	if savePrivkey == walletPrivkey { //addrbook与wallet保存了相同的空投私钥，不需要继续导入
		log.Debug("genAirDropKey", " process done")
		return
	}

	if len(savePrivkey) != 2*privKeyCompressBytesLen { //非压缩私钥,兼容之前老版本的DHT非压缩私钥
		log.Debug("len savePrivkey", len(savePrivkey))
		unCompkey := p.addrbook.GetPrivkey()
		if unCompkey == nil {
			savePrivkey = ""
		} else {
			compkey, err := unCompkey.Raw() //compress key
			if err != nil {
				savePrivkey = ""
				log.Error("genAirDropKey", "compressKey.Raw err", err)
			} else {
				savePrivkey = hex.EncodeToString(compkey)
			}
		}

	}

	if savePrivkey != "" {
		//savePrivkey是随机私钥，兼容老版本，先对其进行导入钱包处理
		//进行压缩处理
		var parm types.ReqWalletImportPrivkey
		parm.Privkey = savePrivkey
		parm.Label = "dht node award"

		for {
			_, err = p.api.ExecWalletFunc("wallet", "WalletImportPrivkey", &parm)
			if err == types.ErrLabelHasUsed {
				//切换随机lable
				parm.Label = fmt.Sprintf("node award %d", rand.Int31n(1024000))
				time.Sleep(time.Second)
				continue
			}
			break
		}
	}

	p.addrbook.saveKey(walletPrivkey, walletPubkey)
	p.reStart()

}
