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
			p.taskGroup.Add(1)
			p.genAirDropKey()
		} else { //先用随机身份启动: 身份一经发布就不再更换, 钱包就绪后只把它导入钱包备份
			p.addrBook.Randkey()
			p.taskGroup.Add(1)
			go p.genAirDropKey() //非阻塞模式
		}
	} else { //非阻塞模式
		p.taskGroup.Add(1)
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
	// taskGroup.Add 必须在 goroutine 启动前调用, 避免与 waitTaskDone 的 Wait 竞态
	p.taskGroup.Add(1)
	go p.managePeers()
	p.taskGroup.Add(1)
	go p.findLANPeers()
	for i := 0; i < runtime.NumCPU(); i++ {
		p.taskGroup.Add(1)
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

// managePeers 由 StartP2P 启动, 调用方需先执行 taskGroup.Add(1)
func (p *P2P) managePeers() {
	defer p.taskGroup.Done()
	p.taskGroup.Add(1)
	go p.connManager.MonitorAllPeers(p.taskGroup)
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
// 由 StartP2P 启动, 调用方需先执行 taskGroup.Add(1)
func (p *P2P) findLANPeers() {
	defer p.taskGroup.Done()
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
			log.Debug("^_^! Well,findLANPeers Let's Play ^_^!<<<<<<<<<<<<<<<<<<<", "peerName", neighbors.ID, "addrs:", neighbors.Addrs, "paddr", p.host.Peerstore().Addrs(neighbors.ID))
			//发现局域网内的邻居节点; 拨号带重试且在独立 goroutine 中进行, 既不会因一次偶发失败
			//丢掉该邻居, 也不会阻塞后续邻居的处理。
			p.taskGroup.Add(1)
			go p.connectLANPeer(neighbors)

		case <-p.ctx.Done():
			log.Info("findLANPeers", "process", "done")
			return
		}
	}
}

// connectLANPeer 连接一个 mDNS 发现的局域网邻居, 失败按退避+抖动重试。
//
// 需要重试的原因: 两个节点几乎同时拨号对方时(局域网内节点同时启动/重启就是这样),
// 由于 libp2p 默认用监听端口作为拨号源端口(reuseport), 双方的源端口与目的端口相同,
// 交叉的两个 SYN 会被内核合并成同一条 TCP 连接(TCP 同时打开), 两端都按 TLS client 发起
// 握手, 握手必然失败(tls: received unexpected handshake message of type *tls.clientHelloMsg
// when waiting for *tls.serverHelloMsg), 两个方向的拨号同时报错。而 mDNS 对同一邻居只通知
// 一次, 旧实现一次拨号失败即丢弃该邻居, 只能等 DHT 路由表刷新(分钟级)才能重新连上,
// 期间该节点一直少一个邻居。这里退避重试, 并在退避上加抖动: 抖动让两端重试的时刻错开,
// 不会再次同时打开, 通常第二次尝试即可连上。
func (p *P2P) connectLANPeer(neighbors peer.AddrInfo) {
	defer p.taskGroup.Done()
	const maxAttempts = 5
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			// 线性退避 + 抖动, 抖动用于打破双方同时重试的对称性
			backoff := time.Duration(attempt)*2*time.Second + time.Duration(rand.Int63n(int64(2*time.Second)))
			select {
			case <-p.ctx.Done():
				return
			case <-time.After(backoff):
			}
		}
		err := p.host.Connect(p.ctx, neighbors)
		if err == nil {
			log.Info("findLANPeers", "connect neighbors success", neighbors.ID.String())
			p.connManager.AddNeighbors(&neighbors)
			return
		}
		// 同时拨号时对方可能已经连上我们, 此时无需再拨
		if len(p.host.Network().ConnsToPeer(neighbors.ID)) != 0 {
			log.Info("findLANPeers", "connect neighbors success", neighbors.ID.String(), "via", "inbound")
			p.connManager.AddNeighbors(&neighbors)
			return
		}
		log.Error("findLANPeers", "err", err.Error(), "peer", neighbors.ID.String(), "attempt", attempt+1)
	}
}

