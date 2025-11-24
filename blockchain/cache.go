// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"fmt"
	"sync"

	"github.com/33cn/chain33/types"
)

const defaultBlockHashCacheSize = 10000 //区块哈希缓存个数暂时采用固定值，约0.4MB内存占用

// BlockCache 区块缓存
type BlockCache struct {
	hashCache     map[int64]string              //高度->区块哈希
	blockCache    map[string]*types.BlockDetail //区块哈希->区块数据
	blkCacheSize  int64
	hashCacheSize int64
	cacheLock     sync.RWMutex
	//cacheQueue    *list.List
	currentHeight int64 //当前区块高度
	cfg           *types.Chain33Config
}

// newBlockCache new
func newBlockCache(cfg *types.Chain33Config, blockHashCacheSize int64) *BlockCache {
	blkCacheSize := cfg.GetModuleConfig().BlockChain.DefCacheSize
	if blkCacheSize > blockHashCacheSize {
		panic(fmt.Sprintf("newBlockCache: block cache size config must less than hash cache size, "+
			"blkCacheSize=%d, blkHashCacheSize=%d", blkCacheSize, blockHashCacheSize))
	}
	return &BlockCache{
		hashCache:     make(map[int64]string, blockHashCacheSize),
		blockCache:    make(map[string]*types.BlockDetail, blkCacheSize),
		blkCacheSize:  blkCacheSize,
		hashCacheSize: blockHashCacheSize,
		cfg:           cfg,
	}
}

// GetBlockHash get block hash by height
func (bc *BlockCache) GetBlockHash(height int64) []byte {
	bc.cacheLock.RLock()
	defer bc.cacheLock.RUnlock()
	if hash, exist := bc.hashCache[height]; exist {
		return []byte(hash)
	}
	return nil
}

// GetBlockByHeight 从cache缓存中获取block信息
func (bc *BlockCache) GetBlockByHeight(height int64) *types.BlockDetail {
	bc.cacheLock.RLock()
	defer bc.cacheLock.RUnlock()

	hash, ok := bc.hashCache[height]
	if !ok {
		return nil
	}
	if blk, ok := bc.blockCache[hash]; ok {
		return blk
	}
	return nil
}

// GetBlockByHash 不做移动，cache最后的 128个区块
func (bc *BlockCache) GetBlockByHash(hash []byte) (block *types.BlockDetail) {
	bc.cacheLock.RLock()
	defer bc.cacheLock.RUnlock()
	if blk, ok := bc.blockCache[string(hash)]; ok {
		return blk
	}
	return nil
}

// AddBlock 区块增长，添加block到cache中，方便快速查询
func (bc *BlockCache) AddBlock(detail *types.BlockDetail) {
	bc.cacheLock.Lock()
	defer bc.cacheLock.Unlock()
	if bc.currentHeight > 0 && detail.Block.Height != bc.currentHeight+1 {
		chainlog.Error("...cacheBlock not continue...")
		if types.Debug {
			panic("...cacheBlock not continue...")
		}
	}

	if len(detail.Receipts) == 0 && len(detail.Block.Txs) != 0 {
		chainlog.Debug("cacheBlock  Receipts == 0", "height", detail.Block.GetHeight())
	}
	bc.currentHeight = detail.Block.Height
	blkHash := string(detail.Block.Hash(bc.cfg))
	// remove oldest key
	delete(bc.hashCache, bc.currentHeight-bc.hashCacheSize)
	delete(bc.blockCache, bc.hashCache[bc.currentHeight-bc.blkCacheSize])

	// add new key
	bc.hashCache[bc.currentHeight] = blkHash
	bc.blockCache[blkHash] = detail
}

// DelBlock 区块回滚，删除block
func (bc *BlockCache) DelBlock(height int64) {
	bc.cacheLock.Lock()
	defer bc.cacheLock.Unlock()
	if bc.currentHeight > 0 && height != bc.currentHeight {
		chainlog.Error("...del cacheBlock not continue...")
		if types.Debug {
			panic("...del cacheBlock not continue...")
		}
	}
	bc.currentHeight = height - 1
	// remove block first
	delete(bc.blockCache, bc.hashCache[height])
	delete(bc.hashCache, height)

}

type txHeightCacheType interface {
	// Add add block
	Add(block *types.Block)
	// Del del block
	Del(height int64)
	// Contains check key if exist
	Contains(key []byte, txHeight int64) bool
}

const txHashCacheLen = 16

