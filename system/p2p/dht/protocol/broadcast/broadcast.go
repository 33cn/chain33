// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package broadcast broadcast protocol
package broadcast

import (
	"context"
	"encoding/hex"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/33cn/chain33/system/p2p/dht/extension"

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

const (
	broadcastV1 = "/chain33/p2p/broadcast/1.0.0"
)

func init() {
	protocol.RegisterProtocolInitializer(InitProtocol)
}

//
type broadcastProtocol struct {
	*protocol.P2PEnv

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

	syncStatus bool
	currHeight int64
	lock       sync.RWMutex
}

// InitProtocol init protocol
func InitProtocol(env *protocol.P2PEnv) {
	new(broadcastProtocol).init(env)
}

func (p *broadcastProtocol) init(env *protocol.P2PEnv) {
	p.P2PEnv = env
	//接收交易和区块过滤缓存, 避免重复提交到mempool或blockchain
	p.txFilter = utils.NewFilter(txRecvFilterCacheNum)
	p.blockFilter = utils.NewFilter(blockRecvFilterCacheNum)

	p.ps = pubsub.NewPubSub(1024)
	// 单独复制一份， 避免data race
	subCfg := *(env.SubConfig)
	p.p2pCfg = &subCfg

	//注册事件处理函数
	protocol.RegisterEventHandler(types.EventTxBroadcast, p.handleBroadcastSend)
	protocol.RegisterEventHandler(types.EventBlockBroadcast, p.handleBroadcastSend)
	protocol.RegisterEventHandler(types.EventIsSync, p.handleIsSyncEvent)
	protocol.RegisterEventHandler(types.EventAddBlock, p.handleAddBlock)

	// pub sub broadcast
	newPubSub(p).broadcast()
	newLtBroadcast(p).broadcast()
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
		if block.Size() > int(p.SubConfig.MinLtBlockSize) &&
			len(block.Txs) > 0 {
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
		p.ps.Pub(psMsg{msg: broadcastData, topic: topic}, psBroadcast)
	}
}

func (p *broadcastProtocol) buildLtBlock(block *types.Block) *types.LightBlock {

	ltBlock := &types.LightBlock{}
	ltBlock.Header = block.GetHeader(p.ChainCfg)
	ltBlock.Header.Signature = block.Signature
	ltBlock.MinerTx = block.Txs[0]
	for _, tx := range block.Txs[1:] {
		//tx short hash
		ltBlock.STxHashes = append(ltBlock.STxHashes, types.CalcTxShortHash(tx.Hash()))
	}
	return ltBlock
}

func (p *broadcastProtocol) handleBroadcastReceive(topic string, msg types.Message, rawData extension.SubMsg) {

	var err error
	var hash string
	// 将接收的交易或区块 转发到内部对应模块

	if topic == psTxTopic {
		tx := msg.(*types.Transaction)
		hash = hex.EncodeToString(tx.Hash())
		err = p.postMempool(hash, msg.(*types.Transaction))

	} else if topic == psBlockTopic {
		block := msg.(*types.Block)
		hash = hex.EncodeToString(block.Hash(p.ChainCfg))
		log.Debug("recvBlkPs", "height", block.GetHeight(), "hash", hash, "from", rawData.ReceivedFrom.String())
		err = p.postBlockChain(hash, rawData.ReceivedFrom.String(), block)

	} else if topic == psLtBlockTopic {

	} else if strings.HasPrefix(topic, psPeerTopicPrefix) {

	}

	if err != nil {
		log.Error("receivePs", "topic", topic, "hash", hash, "post msg err", err)
	}
}

func (p *broadcastProtocol) pubPeerMsg(peerID peer.ID, msgID int32, msg types.Message) {
	pubMsg := &types.PeerPubSubMsg{
		MsgID:    msgID,
		ProtoMsg: types.Encode(msg),
	}
	peerTopic := psPeerTopicPrefix + peerID.String()
	p.ps.Pub(psMsg{msg: pubMsg, topic: peerTopic}, psBroadcast)
}

func (p *broadcastProtocol) postBlockChain(blockHash, pid string, block *types.Block) error {

	return p.P2PManager.PubBroadCast(blockHash, &types.BlockPid{Pid: pid, Block: block}, types.EventBroadcastAddBlock)
}

func (p *broadcastProtocol) postMempool(txHash string, tx *types.Transaction) error {

	return p.P2PManager.PubBroadCast(txHash, tx, types.EventTx)
}
