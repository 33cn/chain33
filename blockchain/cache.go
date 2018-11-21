// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"container/list"
	"sync"

	"github.com/33cn/chain33/types"
)

//BlockCache 区块缓存
type BlockCache struct {
	cache      map[int64]*list.Element
	cacheHash  map[string]*list.Element
	cacheTxs   map[string]bool
	cacheSize  int64
	cachelock  sync.Mutex
	cacheQueue *list.List
	maxHeight  int64 //用来辅助判断cache 是否正确
}

//NewBlockCache new
func NewBlockCache(defCacheSize int64) *BlockCache {
	return &BlockCache{
		cache:      make(map[int64]*list.Element),
		cacheHash:  make(map[string]*list.Element),
		cacheTxs:   make(map[string]bool),
		cacheSize:  defCacheSize,
		cacheQueue: list.New(),
		maxHeight:  0,
	}
}

//CheckcacheBlock 从cache缓存中获取block信息
func (chain *BlockCache) CheckcacheBlock(height int64) (block *types.BlockDetail) {
	chain.cachelock.Lock()
	defer chain.cachelock.Unlock()

	elem, ok := chain.cache[height]
	if ok {
		// Already exists. Move to back of cacheQueue.
		chain.cacheQueue.MoveToBack(elem)
		return elem.Value.(*types.BlockDetail)
	}
	return nil
}

//GetCacheBlock 不做移动，cache最后的 128个区块
func (chain *BlockCache) GetCacheBlock(hash []byte) (block *types.BlockDetail) {
	chain.cachelock.Lock()
	defer chain.cachelock.Unlock()
	elem, ok := chain.cacheHash[string(hash)]
	if ok {
		return elem.Value.(*types.BlockDetail)
	}
	return nil
}

//HasCacheTx 缓存中是否包含该交易
func (chain *BlockCache) HasCacheTx(hash []byte) bool {
	chain.cachelock.Lock()
	defer chain.cachelock.Unlock()
	_, ok := chain.cacheTxs[string(hash)]
	return ok
}

//添加block到cache中，方便快速查询
func (chain *BlockCache) cacheBlock(blockdetail *types.BlockDetail) {
	chain.cachelock.Lock()
	defer chain.cachelock.Unlock()
	if chain.maxHeight > 0 && blockdetail.Block.Height != chain.maxHeight+1 {
		chainlog.Error("...cacheBlock not continue...")
		if types.Debug {
			panic("...cacheBlock not continue...")
		}
	}
	chain.maxHeight = blockdetail.Block.Height
	if len(blockdetail.Receipts) == 0 && len(blockdetail.Block.Txs) != 0 {
		chainlog.Debug("cacheBlock  Receipts == 0", "height", blockdetail.Block.GetHeight())
	}
	chain.addCacheBlock(blockdetail)

	// Maybe expire an item.
	if int64(chain.cacheQueue.Len()) > chain.cacheSize {
		blockdetail := chain.cacheQueue.Remove(chain.cacheQueue.Front()).(*types.BlockDetail)
		chain.delCacheBlock(blockdetail)
	}
}

//添加block到cache中，方便快速查询
func (chain *BlockCache) delBlockFromCache(height int64) {
	chain.cachelock.Lock()
	defer chain.cachelock.Unlock()
	if chain.maxHeight > 0 && height != chain.maxHeight {
		chainlog.Error("...del cacheBlock not continue...")
		if types.Debug {
			panic("...del cacheBlock not continue...")
		}
	}
	chain.maxHeight = height - 1
	elem, ok := chain.cache[height]
	if ok {
		blockdetail := chain.cacheQueue.Remove(elem).(*types.BlockDetail)
		chain.delCacheBlock(blockdetail)
	}
}

func (chain *BlockCache) addCacheBlock(blockdetail *types.BlockDetail) {
	// Create entry in cache and append to cacheQueue.
	elem := chain.cacheQueue.PushBack(blockdetail)
	chain.cache[blockdetail.Block.Height] = elem
	chain.cacheHash[string(blockdetail.Block.Hash())] = elem
	for _, tx := range blockdetail.Block.Txs {
		chain.cacheTxs[string(tx.Hash())] = true
	}
}

func (chain *BlockCache) delCacheBlock(blockdetail *types.BlockDetail) {
	delete(chain.cache, blockdetail.Block.Height)
	delete(chain.cacheHash, string(blockdetail.Block.Hash()))
	for _, tx := range blockdetail.Block.Txs {
		delete(chain.cacheTxs, string(tx.Hash()))
	}
}
