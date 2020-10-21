// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package broadcast broadcast protocol
package broadcast

import (
	"math/rand"
	"sync"
	"time"

	"github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/common/pubsub"
	"github.com/33cn/chain33/p2p/utils"
	"github.com/33cn/chain33/system/p2p/dht/protocol"
	p2pty "github.com/33cn/chain33/system/p2p/dht/types"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/peer"
)

var log = log15.New("module", "p2p.broadcast")

const (
	broadcastV1          = "/chain33/p2p/broadcast/1.0.0"
	broadcastTransaction = "/chain33/broadcast/transaction/1.0.0"
	broadcastBlock       = "/chain33/broadcast/block/1.0.0"
)

// Protocol ...
type Protocol struct {
	*protocol.P2PEnv

	//接收交易和区块过滤缓存, 避免重复提交到mempool或blockchain
	txFilter    *utils.Filterdata
	blockFilter *utils.Filterdata
	//发送交易和区块时过滤缓存, 解决冗余广播发送
	txSendFilter    *utils.Filterdata
	blockSendFilter *utils.Filterdata
	ltBlockCache    *utils.SpaceLimitCache
	p2pCfg          *p2pty.P2PSubConfig
	ps              *pubsub.PubSub

	txQueue    chan *types.Transaction
	blockQueue chan *types.Block
	peers      []peer.ID
	peerMutex  sync.RWMutex
}

var defaultProtocol *Protocol

func init() {
	protocol.RegisterProtocolInitializer(InitProtocol)
}

// InitProtocol init protocol
func InitProtocol(env *protocol.P2PEnv) {
	p := &Protocol{
		P2PEnv:          env,
		txFilter:        utils.NewFilter(txRecvFilterCacheNum),
		blockFilter:     utils.NewFilter(blockRecvFilterCacheNum),
		txSendFilter:    utils.NewFilter(txSendFilterCacheNum),
		blockSendFilter: utils.NewFilter(blockSendFilterCacheNum),
		ps:              pubsub.NewPubSub(10000),
		txQueue:         make(chan *types.Transaction, 10000),
		blockQueue:      make(chan *types.Block, 100),
	}
	defaultProtocol = p
	// 单独复制一份， 避免data race
	subCfg := *(env.SubConfig)
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
	p.ltBlockCache = utils.NewSpaceLimitCache(ltBlockCacheNum, int(subCfg.LtBlockCacheSize*1024*1024))
	p.p2pCfg = &subCfg

	// old是为兼容老版本的临时版本
	protocol.RegisterStreamHandler(p.Host, broadcastV1, p.handleStreamOld)
	//protocol.RegisterStreamHandler(p.Host, broadcastTransaction, p.handleStreamBroadcastTx)
	//protocol.RegisterStreamHandler(p.Host, broadcastBlock, p.handleStreamBroadcastBlock)
	////注册事件处理函数
	protocol.RegisterEventHandler(types.EventTxBroadcast, p.handleEventBroadcastTx)
	protocol.RegisterEventHandler(types.EventBlockBroadcast, p.handleEventBroadcastBlock)
	go p.broadcastRoutine()
	// pub sub broadcast
	go newPubSub(p).broadcast()
}

func (p *Protocol) broadcastRoutine() {
	for p.RoutingTable.Size() == 0 {
		time.Sleep(time.Second / 10)
	}
	p.refreshPeers()
	ticker1 := time.NewTicker(time.Minute)
	for {
		select {
		case <-p.Ctx.Done():
			return
		case <-ticker1.C:
			p.refreshPeers()
		case tx := <-p.txQueue:
			p.broadcastOld(tx)
		case block := <-p.blockQueue:
			p.broadcastOld(block)
		}
	}
}

func (p *Protocol) refreshPeers() {
	pids := p.RoutingTable.ListPeers()
	//最多广播20个节点
	if len(pids) > 20 {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		r.Shuffle(len(pids), func(i, j int) {
			pids[i], pids[j] = pids[j], pids[i]
		})
		pids = pids[:20]
	}

	p.peerMutex.Lock()
	p.peers = pids
	p.peerMutex.Unlock()
}

func (p *Protocol) getBroadcastPeers() []peer.ID {
	p.peerMutex.RLock()
	defer p.peerMutex.RUnlock()
	var peers []peer.ID
	peers = append(peers, p.peers...)
	return peers
}

