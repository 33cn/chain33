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
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/common"
	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/p2p"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/system/p2p/dht/extension"
	"github.com/33cn/chain33/system/p2p/dht/manage"
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	p2pty "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	libp2pLog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/metrics"
	"github.com/libp2p/go-libp2p/core/peer"
	discovery "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/multiformats/go-multiaddr"
)

var log = log15.New("module", p2pty.DHTTypeName)

func init() {
	p2p.RegisterP2PCreate(p2pty.DHTTypeName, New)
}

// P2P p2p struct
type P2P struct {
	chainCfg        *types.Chain33Config
	host            core.Host
	discovery       *Discovery
	connManager     *manage.ConnManager
	peerInfoManager *manage.PeerInfoManager
	blackCache      *manage.TimeCache
	api             client.QueueProtocolAPI
	client          queue.Client
	addrBook        *AddrBook
	taskGroup       *sync.WaitGroup

	pubsub  *extension.PubSub
	restart int32
	p2pCfg  *types.P2P
	subCfg  *p2pty.P2PSubConfig
	mgr     *p2p.Manager
	subChan chan interface{}
	ctx     context.Context
	cancel  context.CancelFunc
	db      dbm.DB

	env *protocol.P2PEnv
}

func setLibp2pLog(logFile, logLevel string) {

	if logFile == "" {
		return
	}
	_, err := os.Stat(logFile)
	if os.IsNotExist(err) {
		err = os.MkdirAll(filepath.Dir(logFile), 0744)
	}
	if err != nil {
		log.Error("setLibp2pLog", "err", err)
		return
	}

	if logLevel == "" {
		logLevel = "ERROR"
	}

	libp2pLog.SetupLogging(libp2pLog.Config{
		Stderr: false,
		Stdout: false,
		File:   logFile,
	})

	err = libp2pLog.SetLogLevel("*", logLevel)
	if err != nil {
		log.Error("NewP2P", "set libp2p log level err", err)
	}
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
	// set libp2p log
	setLibp2pLog(mcfg.Libp2pLogFile, mcfg.Libp2pLogLevel)
	ctx, cancel := context.WithCancel(context.Background())
	p := &P2P{
		client:   mgr.Client,
		chainCfg: chainCfg,
		subCfg:   mcfg,
		p2pCfg:   p2pCfg,
		mgr:      mgr,
		api:      mgr.SysAPI,
		addrBook: NewAddrBook(p2pCfg),
		subChan:  mgr.PubSub.Sub(p2pty.DHTTypeName),
		ctx:      ctx,
		cancel:   cancel,
	}
	return initP2P(p)
}

func initP2P(p *P2P) *P2P {
	//other init work
	p.taskGroup = &sync.WaitGroup{}
	p.ctx, p.cancel = context.WithCancel(context.Background())
	priv := p.addrBook.GetPrivkey()
	if priv == nil { //addrbook存储的peer key 为空
		if p.p2pCfg.WaitPid { //p2p阻塞,直到创建钱包之后
			p.genAirDropKey()
		} else { //创建随机公私钥对,生成临时pid，待创建钱包之后，提换钱包生成的pid
			p.addrBook.Randkey()
			go p.genAirDropKey() //非阻塞模式
		}
	} else { //非阻塞模式
		go p.genAirDropKey()
	}

	maddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", p.subCfg.Port))
	if err != nil {
		panic(err)
	}
	log.Info("NewMulti", "addr", maddr.String())

	bandwidthTracker := metrics.NewBandwidthCounter()
	p.blackCache = manage.NewTimeCache(p.ctx, time.Minute*5)
	options := p.buildHostOptions(p.addrBook.GetPrivkey(), bandwidthTracker, maddr, p.blackCache)
	host, err := libp2p.New(options...)
	if err != nil {
		panic(err)
	}
	if p.subCfg.RelayServiceEnable {
		extension.MakeNodeRelayService(host, nil)
	}
	p.host = host
	ps, err := extension.NewPubSub(p.ctx, p.host, &p.subCfg.PubSub)
	if err != nil {
		return nil
	}
	p.pubsub = ps
	p.discovery = InitDhtDiscovery(p.ctx, p.host, p.addrBook.AddrsInfo(), p.chainCfg, p.subCfg)
	p.connManager = manage.NewConnManager(p.ctx, p.host, p.discovery.RoutingTable(), bandwidthTracker, p.subCfg)
	p.peerInfoManager = manage.NewPeerInfoManager(p.ctx, p.host, p.client)

	p.db = newDB("", p.p2pCfg.Driver, filepath.Dir(p.p2pCfg.DbPath), p.subCfg.DHTDataCache)
	return p
}

