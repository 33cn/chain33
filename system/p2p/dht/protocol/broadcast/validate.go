// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package broadcast

import (
	"bytes"
	"container/list"
	"context"
	"encoding/hex"
	"sync"
	"sync/atomic"
	"time"

	"github.com/33cn/chain33/types"
	ps "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
)

type deniedInfo struct {
	freeTimestamp int64
	count         int
}

// validator 结合区块链业务逻辑, 实现pubsub数据验证接口
type validator struct {
	*pubSub
	blkHeaderCache   map[int64]*types.Header
	maxRecvBlkHeight int64
	headerLock       sync.Mutex
	deniedPeers      map[peer.ID]*deniedInfo //广播屏蔽节点
	peerLock         sync.RWMutex
	msgList          *list.List
	msgLock          sync.Mutex
	msgBuf           []*broadcastMsg
}

func newValidator(p *pubSub) *validator {
	v := &validator{pubSub: p}
	v.blkHeaderCache = make(map[int64]*types.Header)
	v.deniedPeers = make(map[peer.ID]*deniedInfo)
	v.msgBuf = make([]*broadcastMsg, 0, 1024)
	v.msgList = list.New()
	return v
}

func initValidator(p *pubSub) *validator {
	val := newValidator(p)
	go val.manageDeniedPeer()
	return val
}

// manageDeniedPeer  广播屏蔽节点管理
// 对每次接收的广播数据, 由blockchain或mempool校验后反馈结果,
// 如果出错则对始发节点进行若干时长屏蔽
// 单次错误影响不大, 总屏蔽时长根据错误次数不断累计, 主要限制错误广播频率
// 屏蔽时间结束后可恢复为正常节点
func (v *validator) manageDeniedPeer() {

	waitMsgReplyTicker := time.NewTicker(2 * time.Second)
	recoverDeniedPeerTicker := time.NewTicker(10 * time.Minute)
	for {

		select {
		case <-v.Ctx.Done():
			waitMsgReplyTicker.Stop()
			recoverDeniedPeerTicker.Stop()
			return

		case <-waitMsgReplyTicker.C:
			//将数据复制和等待反馈分离, 避免占用list导致广播模块处理逻辑阻塞
			v.copyMsgList()
			for _, bcMsg := range v.msgBuf {
				// 等待blockchain或mempool的广播校验结果
				msg, err := v.P2PEnv.QueueClient.Wait(bcMsg.msg)
				// 理论上不会出错, 只做日志记录
				if msg == nil || err != nil {
					log.Error("manageDeniedPeer", "wait msg err", err)
					continue
				}
				reply, ok := msg.Data.(*types.Reply)
				if !ok {
					continue
				}
				v.handleBroadcastReply(reply, bcMsg)

			}

		case <-recoverDeniedPeerTicker.C:
			v.recoverDeniedPeers()
		}
	}
}

const (
	//单位秒, 广播错误交易单次屏蔽时长
	errTxDenyTime = 10
	//单位秒, 广播错误区块单次屏蔽时长
	errBlockDenyTime = 60 * 60
)

func (v *validator) handleBroadcastReply(reply *types.Reply, msg *broadcastMsg) {

	if reply.IsOk {
		// 收到正确的广播, 减少错误次数
		v.reduceDeniedCount(msg.publisher)
		return
	}
	errMsg := string(reply.GetMsg())
	// 忽略系统性错误, 区块存在错误可能是广播和下载不协调导致, 可能误判, 暂不做处理
	if errMsg == types.ErrMemFull.Error() ||
		errMsg == types.ErrNotSync.Error() ||
		errMsg == types.ErrBlockExist.Error() ||
		errMsg == types.ErrHeightAlreadyFinalized.Error() {
		return
	}
	denyTime := int64(errBlockDenyTime)
	if msg.msg.Ty == types.EventTx {
		denyTime = errTxDenyTime
	}
	peerName := msg.publisher.Pretty()
	log.Debug("handleBcRep", "errMsg", errMsg, "hash", msg.hash, "peer", peerName)
	v.addDeniedPeer(msg.publisher, denyTime)
	// 尝试断开并拉黑处理
	if v.P2PEnv.Host != nil {
		v.P2PEnv.Host.Network().ClosePeer(msg.publisher)
		v.P2PEnv.ConnBlackList.Add(peerName, time.Hour*24)
	}
}