func (p *Protocol) broadcastOld(in interface{}) {
	var data types.MessageBroadCast
	switch v := in.(type) {
	case *types.Transaction:
		data.Message = &types.BroadCastData{
			Value: &types.BroadCastData_Tx{
				Tx: &types.P2PTx{
					Tx: v,
				},
			},
		}
	case *types.Block:
		data.Message = &types.BroadCastData{
			Value: &types.BroadCastData_Block{
				Block: &types.P2PBlock{
					Block: v,
				},
			},
		}
	default:
		return
	}

	for _, pid := range p.getBroadcastPeers() {
		err := p.sendDataToPeer(&data, pid)
		if err != nil {
			log.Error("broadcastTxOld", "pid", pid, "error", err)
		}
	}
}

func (p *Protocol) sendDataToPeer(data *types.MessageBroadCast, pid peer.ID) error {
	stream, err := p.Host.NewStream(p.Ctx, pid, broadcastV1)
	if err != nil {
		return err
	}
	defer protocol.CloseStream(stream)
	return protocol.WriteStream(data, stream)
}

//func (p *Protocol) broadcastBlock() {
//	block := <-p.blockQueue
//	for _, pid := range p.getBroadcastPeers() {
//		err := p.sendBlockToPeer(block, pid)
//		if err != nil {
//			log.Error("broadcastTransaction", "error", err, "pid", pid)
//		}
//	}
//}
//
//func (p *Protocol) sendBlockToPeer(block *types.Block, pid peer.ID) error {
//	stream, err := p.Host.NewStream(p.Ctx, pid, broadcastBlock)
//	if err != nil {
//		return err
//	}
//	defer protocol.CloseStream(stream)
//	return protocol.WriteStream(block, stream)
//}
//
//func (p *Protocol) broadcastTransaction() {
//	txs := loadTransaction(p.txQueue, 100)
//	for _, pid := range p.getBroadcastPeers() {
//		err := p.sendTransactionToPeer(txs, pid)
//		if err != nil {
//			log.Error("broadcastTransaction", "error", err, "pid", pid)
//		}
//	}
//}
//
//func (p *Protocol) sendTransactionToPeer(txs *types.Transactions, pid peer.ID) error {
//	stream, err := p.Host.NewStream(p.Ctx, pid, broadcastTransaction)
//	if err != nil {
//		return err
//	}
//	defer protocol.CloseStream(stream)
//	return protocol.WriteStream(txs, stream)
//}
//
//func loadTransaction(q chan *types.Transaction, count int) *types.Transactions {
//	var txs types.Transactions
//	for {
//		select {
//		case tx := <-q:
//			txs.Txs = append(txs.Txs, tx)
//			if len(txs.Txs) >= count {
//				return &txs
//			}
//		default:
//			return &txs
//		}
//	}
//}

func (p *Protocol) postBlockChain(blockHash, pid string, block *types.Block) error {
	return p.P2PManager.PubBroadCast(blockHash, &types.BlockPid{Pid: pid, Block: block}, types.EventBroadcastAddBlock)
}

func (p *Protocol) postMempool(txHash string, tx *types.Transaction) error {
	return p.P2PManager.PubBroadCast(txHash, tx, types.EventTx)
}

type sendFilterInfo struct {
	//记录广播交易或区块时需要忽略的节点, 这些节点可能是交易的来源节点,也可能节点间维护了多条连接, 冗余发送
	ignoreSendPeers map[peer.ID]bool
}

//检测是否冗余发送, 或者添加到发送过滤(内部存在直接修改读写保护的数据, 对filter lru的读写需要外层锁保护)
func addIgnoreSendPeerAtomic(filter *utils.Filterdata, key string, pid peer.ID) (exist bool) {

	filter.GetAtomicLock()
	defer filter.ReleaseAtomicLock()
	var info *sendFilterInfo
	if !filter.Contains(key) { //之前没有收到过这个key
		info = &sendFilterInfo{ignoreSendPeers: make(map[peer.ID]bool)}
		filter.Add(key, info)
	} else {
		data, _ := filter.Get(key)
		info = data.(*sendFilterInfo)
	}
	_, exist = info.ignoreSendPeers[pid]
	info.ignoreSendPeers[pid] = true
	return exist
}
