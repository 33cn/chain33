// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package broadcast broadcast protocol
package broadcast

import (
	"context"
	"encoding/hex"
	"strings"
	"time"

	"github.com/33cn/chain33/common/pubsub"

	"github.com/33cn/chain33/p2p/utils"

	"github.com/libp2p/go-libp2p-core/peer"

	prototypes "github.com/33cn/chain33/system/p2p/dht/protocol/types"
	core "github.com/libp2p/go-libp2p-core"

	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/queue"
	p2pty "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
)

var log = log15.New("module", "p2p.broadcast")

const (
	protoTypeID = "BroadcastProtocolType"
	broadcastV1 = "/chain33/p2p/broadcast/1.0.0"
	broadcastV2 = "/chain33/p2p/broadcast/2.0.0"
)

func init() {
	prototypes.RegisterProtocol(protoTypeID, &broadcastProtocol{})
	prototypes.RegisterStreamHandler(protoTypeID, broadcastV1, &broadcastHandler{})
	prototypes.RegisterStreamHandler(protoTypeID, broadcastV2, &broadcastHandlerV2{})
}

//
type broadcastProtocol struct {
	*prototypes.BaseProtocol

	txFilter          *utils.Filterdata
	blockFilter       *utils.Filterdata
	txSendFilter      *utils.Filterdata
	blockSendFilter   *utils.Filterdata
	ltBlockCache      *utils.SpaceLimitCache
	p2pCfg            *p2pty.P2PSubConfig
	broadcastPeers    map[peer.ID]context.CancelFunc
	simultaneousPeers map[peer.ID]struct{}
	ps                *pubsub.PubSub
	exitPeer          chan peer.ID
	errPeer           chan peer.ID
	newStream         chan core.Stream
	maxBroadcastPeers int
}

// InitProtocol init protocol
func (protocol *broadcastProtocol) InitProtocol(env *prototypes.P2PEnv) {
	protocol.BaseProtocol = new(prototypes.BaseProtocol)

	protocol.P2PEnv = env
	//接收交易和区块过滤缓存, 避免重复提交到mempool或blockchain
	protocol.txFilter = utils.NewFilter(txRecvFilterCacheNum)
	protocol.blockFilter = utils.NewFilter(blockRecvFilterCacheNum)

	//发送交易和区块时过滤缓存, 解决冗余广播发送
	protocol.txSendFilter = utils.NewFilter(txSendFilterCacheNum)
	protocol.blockSendFilter = utils.NewFilter(blockSendFilterCacheNum)
	protocol.ps = pubsub.NewPubSub(10000)
	protocol.exitPeer = make(chan peer.ID)
	protocol.errPeer = make(chan peer.ID)
	protocol.newStream = make(chan core.Stream)
	protocol.broadcastPeers = make(map[peer.ID]context.CancelFunc)
	protocol.simultaneousPeers = make(map[peer.ID]struct{})
	// 单独复制一份， 避免data race
	subCfg := *(env.SubConfig)
	//注册事件处理函数
	prototypes.RegisterEventHandler(types.EventTxBroadcast, protocol.handleEvent)
	prototypes.RegisterEventHandler(types.EventBlockBroadcast, protocol.handleEvent)

	//ttl至少设为2
	if subCfg.LightTxTTL <= 1 {
		subCfg.LightTxTTL = defaultLtTxBroadCastTTL
	}
	if subCfg.MaxTTL <= 0 {
		subCfg.MaxTTL = defaultMaxTxBroadCastTTL
	}
	if subCfg.MinLtBlockSize <= 0 {
		subCfg.MinLtBlockSize = defaultMinLtBlockSize
	}
	if subCfg.LtBlockCacheSize <= 0 {
		subCfg.LtBlockCacheSize = defaultLtBlockCacheSize
	}

	//接收到短哈希区块数据,只构建出区块部分交易,需要缓存, 并继续向对端节点请求剩余数据
	//内部组装成功失败或成功都会进行清理，实际运行并不会长期占用内存，只要限制极端情况最大值
	protocol.ltBlockCache = utils.NewSpaceLimitCache(ltBlockCacheNum, int(subCfg.LtBlockCacheSize*1024*1024))
	protocol.p2pCfg = &subCfg
	go protocol.handleExit()
}

// 处理退出，释放资源
func (protocol *broadcastProtocol) handleExit() {
	select {
	case <-protocol.Ctx.Done():
		protocol.ps.Shutdown()
	}
}