func (v *validator) reduceDeniedCount(id peer.ID) {

	v.peerLock.Lock()
	defer v.peerLock.Unlock()
	info, ok := v.deniedPeers[id]
	if ok && info.count > 0 {
		info.count--
	}
}

func (v *validator) addDeniedPeer(id peer.ID, denyTime int64) {
	v.peerLock.Lock()
	defer v.peerLock.Unlock()
	info, ok := v.deniedPeers[id]
	if !ok {
		info = &deniedInfo{}
		v.deniedPeers[id] = info
	}
	now := types.Now().Unix()
	if info.freeTimestamp < now {
		info.freeTimestamp = now
	}
	info.count++
	// 屏蔽时长随错误次数指数上升
	info.freeTimestamp += int64(1<<info.count) * denyTime
}

// 检测是否为屏蔽节点
func (v *validator) isDeniedPeer(id peer.ID) bool {
	v.peerLock.RLock()
	defer v.peerLock.RUnlock()
	info, ok := v.deniedPeers[id]
	return ok && info.freeTimestamp > types.Now().Unix()
}

// 恢复为正常节点
func (v *validator) recoverDeniedPeers() {
	v.peerLock.Lock()
	defer v.peerLock.Unlock()
	now := types.Now().Unix()

	for id, info := range v.deniedPeers {
		log.Debug("recoverDeniedPeers", "peer", id.Pretty(), "count", info.count, "freeTime", info.freeTimestamp)
		if info.count <= 0 && info.freeTimestamp <= now {
			delete(v.deniedPeers, id)
		}
	}
}

func (v *validator) addBroadcastMsg(msg *broadcastMsg) {
	v.msgLock.Lock()
	defer v.msgLock.Unlock()
	v.msgList.PushBack(msg)
}

// 拷贝列表
func (v *validator) copyMsgList() {
	v.msgLock.Lock()
	defer v.msgLock.Unlock()
	v.msgBuf = v.msgBuf[:0]
	for it := v.msgList.Front(); it != nil; it = it.Next() {
		v.msgBuf = append(v.msgBuf, it.Value.(*broadcastMsg))
	}
	v.msgList.Init()
}

func (v *validator) addBlockHeader(header *types.Header) {

	v.blkHeaderCache[header.Height] = header
	delete(v.blkHeaderCache, header.Height-blkHeaderCacheSize)
	// 如果丢包，可能遗留未删除历史数据，控制缓存大小
	if len(v.blkHeaderCache) >= 2*blkHeaderCacheSize {
		for _, h := range v.blkHeaderCache {
			if h.Height <= header.Height-blkHeaderCacheSize {
				delete(v.blkHeaderCache, h.Height)
			}
		}
	}
}

