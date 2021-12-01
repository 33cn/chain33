// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package broadcast

import (
	"bytes"
	"context"
	"encoding/hex"
	"sync"
	"sync/atomic"

	"github.com/33cn/chain33/common/difficulty"
	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/peer"
	ps "github.com/libp2p/go-libp2p-pubsub"
)

type validator struct {
	*pubSub
	blkHeaderCache   map[int64]*types.Header
	maxRecvBlkHeight int64
	lock             sync.RWMutex
}

func newValidator(p *pubSub) *validator {
	v := &validator{pubSub: p}
	v.blkHeaderCache = make(map[int64]*types.Header)
	return v
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

func (v *validator) validateBlock(ctx context.Context, id peer.ID, msg *ps.Message) ps.ValidationResult {

	if id == v.Host.ID() {
		return ps.ValidationAccept
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

	v.lock.Lock()
	defer v.lock.Unlock()

	// 分叉区块，选择性广播
	if h, ok := v.blkHeaderCache[block.Height]; ok && bytes.Equal(h.ParentHash, block.ParentHash) {

		log.Debug("validateBlock", "recvForkBlk", block.GetHeight())
		// 区块时间小的优先
		if block.BlockTime > h.BlockTime {
			return ps.ValidationIgnore
		}
		// 难度系数高的优先
		if block.BlockTime == h.BlockTime &&
			difficulty.CalcWork(block.Difficulty).Cmp(difficulty.CalcWork(h.Difficulty)) < 0 {

			return ps.ValidationIgnore
		}
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

//
func (v *validator) validateTx(ctx context.Context, id peer.ID, msg *ps.Message) ps.ValidationResult {

	if id == v.Host.ID() {
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

	return ps.ValidationAccept
}
