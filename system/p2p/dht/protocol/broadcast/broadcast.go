// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package broadcast broadcast protocol
package broadcast

import (
	"encoding/hex"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/33cn/chain33/common"

	"github.com/33cn/chain33/common/pubsub"

	"github.com/33cn/chain33/p2p/utils"

	"github.com/libp2p/go-libp2p-core/peer"

	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	p2pty "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
)

var log = log15.New("module", "p2p.broadcast")

func init() {
	protocol.RegisterProtocolInitializer(InitProtocol)
}

//
type broadcastProtocol struct {
	*protocol.P2PEnv
	txFilter    *utils.Filterdata
	blockFilter *utils.Filterdata
	cfg         p2pty.BroadcastConfig
	ps          *pubsub.PubSub
	syncStatus  bool
	currHeight  int64
	lock        sync.RWMutex
	ltB         *ltBroadcast
}

// InitProtocol init protocol
func InitProtocol(env *protocol.P2PEnv) {
	new(broadcastProtocol).init(env)
}

func (p *broadcastProtocol) init(env *protocol.P2PEnv) {

	p.P2PEnv = env
	p.ps = pubsub.NewPubSub(1024)
	// 单独复制一份， 避免data race
	p.cfg = env.SubConfig.Broadcast

	// set default params
	if p.cfg.TxFilterLen <= 0 {
		p.cfg.TxFilterLen = txRecvFilterCacheNum
	}
	if p.cfg.BlockFilterLen <= 0 {
		p.cfg.BlockFilterLen = blockRecvFilterCacheNum
	}
	if p.cfg.MinLtBlockSize <= 0 {
		p.cfg.MinLtBlockSize = defaultMinLtBlockSize
	}

	if p.cfg.LtBlockPendTimeout <= 0 {
		p.cfg.LtBlockPendTimeout = defaultLtBlockTimeout
	}

	//接收交易和区块过滤缓存, 避免重复提交到mempool或blockchain
	p.txFilter = utils.NewFilter(p.cfg.TxFilterLen)
	p.blockFilter = utils.NewFilter(p.cfg.BlockFilterLen)

	//注册事件处理函数
	protocol.RegisterEventHandler(types.EventTxBroadcast, p.handleBroadcastSend)
	protocol.RegisterEventHandler(types.EventBlockBroadcast, p.handleBroadcastSend)
	protocol.RegisterEventHandler(types.EventIsSync, p.handleIsSyncEvent)
	protocol.RegisterEventHandler(types.EventAddBlock, p.handleAddBlock)

	// pub sub broadcast
	newPubSub(p).broadcast()
	if !p.cfg.DisableLtBlock {
		p.ltB = newLtBroadcast(p)
		p.ltB.broadcast()
	}
}

func (p *broadcastProtocol) getSyncStatus() bool {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.syncStatus
}

func (p *broadcastProtocol) handleIsSyncEvent(msg *queue.Message) {
	p.lock.Lock()
	p.syncStatus = true
	p.lock.Unlock()
}

func (p *broadcastProtocol) handleAddBlock(msg *queue.Message) {
	atomic.StoreInt64(&p.currHeight, msg.GetData().(*types.Block).GetHeight())
}

func (p *broadcastProtocol) getCurrentHeight() int64 {
	return atomic.LoadInt64(&p.currHeight)
}

type publishMsg struct {
	topic string
	msg   types.Message
}

type subscribeMsg struct {
	topic       string
	value       types.Message
	receiveFrom peer.ID
	publisher   peer.ID
}

// 处理系统广播发送事件，交易及区块
func (p *broadcastProtocol) handleBroadcastSend(msg *queue.Message) {

	var topic, hash string
	var filter *utils.Filterdata
	broadcastData := msg.GetData().(types.Message)
	if tx, ok := broadcastData.(*types.Transaction); ok {
		hash = hex.EncodeToString(tx.Hash())
		filter = p.txFilter
		topic = psTxTopic
	} else if block, ok := broadcastData.(*types.Block); ok {
		hash = hex.EncodeToString(block.Hash(p.ChainCfg))
		filter = p.blockFilter
		topic = psBlockTopic
		// light block
		if !p.cfg.DisableLtBlock &&
			len(block.Txs) > 0 &&
			block.Size() > p.cfg.MinLtBlockSize {
			broadcastData = p.buildLtBlock(block)
			topic = psLtBlockTopic
		}
	} else {
		log.Error("handleBroadcastSend", "receive unexpect msg", msg)
		return
	}
	//目前p2p可能存在多个插件并存，dht和gossip，消息回收容易混乱，需要进一步梳理 TODO：p2p模块热点区域消息回收
	//p.QueueClient.FreeMessage(msg)

	// pub sub只需要转发本节点产生的交易或区块
	if !filter.Contains(hash) {
		filter.Add(hash, struct{}{})
		p.ps.Pub(publishMsg{msg: broadcastData, topic: topic}, psBroadcast)
	}
}

