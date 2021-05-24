// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package broadcast

import (
	"bytes"
	"context"
	"encoding/hex"
	"sync/atomic"

	"github.com/33cn/chain33/common/difficulty"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

func (p *pubSub) addBlockHeader(header *types.Header) {

	p.blkHeaderCache[header.Height] = header
	delete(p.blkHeaderCache, header.Height-blkHeaderCacheSize)
	// 如果丢包，可能遗留未删除历史数据，控制缓存大小
	if len(p.blkHeaderCache) >= 2*blkHeaderCacheSize {
		for _, h := range p.blkHeaderCache {
			if h.Height <= header.Height-blkHeaderCacheSize {
				delete(p.blkHeaderCache, h.Height)
			}
		}
	}
}

func (p *pubSub) validateBlock(ctx context.Context, id peer.ID, msg *pubsub.Message) pubsub.ValidationResult {

	if id == p.Host.ID() {
		return pubsub.ValidationAccept
	}

	block := &types.Block{}
	err := p.decodeMsg(msg.Data, nil, block)
	if err != nil {
		log.Error("validateBlock", "decodeMsg err", err)
		return pubsub.ValidationReject
	}

	blockHash := block.Hash(p.ChainCfg)
	blockHashHex := hex.EncodeToString(blockHash)
	//重复检测
	if p.blockFilter.AddWithCheckAtomic(blockHashHex, struct{}{}) {
		log.Debug("validateBlock", "recvDupBlk", block.GetHeight())
		return pubsub.ValidationIgnore
	}

	// 丢弃收到高度落后较多的区块
	maxHeight := atomic.LoadInt64(&p.maxRecvBlkHeight)
	if block.GetHeight() <= maxHeight-int64(blkHeaderCacheSize) {
		log.Debug("validateBlock", "recvHisBlk", block.GetHeight())
		return pubsub.ValidationReject
	}

	if block.GetHeight() > maxHeight {
		atomic.StoreInt64(&p.maxRecvBlkHeight, block.GetHeight())
	}

	p.lock.Lock()
	defer p.lock.Unlock()

	// 分叉区块，选择性广播
	if h, ok := p.blkHeaderCache[block.Height]; ok && bytes.Equal(h.ParentHash, block.ParentHash) {

		log.Debug("validateBlock", "recvForkBlk", block.GetHeight())
		// 区块时间小的优先
		if block.BlockTime > h.BlockTime {
			return pubsub.ValidationIgnore
		}
		// 难度系数高的优先
		if block.BlockTime == h.BlockTime &&
			difficulty.CalcWork(block.Difficulty).Cmp(difficulty.CalcWork(h.Difficulty)) < 0 {

			return pubsub.ValidationIgnore
		}
	} else {
		recvHeader := &types.Header{
			Hash:       blockHash,
			BlockTime:  block.BlockTime,
			Height:     block.Height,
			Difficulty: block.Difficulty,
			ParentHash: block.ParentHash,
		}
		p.addBlockHeader(recvHeader)
	}

	log.Debug("validateBlock", "height", block.GetHeight(), "fromPid", msg.ReceivedFrom.String(),
		"hash", blockHashHex)

	return pubsub.ValidationAccept
}

//
func (p *pubSub) validateTx(ctx context.Context, id peer.ID, msg *pubsub.Message) pubsub.ValidationResult {

	if id == p.Host.ID() {
		return pubsub.ValidationAccept
	}

	tx := &types.Transaction{}
	err := p.decodeMsg(msg.Data, nil, tx)
	if err != nil {
		log.Error("validateTx", "decodeMsg err", err)
		return pubsub.ValidationReject
	}

	//重复检测
	if p.txFilter.AddWithCheckAtomic(hex.EncodeToString(tx.Hash()), struct{}{}) {
		return pubsub.ValidationIgnore
	}

	return pubsub.ValidationAccept
}
