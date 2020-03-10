// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// package broadcast broadcast protocol
package broadcast

import (
	"context"
	"encoding/hex"

	"github.com/33cn/chain33/p2p/utils"

	//	"errors"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"

	prototypes "github.com/33cn/chain33/p2pnext/protocol/types"
	core "github.com/libp2p/go-libp2p-core"

	"github.com/33cn/chain33/common/log/log15"
	p2pty "github.com/33cn/chain33/p2pnext/types"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

var log = log15.New("module", "p2p.broadcast")

const (
	protoTypeID = "BroadcastProtocolType"
	ID          = "/chain33/p2p/broadcast/1.0.0"
)

func init() {
	prototypes.RegisterProtocolType(protoTypeID, &broadCastProtocol{})
	prototypes.RegisterStreamHandlerType(protoTypeID, ID, &broadCastHandler{})
}

//
type broadCastProtocol struct {
	*prototypes.BaseProtocol
	*prototypes.BaseStreamHandler

	txFilter        *utils.Filterdata
	blockFilter     *utils.Filterdata
	txSendFilter    *utils.Filterdata
	blockSendFilter *utils.Filterdata
	totalBlockCache *utils.SpaceLimitCache
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

	//在本地暂时缓存一些区块数据, 限制最大大小
	protocol.totalBlockCache = utils.NewSpaceLimitCache(blockCacheNum, maxBlockCacheByteSize)
	//接收到短哈希区块数据,只构建出区块部分交易,需要缓存, 并继续向对端节点请求剩余数据
	protocol.ltBlockCache = utils.NewSpaceLimitCache(blockCacheNum/2, maxBlockCacheByteSize/2)
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

	if subCfg.MinLtBlockTxNum <= 0 {
		subCfg.MinLtBlockTxNum = defaultMinLtBlockTxNum
	}
	protocol.p2pCfg = &subCfg
}

type broadCastHandler struct {
	*prototypes.BaseStreamHandler
}

//Handle 处理请求
func (handler *broadCastHandler) Handle(stream core.Stream) {

	protocol := handler.GetProtocol().(*broadCastProtocol)
	pid := stream.Conn().RemotePeer().Pretty()
	peerAddr := stream.Conn().RemoteMultiaddr().String()
	log.Debug("Handle", "pid", pid, "peerAddr", peerAddr)
	var data types.MessageBroadCast
	err := handler.ReadProtoMessage(&data, stream)
	if err != nil {
		log.Error("Handle", "pid", pid, "peerAddr", peerAddr, "err", err)
		return
	}

	_ = protocol.handleReceive(data.Message, pid, peerAddr)
}

// SetProtocol set protocol
func (handler *broadCastHandler) SetProtocol(protocol prototypes.IProtocol) {
	handler.BaseStreamHandler = new(prototypes.BaseStreamHandler)
	handler.Protocol = protocol
}

// VerifyRequest verify request
func (handler *broadCastHandler) VerifyRequest(types.Message, *types.MessageComm) bool {

	return true
}

//
func (protocol *broadCastProtocol) handleEvent(msg *queue.Message) {

	log.Debug("HandleBroadCastEvent", "msgTy", msg.Ty, "msgID", msg.ID)
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
		sendData = &types.P2PTx{Tx: tx}
	} else if block, ok := msg.GetData().(*types.Block); ok {
		protocol.blockFilter.Add(hex.EncodeToString(block.Hash(protocol.GetChainCfg())), true)
		sendData = &types.P2PBlock{Block: block}
	} else {
		return
	}

	protocol.sendAllStream(sendData)
}

func (protocol *broadCastProtocol) sendAllStream(data interface{}) {

	log.Debug("sendAllStream")
	pds := protocol.GetConnsManager().FindNearestPeers()

	for _, pid := range pds {

		err := protocol.sendStream(pid.Pretty(), data)
		if err != nil {
			log.Debug("sendAllStream", "sendStreamErr", err)
		}
	}
}

func (protocol *broadCastProtocol) sendStream(pid string, data interface{}) error {

	rawID, err := peer.IDB58Decode(pid)
	if err != nil {
		log.Error("sendStream", "id", pid, "decodePeerIDErr", err)
		return err
	}
	stream, err := protocol.Host.NewStream(context.Background(), rawID, ID)
	if err != nil {
		log.Error("sendStream", "id", pid, "NewStreamErr", err)
		return err
	}
	peerAddr := stream.Conn().RemoteMultiaddr().String()
	sendData, doSend := protocol.handleSend(data, pid, peerAddr)
	log.Debug("sendStream", "pid", pid, "peerAddr", peerAddr, doSend)
	if !doSend {
		return nil
	}

	//包装一层MessageBroadCast
	broadData := &types.MessageBroadCast{
		Message: sendData}

	err = protocol.SendProtoMessage(broadData, stream)
	if err != nil {
		log.Error("sendStream", "peerAddr", peerAddr, "send msg err", err)
		_ = stream.Close()
		return err
	}

	return nil
}

// handleSend 对数据进行处理，包装成BroadCast结构
func (protocol *broadCastProtocol) handleSend(rawData interface{}, pid, peerAddr string) (sendData *types.BroadCastData, doSend bool) {
	//出错处理
	defer func() {
		if r := recover(); r != nil {
			log.Error("handleSend_Panic", "sendData", rawData, "peerAddr", peerAddr, "recoverErr", r)
			doSend = false
		}
	}()
	log.Debug("ProcessSendP2PBegin", "peerID", pid, "peerAddr", peerAddr)
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
	log.Debug("handleSend", "peerAddr", peerAddr, "doSend", doSend)
	return
}

func (protocol *broadCastProtocol) handleReceive(data *types.BroadCastData, pid string, peerAddr string) (err error) {

	//接收网络数据不可靠
	defer func() {
		if r := recover(); r != nil {
			log.Error("handleReceive_Panic", "recvData", data, "peerAddr", peerAddr, "recoverErr", r)
		}
	}()
	log.Debug("handleReceive", "peerID", pid, "peerAddr", peerAddr)
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
	log.Debug("handleReceive", "peerAddr", peerAddr, "err", err)
	return
}

func (protocol *broadCastProtocol) sendToMempool(ty int64, data interface{}) (interface{}, error) {

	client := protocol.GetQueueClient()
	msg := client.NewMessage("mempool", ty, data)
	err := client.Send(msg, true)
	if err != nil {
		return nil, err
	}
	resp, err := client.WaitTimeout(msg, time.Second*10)
	if err != nil {
		return nil, err
	}
	return resp.Data, nil
}

func (protocol *broadCastProtocol) postBlockChain(blockHash, pid string, block *types.Block) error {
	return protocol.P2PManager.PubBroadCast(blockHash, &types.BlockPid{Pid: pid, Block: block}, types.EventBroadcastAddBlock)
}

func (protocol *broadCastProtocol) postMempool(txHash string, tx *types.Transaction) error {
	return protocol.P2PManager.PubBroadCast(txHash, tx, types.EventTx)
}

type sendFilterInfo struct {
	//记录广播交易或区块时需要忽略的节点, 这些节点可能是交易的来源节点,也可能节点间维护了多条连接, 冗余发送
	ignoreSendPeers map[string]bool
}

//检测是否冗余发送, 或者添加到发送过滤(内部存在直接修改读写保护的数据, 对filter lru的读写需要外层锁保护)
func addIgnoreSendPeerAtomic(filter *utils.Filterdata, key, pid string) (exist bool) {

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