func (v *validator) validateBlock(ctx context.Context, _ peer.ID, msg *ps.Message) ps.ValidationResult {

	id := msg.GetFrom()
	if id == v.Host.ID() {
		return ps.ValidationAccept
	}

	if v.isDeniedPeer(id) {
		log.Debug("validateBlock", "denied peer", id.Pretty())
		return ps.ValidationReject
	}

	block := &types.Block{}
	err := v.decodeMsg(msg.Data, nil, block)
	if err != nil {
		log.Error("validateBlock", "decodeMsg err", err)
		return ps.ValidationReject
	}

	blockHash := block.Hash(v.ChainCfg)
	blockHashHex := hex.EncodeToString(blockHash)
	//重复检测
	if v.blockFilter.AddWithCheckAtomic(blockHashHex, struct{}{}) {
		log.Debug("validateBlock", "recvDupBlk", block.GetHeight())
		return ps.ValidationIgnore
	}

	// 丢弃收到高度落后较多的区块
	maxHeight := atomic.LoadInt64(&v.maxRecvBlkHeight)
	if block.GetHeight() <= maxHeight-int64(blkHeaderCacheSize) {
		log.Debug("validateBlock", "recvHisBlk", block.GetHeight())
		return ps.ValidationReject
	}

	if block.GetHeight() > maxHeight {
		atomic.StoreInt64(&v.maxRecvBlkHeight, block.GetHeight())
	}

	v.headerLock.Lock()
	defer v.headerLock.Unlock()

	// 分叉区块，选择性广播
	if h, ok := v.blkHeaderCache[block.Height]; ok && bytes.Equal(h.ParentHash, block.ParentHash) {

		log.Debug("validateBlock", "recvForkBlk", block.GetHeight())
		// 区块时间小的优先
		//if block.BlockTime > h.BlockTime {
		//	return ps.ValidationIgnore
		//}
		//// 难度系数高的优先
		//if block.BlockTime == h.BlockTime &&
		//	difficulty.CalcWork(block.Difficulty).Cmp(difficulty.CalcWork(h.Difficulty)) < 0 {
		//
		//	return ps.ValidationIgnore
		//}
	} else {
		recvHeader := &types.Header{
			Hash:       blockHash,
			BlockTime:  block.BlockTime,
			Height:     block.Height,
			Difficulty: block.Difficulty,
			ParentHash: block.ParentHash,
		}
		v.addBlockHeader(recvHeader)
	}

	log.Debug("validateBlock", "height", block.GetHeight(), "fromPid", msg.ReceivedFrom.String(),
		"hash", blockHashHex)

	return ps.ValidationAccept
}

func (v *validator) validatePeer(ctx context.Context, _ peer.ID, msg *ps.Message) ps.ValidationResult {
	id := msg.GetFrom()
	if v.isDeniedPeer(id) {
		log.Debug("validatePeer", "topic", *msg.Topic, "denied peer", id.Pretty())
		return ps.ValidationReject
	}
	return ps.ValidationAccept
}

func isTxErr(err error) bool {
	return (err != types.ErrMemFull) && (err != types.ErrNotSync)
}

func (v *validator) validateTx(ctx context.Context, id peer.ID, msg *ps.Message) ps.ValidationResult {

	from := msg.GetFrom()
	if from == v.Host.ID() {
		return ps.ValidationAccept
	}

	tx := &types.Transaction{}
	err := v.decodeMsg(msg.Data, nil, tx)
	if err != nil {
		log.Error("validateTx", "decodeMsg err", err)
		return ps.ValidationReject
	}

	//重复检测
	if v.txFilter.AddWithCheckAtomic(hex.EncodeToString(tx.Hash()), struct{}{}) {
		return ps.ValidationIgnore
	}

	_, err = v.API.SendTx(tx)
	if err != nil {
		if isTxErr(err) {
			log.Debug("validateTx", "hash", hex.EncodeToString(tx.Hash()), "err", err)
		}
		return ps.ValidationIgnore
	}

	return ps.ValidationAccept
}

func (v *validator) validateBatchTx(ctx context.Context, id peer.ID, msg *ps.Message) ps.ValidationResult {

	from := msg.GetFrom()
	if from == v.Host.ID() {
		return ps.ValidationAccept
	}

	txs := &types.Transactions{}
	err := v.decodeMsg(msg.Data, nil, txs)
	if err != nil {
		log.Error("validateBatchTx", "decodeMsg err", err)
		return ps.ValidationReject
	}

	validTxCount := 0

	for _, tx := range txs.GetTxs() {
		//重复检测
		if v.txFilter.AddWithCheckAtomic(hex.EncodeToString(tx.Hash()), struct{}{}) {
			continue
		}

		_, err = v.API.SendTx(tx)
		if err != nil {
			if isTxErr(err) {
				log.Debug("validateBatchTx", "hash", hex.EncodeToString(tx.Hash()), "err", err)
			}
			continue
		}
		validTxCount++
	}

	if validTxCount > len(txs.GetTxs())/2 {
		return ps.ValidationAccept
	}

	return ps.ValidationIgnore
}