func (p *broadcastProtocol) buildLtBlock(block *types.Block) *types.LightBlock {

	ltBlock := &types.LightBlock{}
	ltBlock.Header = block.GetHeader(p.ChainCfg)
	ltBlock.Header.Signature = block.Signature
	ltBlock.MinerTx = block.Txs[0]
	ltBlock.STxHashes = make([]string, 0, ltBlock.GetHeader().GetTxCount())
	ltBlock.BlockDataHash = common.Sha256(types.Encode(block))
	for _, tx := range block.Txs[1:] {
		//tx short hash
		ltBlock.STxHashes = append(ltBlock.STxHashes, types.CalcTxShortHash(tx.Hash()))
	}
	return ltBlock
}

func (p *broadcastProtocol) handleBroadcastReceive(msg subscribeMsg) {

	var err error
	var hash string
	topic := msg.topic
	defer func() {
		if r := recover(); r != nil {
			log.Error("handleReceive_Panic", "topic", topic, "from", msg.publisher.String(), "recoverErr", r)
		}
	}()
	// 将接收的交易或区块 转发到内部对应模块
	if topic == psTxTopic {
		tx := msg.value.(*types.Transaction)
		hash = hex.EncodeToString(tx.Hash())
		err = p.postMempool(hash, tx)

	} else if topic == psBlockTopic {
		block := msg.value.(*types.Block)
		hash = hex.EncodeToString(block.Hash(p.ChainCfg))
		log.Debug("recvBlkPs", "height", block.GetHeight(), "hash", hash)
		err = p.postBlockChain(hash, msg.receiveFrom.String(), block)

	} else if topic == psLtBlockTopic {
		lb := msg.value.(*types.LightBlock)
		p.ltB.addLtBlock(lb, msg.receiveFrom)
		log.Debug("recvBlkPs", "height", lb.GetHeader().GetHeight())

	} else if strings.HasPrefix(topic, psPeerMsgTopicPrefix) {
		err = p.handlePeerMsg(msg.value.(*types.PeerPubSubMsg), msg.receiveFrom)
	}

	if err != nil {
		log.Error("receivePs", "topic", topic, "hash", hash, "post msg err", err)
	}
}

func (p *broadcastProtocol) handlePeerMsg(msg *types.PeerPubSubMsg, pid peer.ID) error {
	var err error
	from := pid.String()
	switch msg.GetMsgID() {
	case blockReqMsgID:
		req := &types.ReqInt{}
		err = types.Decode(msg.ProtoMsg, req)
		log.Debug("recvBlkReq", "height", req.GetHeight(), "from", from)
		if err == nil {
			p.ltB.addBlockRequest(req.Height, pid)
		}
	case blockRespMsgID:
		block := &types.Block{}
		err = types.Decode(msg.ProtoMsg, block)
		if err != nil {
			log.Error("recvBlkResp", "decode err", err)
			break
		}
		hash := hex.EncodeToString(block.Hash(p.ChainCfg))
		log.Debug("recvBlkResp", "height", block.GetHeight(), "hash", hash, "from", from)
		err = p.postBlockChain(hash, from, block)
	default:
		err = types.ErrActionNotSupport
	}
	if err != nil {
		log.Error("handlePeerMsg", "msgID", msg.GetMsgID(), "err", err)
	}
	return err
}

func (p *broadcastProtocol) getPeerTopic(peerID peer.ID) string {
	return psPeerMsgTopicPrefix + peerID.String()
}

func (p *broadcastProtocol) pubPeerMsg(peerID peer.ID, msgID int32, msg types.Message) {
	pubMsg := &types.PeerPubSubMsg{
		MsgID:    msgID,
		ProtoMsg: types.Encode(msg),
	}
	p.ps.Pub(publishMsg{msg: pubMsg, topic: p.getPeerTopic(peerID)}, psBroadcast)
}

func (p *broadcastProtocol) postBlockChain(blockHash, pid string, block *types.Block) error {

	return p.P2PManager.PubBroadCast(blockHash, &types.BlockPid{Pid: pid, Block: block}, types.EventBroadcastAddBlock)
}

func (p *broadcastProtocol) postMempool(txHash string, tx *types.Transaction) error {

	return p.P2PManager.PubBroadCast(txHash, tx, types.EventTx)
}