// 更新广播节点
func (protocol *broadcastProtocol) manageBroadcastPeers() {

	// 节点启动前十分钟，网络还未稳定，每5秒更新一次连接信息, 后续每分钟更新一次
	tick := time.NewTicker(time.Second * 5)
	timer := time.NewTimer(time.Minute * 10)

	for {
		select {
		//定时获取节点， 加入广播节点列表
		case <-tick.C:
			morePeers := protocol.maxBroadcastPeers - len(protocol.broadcastPeers)
			peers := protocol.GetConnsManager().FetchConnPeers()
			for i := 0; i < len(peers) && morePeers > 0; i++ {
				pid := peers[i]
				_, ok := protocol.broadcastPeers[pid]
				if !ok {
					protocol.addBroadcastPeer(pid, nil)
					morePeers--
				}
			}

		case <-timer.C:
			tick.Stop()
			timer.Stop()
			tick = time.NewTicker(time.Minute)

		//被动接收到节点 加入广播节点列表
		case stream := <-protocol.newStream:
			pid := stream.Conn().RemotePeer()
			//广播节点列表已达上限
			if len(protocol.broadcastPeers) >= protocol.maxBroadcastPeers {
				_ = stream.Reset()
				break
			}
			pCancel, ok := protocol.broadcastPeers[pid]
			// 该节点已经在广播列表中, 双方同时发起的情况导致双向重复, 这里采用直接比较节点id策略，保留节点id大的的一方主动发起的广播流
			if ok {

				if strings.Compare(pid.String(), stream.Conn().LocalPeer().String()) > 0 {
					//关闭本地已存在的该节点广播协程
					pCancel()
					//上述操作会将节点从相关结构删除，需要做记录避免误删除
					protocol.simultaneousPeers[pid] = struct{}{}
				} else {
					//保留本地已存在的广播协程， 直接关闭stream并退出
					_ = stream.Reset()
					break
				}
			}
			protocol.addBroadcastPeer(pid, stream)
		case pid := <-protocol.exitPeer:
			protocol.removeBroadcastPeer(pid)
		case pid := <-protocol.errPeer:
			//错误节点减少tag值， 这样在内部连接超额时会优先断开
			protocol.Host.ConnManager().UpsertTag(pid, broadcastTag, func(oldVal int) int { return oldVal - 1 })
		case <-protocol.Ctx.Done():
			tick.Stop()
			timer.Stop()
			return
		}
	}
}

// 增加广播节点， 每个节点分配一个协程处理广播逻辑
func (protocol *broadcastProtocol) addBroadcastPeer(id peer.ID, stream core.Stream) {
	var err error
	if stream == nil {
		stream, err = prototypes.NewStream(protocol.Host, id, broadcastV2, broadcastV1)
		if err != nil {
			protocol.errPeer <- id
			log.Error("addBroadcastPeer", "pid", id, "newStreamErr", err)
			return
		}
	}
	// 广播节点加入保护， 避免被连接管理误删除
	pCtx, pCancel := context.WithCancel(protocol.Ctx)
	protocol.broadcastPeers[id] = pCancel
	protocol.Host.ConnManager().Protect(id, broadcastTag)
	// 区分不同的广播版本
	if stream.Protocol() == broadcastV2 {
		go protocol.broadcastV2(id, stream, pCtx, pCancel)

	} else if stream.Protocol() == broadcastV1 {
		go protocol.broadcastV1(id, stream, pCtx)
	}
}

// 移除广播节点
func (protocol *broadcastProtocol) removeBroadcastPeer(id peer.ID) {

	//同步打开的情况特殊处理
	_, ok := protocol.simultaneousPeers[id]
	if ok {
		delete(protocol.simultaneousPeers, id)
		return
	}
	protocol.Host.ConnManager().Unprotect(id, broadcastTag)
	delete(protocol.broadcastPeers, id)
}

// 处理系统广播发送事件，交易及区块
func (protocol *broadcastProtocol) handleEvent(msg *queue.Message) {

	var sendData interface{}
	if tx, ok := msg.GetData().(*types.Transaction); ok {
		txHash := hex.EncodeToString(tx.Hash())
		//此处使用新分配结构，避免重复修改已保存的ttl
		route := &types.P2PRoute{TTL: 1}
		//是否已存在记录，不存在表示本节点发起的交易
		data, exist := protocol.txFilter.Get(txHash)
		if ttl, ok := data.(*types.P2PRoute); exist && ok {
			route.TTL = ttl.GetTTL() + 1
		} else {
			protocol.txFilter.Add(txHash, true)
		}
		sendData = &types.P2PTx{Tx: tx, Route: route}
	} else if block, ok := msg.GetData().(*types.Block); ok {
		protocol.blockFilter.Add(hex.EncodeToString(block.Hash(protocol.GetChainCfg())), true)
		sendData = &types.P2PBlock{Block: block}
	} else {
		log.Error("handleEvent", "receive unexpect msg", msg)
		return
	}
	//目前p2p可能存在多个插件并存，dht和gossip，消息回收容易混乱，需要进一步梳理 TODO：p2p模块热点区域消息回收
	//protocol.QueueClient.FreeMessage(msg)

	//通过pubsub发布需要广播的信息，各个节点sub并发处理
	protocol.ps.FIFOPub(sendData, bcTopic)
}

