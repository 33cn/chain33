// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package broadcast broadcast protocol
package broadcast

import (
	"encoding/hex"

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
	broadcastID = "/chain33/p2p/broadcast/1.0.0"
)

func init() {
	prototypes.RegisterProtocol(protoTypeID, &broadCastProtocol{})
	prototypes.RegisterStreamHandler(protoTypeID, broadcastID, &broadCastHandler{})
}

//
type broadCastProtocol struct {
	*prototypes.BaseProtocol

	txFilter        *utils.Filterdata
	blockFilter     *utils.Filterdata
	txSendFilter    *utils.Filterdata
	blockSendFilter *utils.Filterdata
	ltBlockCache    *utils.SpaceLimitCache
	p2pCfg          *p2pty.P2PSubConfig
}

// InitProtocol init protocol
func (protocol *broadCastProtocol) InitProtocol(env *prototypes.P2PEnv) {
	protocol.BaseProtocol = new(prototypes.BaseProtocol)

	protocol.P2PEnv = env
	//接收交易和区块过滤缓存, 避免重复提交到mempool或blockchain
	protocol.txFilter = utils.NewFilter(txRecvFilterCacheNum)
	protocol.blockFilter = utils.NewFilter(blockRecvFilterCacheNum)

	//发送交易和区块时过滤缓存, 解决冗余广播发送
	protocol.txSendFilter = utils.NewFilter(txSendFilterCacheNum)
	protocol.blockSendFilter = utils.NewFilter(blockSendFilterCacheNum)

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
}

type broadCastHandler struct {
	*prototypes.BaseStreamHandler
}

// Handle 处理请求
func (handler *broadCastHandler) Handle(stream core.Stream) {

	protocol := handler.GetProtocol().(*broadCastProtocol)
	pid := stream.Conn().RemotePeer()
	peerAddr := stream.Conn().RemoteMultiaddr().String()
	var data types.MessageBroadCast
	err := prototypes.ReadStream(&data, stream)
	if err != nil {
		log.Error("Handle", "pid", pid.Pretty(), "addr", peerAddr, "err", err)
		return
	}

	_ = protocol.handleReceive(data.Message, pid, peerAddr)
}

// SetProtocol set protocol
func (handler *broadCastHandler) SetProtocol(protocol prototypes.IProtocol) {
	handler.Protocol = protocol
}

// VerifyRequest verify request
func (handler *broadCastHandler) VerifyRequest(types.Message, *types.MessageComm) bool {
	return true
}

//
func (protocol *broadCastProtocol) handleEvent(msg *queue.Message) {

	//log.Debug("HandleBroadCastEvent", "msgTy", msg.Ty, "msgID", msg.broadcastID)
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
		return
	}
	//目前p2p可能存在多个插件并存，dht和gossip，消息回收容易混乱，需要进一步梳理 TODO：p2p模块热点区域消息回收
	//protocol.QueueClient.FreeMessage(msg)
	protocol.broadcast(sendData)
}

func (protocol *broadCastProtocol) broadcast(data interface{}) {

	pds := protocol.GetConnsManager().FetchConnPeers()
	//log.Debug("broadcast", "peerNum", len(pds))
	openedStreams := make([]core.Stream, 0)
	for _, pid := range pds {

		stream, err := protocol.sendPeer(pid, data, true)
		if err != nil {
			log.Error("broadcast", "send peer err", err)
		}
		if stream != nil {
			openedStreams = append(openedStreams, stream)
		}
	}

	// 广播发送数据结束后，统一关闭打开的stream
	for _, stream := range openedStreams {
		prototypes.CloseStream(stream)
	}
}

// 发送广播数据到节点, 支持延迟关闭内部stream，主要考虑多个节点并行发送情况，不需要等待关闭
func (protocol *broadCastProtocol) sendPeer(pid peer.ID, data interface{}, delayStreamClose bool) (core.Stream, error) {

	//这里传peeraddr用pid替代不会影响，内部只做log记录， 暂时不更改代码
	//TODO：增加peer addr获取渠道
	sendData, doSend := protocol.handleSend(data, pid, pid.Pretty())
	if !doSend {
		return nil, nil
	}
	//包装一层MessageBroadCast
	broadData := &types.MessageBroadCast{
		Message: sendData}

	stream, err := prototypes.NewStream(protocol.Host, pid, broadcastID)
	if err != nil {
		log.Error("sendPeer", "id", pid.Pretty(), "NewStreamErr", err)
		return nil, err
	}

	err = prototypes.WriteStream(broadData, stream)
	if err != nil {
		log.Error("sendPeer", "pid", pid.Pretty(), "WriteStream err", err)
	}
	if !delayStreamClose {
		prototypes.CloseStream(stream)
		stream = nil
	}
	return stream, err
}

// handleSend 对数据进行处理，包装成BroadCast结构
func (protocol *broadCastProtocol) handleSend(rawData interface{}, pid peer.ID, peerAddr string) (sendData *types.BroadCastData, doSend bool) {
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
	} else if ping, ok := rawData.(*types.P2PPing); ok {
		doSend = true
		sendData.Value = &types.BroadCastData_Ping{Ping: ping}
	}
	return
}

func (protocol *broadCastProtocol) handleReceive(data *types.BroadCastData, pid peer.ID, peerAddr string) (err error) {

	//接收网络数据不可靠
	defer func() {
		if r := recover(); r != nil {
			log.Error("handleReceive_Panic", "recvData", data, "pid", pid, "addr", peerAddr, "recoverErr", r)
		}
	}()
	if tx := data.GetTx(); tx != nil {
		err = protocol.recvTx(tx, pid, peerAddr)
	} else if ltTx := data.GetLtTx(); ltTx != nil {
		err = protocol.recvLtTx(ltTx, pid, peerAddr)
	} else if ltBlc := data.GetLtBlock(); ltBlc != nil {
		err = protocol.recvLtBlock(ltBlc, pid, peerAddr)
	} else if blc := data.GetBlock(); blc != nil {
		err = protocol.recvBlock(blc, pid, peerAddr)
	} else if query := data.GetQuery(); query != nil {
		err = protocol.recvQueryData(query, pid, peerAddr)
	} else if rep := data.GetBlockRep(); rep != nil {
		err = protocol.recvQueryReply(rep, pid, peerAddr)
	}
	if err != nil {
		log.Error("handleReceive", "pid", pid, "addr", peerAddr, "recvData", data.Value, "err", err)
	}
	return
}

func (protocol *broadCastProtocol) postBlockChain(blockHash string, pid peer.ID, block *types.Block) error {
	return protocol.P2PManager.PubBroadCast(blockHash, &types.BlockPid{Pid: pid.Pretty(), Block: block}, types.EventBroadcastAddBlock)
}

func (protocol *broadCastProtocol) postMempool(txHash string, tx *types.Transaction) error {
	return protocol.P2PManager.PubBroadCast(txHash, tx, types.EventTx)
}

type sendFilterInfo struct {
	//记录广播交易或区块时需要忽略的节点, 这些节点可能是交易的来源节点,也可能节点间维护了多条连接, 冗余发送
	ignoreSendPeers map[string]bool
}

//检测是否冗余发送, 或者添加到发送过滤(内部存在直接修改读写保护的数据, 对filter lru的读写需要外层锁保护)
func addIgnoreSendPeerAtomic(filter *utils.Filterdata, key string, pid peer.ID) (exist bool) {

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
	_, exist = info.ignoreSendPeers[pid.Pretty()]
	info.ignoreSendPeers[pid.Pretty()] = true
	return exist
}