// txHashCache 缓存txHeight功能的交易哈希
// 对于区块，只关心允许打包txHeight区间内的所有交易
// 对于单笔交易，只需检查相同txHeight值的交易是否存在重复
// 缓存采用双层map结构，交易按照txHeight值分类，提高map查重效率
// 占用空间， (lowerTxHeightRange+upperTxHeightRange) * 单个区块平均交易数 * 16 (byte)
// 1000 * 10000 * 16/1024/1024 = 150MB
// 不考虑回滚处理，缓存空间进一步优化减少，TODO：实现上区分区块链是否可能回滚
type txHashCache struct {
	currBlockHeight    int64
	lowerTxHeightRange int64                         //区块允许的下限txHeight范围
	upperTxHeightRange int64                         //区块允许的上限txHeight范围
	txHashes           map[int64]map[string]struct{} // [txHeight => [txHash=> struct]]
	lock               *sync.RWMutex
	chain              *BlockChain
}

// newTxHashCache new tx height cache type
func newTxHashCache(chain *BlockChain, lowerTxHeightRange, upperTxHeightRange int64) *txHashCache {
	return &txHashCache{
		txHashes:           make(map[int64]map[string]struct{}, lowerTxHeightRange+upperTxHeightRange),
		lock:               &sync.RWMutex{},
		lowerTxHeightRange: lowerTxHeightRange,
		upperTxHeightRange: upperTxHeightRange,
		chain:              chain,
	}
}

// Add :区块增长，缓存txHeight类交易的哈希, 缓存窗口前进一个高度
func (tc *txHashCache) Add(block *types.Block) {
	tc.lock.Lock()
	defer tc.lock.Unlock()

	tc.currBlockHeight = block.GetHeight()
	tc.addTxList(block.Txs)
	delHeight := tc.currBlockHeight - tc.upperTxHeightRange - tc.lowerTxHeightRange
	if delHeight >= 0 {
		delBlock, err := tc.chain.GetBlock(delHeight)
		if err != nil {
			//获取区块出错，将导致缓存异常
			chainlog.Crit("txHashCache Add", "height", delHeight, "Get del Block err", err)
		} else {
			//对于新区快，所有txHeight<=tc.currBlockHeight - tc.lowerTxHeightRange都已经过期，可以从缓存中删除
			//但批量删除可能导致区块回滚时再重新添加这些交易不好定位，因为这些交易可能分布在多个区块中, 这里只删除一个区块的交易
			tc.delTxList(delBlock.Block.Txs)
		}
	}
}

// Del 回滚区块时是Add的逆向处理, 缓存窗口后退一个高度
func (tc *txHashCache) Del(height int64) {

	tc.lock.Lock()
	defer tc.lock.Unlock()
	tc.currBlockHeight = height - 1
	//窗口向左移动一个高度，需要添加窗口中第一个区块内交易哈希
	addHeight := height - tc.upperTxHeightRange - tc.lowerTxHeightRange
	if addHeight >= 0 {
		addBlock, err := tc.chain.GetBlock(addHeight)
		if err != nil {
			//获取区块出错，将导致缓存异常
			chainlog.Crit("txHashCache Del", "height", height, "Get del Block err", err)
			return
		}
		tc.addTxList(addBlock.Block.Txs)
	}

	delBlock, err := tc.chain.GetBlock(height)
	if err != nil {
		//获取区块出错，将导致缓存异常
		chainlog.Crit("txHashCache Del", "height", height, "Get del Block err", err)
		return
	}
	tc.delTxList(delBlock.Block.Txs)
}

func (tc *txHashCache) addTxList(txs []*types.Transaction) {

	//只需要将expire为txHeight类型缓存
	for _, tx := range txs {
		txHeight := types.GetTxHeight(tc.chain.client.GetConfig(), tx.Expire, tc.currBlockHeight)
		if txHeight < 0 {
			continue
		}
		hashMap, ok := tc.txHashes[txHeight]
		if !ok {
			//预分配一定长度空间
			hashMap = make(map[string]struct{}, 2<<10)
			tc.txHashes[txHeight] = hashMap
		}
		hashMap[string(tx.Hash()[:txHashCacheLen])] = struct{}{}

	}
}

func (tc *txHashCache) delTxList(txs []*types.Transaction) {

	for _, tx := range txs {
		txHeight := types.GetTxHeight(tc.chain.client.GetConfig(), tx.Expire, tc.currBlockHeight)
		if hashMap, ok := tc.txHashes[txHeight]; ok {
			delete(hashMap, string(tx.Hash()[:txHashCacheLen]))
			if len(hashMap) == 0 {
				delete(tc.txHashes, txHeight)
			}
		}
	}
}

// Contains 缓存中是否包含该交易
func (tc *txHashCache) Contains(hash []byte, txHeight int64) bool {
	tc.lock.RLock()
	defer tc.lock.RUnlock()

	if hashMap, ok := tc.txHashes[txHeight]; ok {
		_, exist := hashMap[string(hash[:txHashCacheLen])]
		return exist
	}
	return false
}

// txHeight未开启情况，兼容blockchain调用，实现txHeightCacheType接口
type noneCache struct{}

func (n *noneCache) Add(*types.Block)            {}
func (n *noneCache) Del(int64)                   {}
func (n *noneCache) Contains([]byte, int64) bool { return false }