// handleP2PEvent 由 StartP2P 启动, 调用方需先执行 taskGroup.Add(1)
func (p *P2P) handleP2PEvent() {
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
// 注意: taskGroup.Add 必须由调用方在启动 goroutine 之前完成,
// 否则会与 waitTaskDone 中的 Wait 形成 data race
func (p *P2P) genAirDropKey() {
	// 必须先归还 taskGroup 名额再 reStart:
	// reStart -> CloseP2P -> waitTaskDone 会等待本 goroutine 的名额,
	// 若持有名额调用会自等待 20s 超时
	needRestart := p.doGenAirDropKey()
	if needRestart {
		p.reStart()
	}
}

func (p *P2P) doGenAirDropKey() bool {
	defer p.taskGroup.Done()
	for { //等待钱包创建，解锁
		select {
		case <-p.ctx.Done():
			log.Info("genAirDropKey", "p2p closed")
			return false
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
	//用助记词和随机索引创建空投地址。纳秒加 pid，避免同一秒启动的节点得到同一个索引。
	randIndex := airDropIndex(types.Now().UnixNano() + int64(os.Getpid()))
	reqIndex := &types.Int32{Data: randIndex}
	msg, err := p.api.ExecWalletFunc("wallet", "NewAccountByIndex", reqIndex)
	if err != nil {
		log.Error("genAirDropKey", "NewAccountByIndex err", err)
		return false
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

		return false
	}
	//如果addrbook之前保存的savePrivkey相同，则意味着节点启动之前已经创建了airdrop 空投地址
	savePrivkey, _ := p.addrBook.GetPrivPubKey()
	if savePrivkey == walletPrivkey { //addrbook与wallet保存了相同的空投私钥，不需要继续导入
		log.Debug("genAirDropKey", " same privekey ,process done")
		return false
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

	//addrbook 里已经有 dht 节点身份时, 把它导入钱包做备份(该私钥由助记词推不出来)。
	//identitySavedInWallet: 私钥已经在钱包里(ErrPrivkeyExist)也算成功, 不必再试;
	//名字冲突时才换一个随机名字重试; 其它错误记录日志后放弃, 不影响节点继续运行。
	if savePrivkey != "" && !p.p2pCfg.WaitPid {
		var parm types.ReqWalletImportPrivkey
		parm.Privkey = savePrivkey
		parm.Label = "dht node award"

		if strings.ToUpper(p.chainCfg.GetModuleConfig().Address.DefaultDriver) == "ETH" {
			parm.AddressID = 2 //eth address type
		}
		for {
			_, err = p.api.ExecWalletFunc("wallet", "WalletImportPrivkey", &parm)
			if err == nil || err == types.ErrPrivkeyExist {
				break
			}
			if err == types.ErrLabelHasUsed {
				//切换随机lable
				parm.Label = fmt.Sprintf("node award %d", rand.Int31n(1024000))
				continue
			}
			log.Error("genAirDropKey", "WalletImportPrivkey err", err)
			break
		}
	}

	// 节点身份一旦在 addrbook 里持久化, 就不再在运行期用钱包空投地址替换它:
	// 此时 DHT host 已经用这个私钥对外发布了 peer id, 运行期更换会让所有按 peer id
	// 配置了本节点的对端(peering 列表、TSS peers、白名单等)全部失效。
	// v1.72.1 修好空投索引取值范围后(ec5476309 等), 这条替换路径开始真的执行, 于是
	// 暴露成“钱包就绪后约 1 秒 peer id 变化”的确定性回归——一个新节点只在第一次
	// 钱包就绪时换一次(空投索引 sticky), plugin ci_rgbx 的 TSS peers 即被此打断。
	// 注意 waitPid=false(默认)时 initP2P 已经 Randkey() 落盘了一把身份, 所以这类节点
	// 读到非空身份就直接返回, 不会采用钱包空投地址; 只有 addrbook 里没有身份、且在
	// DHT host 启动之前就拿到钱包空投地址的节点(waitPid=true)才会用它作为节点身份。
	if savePrivkey != "" {
		log.Info("genAirDropKey", "keep the established dht identity, skip the airdrop key replacement")
		return false
	}

	p.addrBook.saveKey(walletPrivkey, walletPubkey)
	return true
}

// airDropSpan 是空投索引闭区间的长度。少 1 会让 AirDropMaxIndex 取不到。
func airDropSpan() int32 {
	return int32(types.AirDropMaxIndex-types.AirDropMinIndex) + 1
}

// airDropIndex 在空投合法闭区间内取索引。
func airDropIndex(seed int64) int32 {
	r := rand.New(rand.NewSource(seed))
	return int32(types.AirDropMinIndex) + r.Int31n(airDropSpan())
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