// StartP2P start p2p
func (p *P2P) StartP2P() {
	if atomic.LoadInt32(&p.restart) == 1 {
		log.Info("RestartP2P...")
		initP2P(p) //重新创建host
	}
	atomic.StoreInt32(&p.restart, 0)
	p.addrBook.StoreHostID(p.host.ID(), p.p2pCfg.DbPath)
	log.Info("NewP2p", "peerId", p.host.ID(), "addrs", p.host.Addrs())

	env := &protocol.P2PEnv{
		Ctx:             p.ctx,
		ChainCfg:        p.chainCfg,
		QueueClient:     p.client,
		Host:            p.host,
		P2PManager:      p.mgr,
		SubConfig:       p.subCfg,
		DB:              p.db,
		Discovery:       discovery.NewRoutingDiscovery(p.discovery.kademliaDHT),
		RoutingTable:    p.discovery.RoutingTable(),
		API:             p.api,
		Pubsub:          p.pubsub,
		PeerInfoManager: p.peerInfoManager,
		ConnManager:     p.connManager,
		ConnBlackList:   p.blackCache,
	}
	p.env = env
	protocol.InitAllProtocol(env)
	p.discovery.Start()
	go p.managePeers()
	go p.findLANPeers()
	for i := 0; i < runtime.NumCPU(); i++ {
		go p.handleP2PEvent()
	}
}

// CloseP2P close p2p
func (p *P2P) CloseP2P() {
	log.Info("p2p closing")
	p.cancel()
	p.waitTaskDone()
	p.db.Close()
	protocol.ClearEventHandler()
	if !p.isRestart() {
		p.mgr.PubSub.Shutdown()
	}
	p.host.Close()
	p.discovery.Close()

	log.Info("p2p closed")

}

func (p *P2P) reStart() {
	if p.host == nil {
		return
	}
	atomic.StoreInt32(&p.restart, 1)
	log.Info("reStart p2p")
	p.CloseP2P()
	p.StartP2P()
}

func (p *P2P) buildHostOptions(priv crypto.PrivKey, bandwidthTracker metrics.Reporter, maddr multiaddr.Multiaddr, timeCache *manage.TimeCache) []libp2p.Option {
	if bandwidthTracker == nil {
		bandwidthTracker = metrics.NewBandwidthCounter()
	}

	var options []libp2p.Option

	if p.subCfg.RelayEnable {
		options = append(options, libp2p.EnableRelay())
		//用配置的节点作为中继节点
		if len(p.subCfg.RelayNodeAddr) != 0 {
			relays := p.subCfg.RelayNodeAddr
			options = append(options, libp2p.AddrsFactory(extension.WithRelayAddrs(relays)))

		}

	}

	options = append(options, libp2p.NATPortMap())
	if maddr != nil {
		options = append(options, libp2p.ListenAddrs(maddr))
	}
	if priv != nil {
		options = append(options, libp2p.Identity(priv))
	}

	//enable private network,私有网络，拥有相同配置的节点才能连接进来。
	if p.subCfg.Psk != "" {
		psk, err := common.FromHex(p.subCfg.Psk)
		if err != nil {
			panic("set psk" + err.Error())
		}
		if len(psk) != 32 {
			panic("psk must 32 bytes")
		}
		options = append(options, libp2p.PrivateNetwork(psk))
	}

	options = append(options, libp2p.BandwidthReporter(bandwidthTracker))

	if p.subCfg.MaxConnectNum > 0 { //如果不设置最大连接数量，默认允许dht自由连接并填充路由表
		maxconnect := int(p.subCfg.MaxConnectNum)
		minconnect := 20
		if minconnect > maxconnect/2 {
			minconnect = maxconnect / 2
		}

		mgr, err := connmgr.NewConnManager(minconnect, maxconnect, connmgr.WithGracePeriod(time.Minute))
		if err != nil {
			panic("NewConnManager err:" + err.Error())
		}
		//1分钟的宽限期,定期清理
		options = append(options, libp2p.ConnectionManager(mgr))

	}
	//ConnectionGater,处理网络连接的策略
	options = append(options, libp2p.ConnectionGater(manage.NewConnGater(&p.host, p.subCfg.MaxConnectNum, timeCache, genAddrInfos(p.subCfg.WhitePeerList))))
	//关闭ping
	options = append(options, libp2p.Ping(false))
	return options
}

func (p *P2P) managePeers() {
	go p.connManager.MonitorAllPeers(p.taskGroup)
	p.taskGroup.Add(1)
	defer p.taskGroup.Done()
	for {
		log.Debug("managePeers", "table size", p.discovery.RoutingTable().Size())
		select {
		case <-p.ctx.Done():
			log.Info("managePeers", "p2p", "closed")
			return
		case <-time.After(time.Minute * 10):
			//Refresh addr book
			var peersInfo []peer.AddrInfo
			for _, pid := range p.connManager.FetchNearestPeers(50) {
				info := p.discovery.FindLocalPeer(pid)
				if len(info.Addrs) != 0 {
					peersInfo = append(peersInfo, info)
				}
			}
			if len(peersInfo) != 0 {
				_ = p.addrBook.SaveAddr(peersInfo)
			}
		}
	}
}

