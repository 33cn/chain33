// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package broadcast broadcast protocol
package broadcast

import (
	"context"
	"encoding/hex"
	"sync/atomic"

	"github.com/33cn/chain33/common/pubsub"

	"github.com/33cn/chain33/p2p/utils"

	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/queue"
	prototypes "github.com/33cn/chain33/system/p2p/dht/protocol/types"
	p2pty "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
)

var log = log15.New("module", "p2p.broadcast")

const (
	protoTypeID = "BroadcastProtocolType"
	broadcastV1 = "/chain33/p2p/broadcast/1.0.0"
	//broadcastV2     = "/chain33/p2p/broadcast/2.0.0"
	broadcastPubSub = "/chain33/broadcast/pubsub/1.0.0"
)

func init() {
	prototypes.RegisterProtocol(protoTypeID, &broadcastProtocol{})
	prototypes.RegisterStreamHandler(protoTypeID, broadcastV1, &broadcastHandler{})
	//prototypes.RegisterStreamHandler(protoTypeID, broadcastV2, &broadcastHandlerV2{})
	prototypes.RegisterStreamHandler(protoTypeID, broadcastPubSub, &pubsubHandler{})
}

//
type broadcastProtocol struct {
	*prototypes.BaseProtocol

	txFilter        *utils.Filterdata
	blockFilter     *utils.Filterdata
	txSendFilter    *utils.Filterdata
	blockSendFilter *utils.Filterdata
	ltBlockCache    *utils.SpaceLimitCache
	p2pCfg          *p2pty.P2PSubConfig
	broadcastPeers  map[peer.ID]context.CancelFunc
	ps              *pubsub.PubSub
	exitPeer        chan peer.ID
	errPeer         chan peer.ID
	// 接收V1版本节点
	peerV1    chan peer.ID
	peerV1Num int32
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
	protocol.peerV1 = make(chan peer.ID, 5)
	protocol.broadcastPeers = make(map[peer.ID]context.CancelFunc)
	// 单独复制一份， 避免data race
	subCfg := *(env.SubConfig)
	//注册事件处理函数
	prototypes.RegisterEventHandler(types.EventTxBroadcast, protocol.handleBroadCastEvent)
	prototypes.RegisterEventHandler(types.EventBlockBroadcast, protocol.handleBroadCastEvent)

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

	// 老版本保持兼容性， 默认最多选择5个节点广播
	if subCfg.MaxBroadcastPeers <= 0 {
		subCfg.MaxBroadcastPeers = 5
	}

	//接收到短哈希区块数据,只构建出区块部分交易,需要缓存, 并继续向对端节点请求剩余数据
	//内部组装成功失败或成功都会进行清理，实际运行并不会长期占用内存，只要限制极端情况最大值
	protocol.ltBlockCache = utils.NewSpaceLimitCache(ltBlockCacheNum, int(subCfg.LtBlockCacheSize*1024*1024))
	protocol.p2pCfg = &subCfg

	// pub sub broadcast
	go newPubSub(protocol).broadcast()
	go protocol.handleClassicBroadcast()
}

// 处理系统广播发送事件，交易及区块
func (protocol *broadcastProtocol) handleBroadCastEvent(msg *queue.Message) {

	var sendData interface{}
	var topic, hash string
	var filter *utils.Filterdata
	if tx, ok := msg.GetData().(*types.Transaction); ok {
		hash = hex.EncodeToString(tx.Hash())
		filter = protocol.txFilter
		topic = psTxTopic
		//兼容老版本，总是转发全交易
		route := &types.P2PRoute{TTL: 1}
		sendData = &types.P2PTx{Tx: tx, Route: route}
	} else if block, ok := msg.GetData().(*types.Block); ok {
		hash = hex.EncodeToString(block.Hash(protocol.GetChainCfg()))
		filter = protocol.blockFilter
		topic = psBlockTopic
		sendData = &types.P2PBlock{Block: block}
	} else {
		log.Error("handleBroadCastEvent", "receive unexpect msg", msg)
		return
	}
	//目前p2p可能存在多个插件并存，dht和gossip，消息回收容易混乱，需要进一步梳理 TODO：p2p模块热点区域消息回收
	//protocol.QueueClient.FreeMessage(msg)

	// pub sub只需要转发本节点产生的交易或区块
	if !filter.Contains(hash) {
		filter.Add(hash, struct{}{})
		protocol.ps.FIFOPub(msg.GetData(), topic)
	}

	//发布到老版本接收通道
	if atomic.LoadInt32(&protocol.peerV1Num) > 0 {
		protocol.ps.FIFOPub(sendData, bcTopic)
	}
}

// 发送广播数据到节点, 支持延迟关闭内部stream，主要考虑多个节点并行发送情况，不需要等待关闭
func (protocol *broadcastProtocol) sendPeer(data interface{}, pid, version string) error {

	//if version == broadcastV2 {
	//	protocol.ps.Pub(data, pid)
	//	return nil
	//}
	// broadcast v1 TODO 版本升级后移除代码
	sendData, doSend := protocol.handleSend(data, pid)
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
func (protocol *broadcastProtocol) handleSend(rawData interface{}, pid string) (sendData *types.BroadCastData, doSend bool) {
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
		doSend = protocol.sendTx(tx, sendData, pid)
	} else if blc, ok := rawData.(*types.P2PBlock); ok {
		doSend = protocol.sendBlock(blc, sendData, pid)
	} else if query, ok := rawData.(*types.P2PQueryData); ok {
		doSend = protocol.sendQueryData(query, sendData, pid)
	} else if rep, ok := rawData.(*types.P2PBlockTxReply); ok {
		doSend = protocol.sendQueryReply(rep, sendData, pid)
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
		err = protocol.recvTx(tx, pid)
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