//TODO 老版本广播后期全网升级后，可以移除
func (protocol *broadcastProtocol) broadcastV1(pid peer.ID, stream core.Stream, peerCtx context.Context) {

	var err error
	outgoing := protocol.ps.Sub(bcTopic)
	peerAddr := stream.Conn().RemoteMultiaddr().String()
	sPid := pid.String()
	log.Debug("broadcastV1Start", "pid", sPid, "addr", peerAddr)
	defer func() {
		protocol.ps.Unsub(outgoing)
		protocol.exitPeer <- pid
		if stream != nil {
			_ = stream.Reset()
		}
		if err != nil {
			protocol.errPeer <- pid
		}
		log.Debug("broadcastV1End", "pid", sPid, "addr", peerAddr)
	}()

	for {
		select {
		case data := <-outgoing:
			sendData, doSend := protocol.handleSend(data, sPid, peerAddr)
			if !doSend {
				break //ignore send
			}
			//包装一层MessageBroadCast
			broadData := &types.MessageBroadCast{
				Message: sendData}

			err = prototypes.WriteStream(broadData, stream)
			if err != nil {
				log.Error("broadcastV1", "pid", sPid, "WriteStream err", err)
				return
			}
			err = prototypes.CloseStream(stream)
			if err != nil {
				log.Error("broadcastV1", "pid", sPid, "CloseStream err", err)
				return
			}
			stream, err = prototypes.NewStream(protocol.Host, pid, broadcastV1)
			if err != nil {
				log.Error("broadcastV1", "pid", sPid, "NewStreamErr", err)
				return
			}
		case <-peerCtx.Done():
			return

		}
	}

}

func (protocol *broadcastProtocol) broadcastV2(pid peer.ID, stream core.Stream, peerCtx context.Context, peerCancel context.CancelFunc) {

	sPid := pid.String()
	outgoing := protocol.ps.Sub(bcTopic, sPid)
	peerAddr := stream.Conn().RemoteMultiaddr().String()
	log.Debug("broadcastV2Start", "pid", sPid, "addr", peerAddr)
	errChan := make(chan error, 2)
	defer func() {
		protocol.ps.Unsub(outgoing)
		err := <-errChan
		if err != nil {
			protocol.errPeer <- pid
		}
		protocol.exitPeer <- pid
		_ = stream.Reset()
		log.Debug("broadcastV2End", "pid", sPid, "addr", peerAddr)
	}()

	// handle broadcast recv
	go func() {
		data := &types.BroadCastData{}
		for {
			//阻塞等待
			err := prototypes.ReadStreamTimeout(data, stream, -1)
			if err != nil {
				log.Error("broadcastV2", "pid", sPid, "addr", peerAddr, "read stream err", err)
				errChan <- err
				peerCancel()
				return
			}
			_ = protocol.handleReceive(data, sPid, peerAddr, broadcastV2)
		}
	}()

	// handle broadcast send
	for {
		var err error
		select {
		case data := <-outgoing:
			sendData, doSend := protocol.handleSend(data, sPid, peerAddr)
			if !doSend {
				break //ignore send
			}
			err = prototypes.WriteStream(sendData, stream)
			if err != nil {
				errChan <- err
				log.Error("broadcastV2", "pid", sPid, "WriteStream err", err)
				return
			}

		case <-peerCtx.Done():
			errChan <- nil
			return
		}
	}
}

// 发送广播数据到节点, 支持延迟关闭内部stream，主要考虑多个节点并行发送情况，不需要等待关闭
func (protocol *broadcastProtocol) sendPeer(data interface{}, pid, version string) error {

	if version == broadcastV2 {
		protocol.ps.Pub(data, pid)
		return nil
	}
	// broadcast v1 TODO 版本升级后移除代码
	sendData, doSend := protocol.handleSend(data, pid, pid)
	if !doSend {
		return nil
	}
	//包装一层MessageBroadCast
	broadData := &types.MessageBroadCast{
		Message: sendData}

	rawPid, err := peer.Decode(pid)
	if err != nil {
		log.Error("sendPeer", "id", pid, "decode pid err", err)
		return err
	}
	stream, err := prototypes.NewStream(protocol.Host, rawPid, broadcastV1)
	if err != nil {
		log.Error("sendPeer", "id", pid, "NewStreamErr", err)
		return err
	}

	err = prototypes.WriteStream(broadData, stream)
	if err != nil {
		log.Error("sendPeer", "pid", pid, "WriteStream err", err)
		return err
	}
	err = prototypes.CloseStream(stream)
	if err != nil {
		log.Error("sendPeer", "pid", pid, "CloseStream err", err)
		return err
	}
	return nil
}