// 查询本局域网内是否有节点
func (p *P2P) findLANPeers() {
	if p.subCfg.DisableFindLANPeers {
		return
	}
	peerChan, err := p.discovery.FindLANPeers(p.host, fmt.Sprintf("/%s-mdns/%d", p.chainCfg.GetTitle(), p.subCfg.Channel))
	if err != nil {
		log.Error("findLANPeers", "err", err.Error())
		return
	}

	p.taskGroup.Add(1)
	defer p.taskGroup.Done()

	for {
		select {
		case neighbors := <-peerChan:
			log.Debug("^_^! Well,findLANPeers Let's Play ^_^!<<<<<<<<<<<<<<<<<<<", "peerName", neighbors.ID, "addrs:", neighbors.Addrs, "paddr", p.host.Peerstore().Addrs(neighbors.ID))
			//发现局域网内的邻居节点
			err := p.host.Connect(context.Background(), neighbors)
			if err != nil {
				log.Error("findLANPeers", "err", err.Error())
				continue
			}
			log.Info("findLANPeers", "connect neighbors success", neighbors.ID.Pretty())
			p.connManager.AddNeighbors(&neighbors)

		case <-p.ctx.Done():
			log.Info("findLANPeers", "process", "done")
			return
		}
	}
}

func (p *P2P) handleP2PEvent() {

	p.taskGroup.Add(1)
	defer p.taskGroup.Done()
	for {
		select {
		case <-p.ctx.Done():
			log.Info("handleP2PEvent close")
			return
		case data, ok := <-p.subChan:
			if !ok {
				return
			}
			msg, ok := data.(*queue.Message)
			if !ok || data == nil {
				log.Error("handleP2PEvent", "recv invalid msg, data=", data)
				continue
			}
			handler := protocol.GetEventHandler(msg.Ty)
			if handler == nil {
				log.Error("handleP2PEvent", "unknown message type", msg.Ty)
				continue
			}
			// 同步调用
			if handler.Inline {
				handler.CallBack(msg)
				continue
			}

			// 异步调用
			p.taskGroup.Add(1)
			go func(m *queue.Message) {
				defer p.taskGroup.Done()
				handler.CallBack(m)
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

// 创建空投地址
func (p *P2P) genAirDropKey() {
	p.taskGroup.Add(1)
	defer p.taskGroup.Done()
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
			if !resp.(*types.WalletStatus).GetIsHasSeed() {
				continue
			}

			if resp.(*types.WalletStatus).GetIsWalletLock() { //上锁状态，无法用助记词创建空投地址,等待...
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
	if walletPrivkey != "" && walletPrivkey[:2] == "0x" {
		walletPrivkey = walletPrivkey[2:]
	}

	walletPubkey, err := GenPubkey(walletPrivkey)
	if err != nil {

		return
	}
	//如果addrbook之前保存的savePrivkey相同，则意味着节点启动之前已经创建了airdrop 空投地址
	savePrivkey, _ := p.addrBook.GetPrivPubKey()
	if savePrivkey == walletPrivkey { //addrbook与wallet保存了相同的空投私钥，不需要继续导入
		log.Debug("genAirDropKey", " same privekey ,process done")
		return
	}
	if len(savePrivkey) != 2*privKeyCompressBytesLen { //非压缩私钥,兼容之前老版本的DHT非压缩私钥
		log.Debug("len savePrivkey", len(savePrivkey))
		unCompkey := p.addrBook.GetPrivkey()
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

	if savePrivkey != "" && !p.p2pCfg.WaitPid { //如果是waitpid 则不生成dht node award,保证之后一个空投地址，即钱包创建的airdropaddr
		//savePrivkey是随机私钥，兼容老版本，先对其进行导入钱包处理
		//进行压缩处理
		var parm types.ReqWalletImportPrivkey
		parm.Privkey = savePrivkey
		parm.Label = "dht node award"

		if strings.ToUpper(p.chainCfg.GetModuleConfig().Address.DefaultDriver) == "ETH" {
			parm.AddressID = 2 //eth address type
		}
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

	p.addrBook.saveKey(walletPrivkey, walletPubkey)
	p.reStart()
}

func newDB(name, backend, dir string, cache int32) dbm.DB {
	if name == "" {
		name = "p2pstore"
	}
	if backend == "" {
		backend = "leveldb"
	}
	if dir == "" || dir == "." {
		dir = "datadir"
	}
	if cache <= 0 {
		cache = 128
	}
	return dbm.NewDB(name, backend, dir, cache)
}