// handleSend 对数据进行处理，包装成BroadCast结构
func (protocol *broadcastProtocol) handleSend(rawData interface{}, pid string, peerAddr string) (sendData *types.BroadCastData, doSend bool) {
	//出错处理
	defer func() {
		if r := recover(); r != nil {
			log.Error("handleSend_Panic", "sendData", rawData, "pid", pid, "recoverErr", r)
			doSend = false
		}
	}()
	sendData = &types.BroadCastData{}

	doSend = false
	if tx, ok := rawData.(*types.P2PTx); ok {
		doSend = protocol.sendTx(tx, sendData, pid, peerAddr)
	} else if blc, ok := rawData.(*types.P2PBlock); ok {
		doSend = protocol.sendBlock(blc, sendData, pid, peerAddr)
	} else if query, ok := rawData.(*types.P2PQueryData); ok {
		doSend = protocol.sendQueryData(query, sendData, peerAddr)
	} else if rep, ok := rawData.(*types.P2PBlockTxReply); ok {
		doSend = protocol.sendQueryReply(rep, sendData, peerAddr)
	}
	return
}

func (protocol *broadcastProtocol) handleReceive(data *types.BroadCastData, pid, peerAddr, version string) (err error) {

	//接收网络数据不可靠
	defer func() {
		if r := recover(); r != nil {
			log.Error("handleReceive_Panic", "recvData", data, "pid", pid, "addr", peerAddr, "recoverErr", r)
		}
	}()
	if tx := data.GetTx(); tx != nil {
		err = protocol.recvTx(tx, pid, peerAddr)
	} else if ltTx := data.GetLtTx(); ltTx != nil {
		err = protocol.recvLtTx(ltTx, pid, peerAddr, version)
	} else if ltBlc := data.GetLtBlock(); ltBlc != nil {
		err = protocol.recvLtBlock(ltBlc, pid, peerAddr, version)
	} else if blc := data.GetBlock(); blc != nil {
		err = protocol.recvBlock(blc, pid, peerAddr)
	} else if query := data.GetQuery(); query != nil {
		err = protocol.recvQueryData(query, pid, peerAddr, version)
	} else if rep := data.GetBlockRep(); rep != nil {
		err = protocol.recvQueryReply(rep, pid, peerAddr, version)
	}
	if err != nil {
		log.Error("handleReceive", "pid", pid, "addr", peerAddr, "recvData", data.Value, "err", err)
	}
	return
}

func (protocol *broadcastProtocol) postBlockChain(blockHash, pid string, block *types.Block) error {
	return protocol.P2PManager.PubBroadCast(blockHash, &types.BlockPid{Pid: pid, Block: block}, types.EventBroadcastAddBlock)
}

func (protocol *broadcastProtocol) postMempool(txHash string, tx *types.Transaction) error {
	return protocol.P2PManager.PubBroadCast(txHash, tx, types.EventTx)
}

type sendFilterInfo struct {
	//记录广播交易或区块时需要忽略的节点, 这些节点可能是交易的来源节点,也可能节点间维护了多条连接, 冗余发送
	ignoreSendPeers map[string]bool
}

//检测是否冗余发送, 或者添加到发送过滤(内部存在直接修改读写保护的数据, 对filter lru的读写需要外层锁保护)
func addIgnoreSendPeerAtomic(filter *utils.Filterdata, key string, pid string) (exist bool) {

	filter.GetAtomicLock()
	defer filter.ReleaseAtomicLock()
	var info *sendFilterInfo
	if !filter.Contains(key) { //之前没有收到过这个key
		info = &sendFilterInfo{ignoreSendPeers: make(map[string]bool)}
		filter.Add(key, info)
	} else {
		data, _ := filter.Get(key)
		info = data.(*sendFilterInfo)
	}
	_, exist = info.ignoreSendPeers[pid]
	info.ignoreSendPeers[pid] = true
	return exist
}

// 删除发送过滤器记录
func removeIgnoreSendPeerAtomic(filter *utils.Filterdata, key, pid string) {

	filter.GetAtomicLock()
	defer filter.ReleaseAtomicLock()
	if filter.Contains(key) {
		data, _ := filter.Get(key)
		info := data.(*sendFilterInfo)
		delete(info.ignoreSendPeers, pid)
	}
}
