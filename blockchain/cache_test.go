// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"testing"

	"github.com/33cn/chain33/common/crypto"
	"github.com/33cn/chain33/queue"

	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/stretchr/testify/require"
)

func TestBlockCache(t *testing.T) {

	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	_, priv := util.Genaddress()
	cacheSize := int64(5)
	blockHashCacheSize := int64(10)
	cfg.GetModuleConfig().BlockChain.DefCacheSize = cacheSize
	cache := newBlockCache(cfg, blockHashCacheSize)
	currHeight := int64(-1)
	// add 20 block
	for i := 0; i < int(blockHashCacheSize+2*cacheSize); i++ {
		block := util.CreateCoinsBlock(cfg, priv, 1)
		block.Height = int64(i)
		cache.AddBlock(&types.BlockDetail{Block: block})
		currHeight++
	}
	// rollback 4 block
	for i := 0; i < int(cacheSize)-1; i++ {
		cache.DelBlock(currHeight)
		currHeight--
	}

	require.Equal(t, int64(15), currHeight)
	require.Equal(t, 32, len(cache.GetBlockHash(currHeight)))
	require.Equal(t, 0, len(cache.GetBlockHash(currHeight+1)))

	require.Equal(t, 6, len(cache.hashCache))
	require.Equal(t, 1, len(cache.blockCache))
	require.Equal(t, currHeight, cache.currentHeight)
	require.Equal(t, 0, len(cache.GetBlockHash(9)))
	require.Equal(t, 32, len(cache.GetBlockHash(10)))
	require.NotNil(t, cache.GetBlockByHeight(currHeight))
	require.Nil(t, cache.GetBlockByHeight(currHeight-1))
	require.NotNil(t, cache.GetBlockByHash(cache.GetBlockHash(currHeight)))
}

func initTxHashCache(txHeightRange int64) (*txHashCache, *BlockChain, *types.Chain33Config) {
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	cfg.GetModuleConfig().BlockChain.DefCacheSize = txHeightRange * 4
	chain := New(cfg)
	chain.blockStore = &BlockStore{blockCache: chain.blockCache}
	q := queue.New("channel")
	q.SetConfig(cfg)
	chain.client = q.Client()
	cache := newTxHashCache(chain, txHeightRange, txHeightRange)
	chain.txHeightCache = cache
	return cache, chain, cfg

}

func newTestTxs(cfg *types.Chain33Config, priv crypto.PrivKey, currHeight, txHeightRange int64) []*types.Transaction {

	txs := util.GenNoneTxs(cfg, priv, 4)
	//分别设置，最低txHeight， 最高txHeight， 正常txHeight，非txHeight交易
	util.UpdateExpireWithTxHeight(txs[0], priv, currHeight-txHeightRange)
	util.UpdateExpireWithTxHeight(txs[1], priv, currHeight)
	util.UpdateExpireWithTxHeight(txs[2], priv, currHeight+txHeightRange)
	return txs
}

func addTestBlock(chain *BlockChain, cfg *types.Chain33Config, priv crypto.PrivKey, startHeight, txHeightRange int64, blockNum int) {

	for i := 0; i < blockNum; i++ {
		txs := newTestTxs(cfg, priv, startHeight, txHeightRange)
		block := &types.BlockDetail{Block: &types.Block{Txs: txs, Height: startHeight}}
		chain.blockCache.AddBlock(block)
		chain.txHeightCache.Add(block.GetBlock())
		startHeight++
	}
}

func TestTxHashCache_Add(t *testing.T) {

	var txHeightRange int64 = 5
	tc, chain, cfg := initTxHashCache(txHeightRange)
	_, priv := util.Genaddress()

	addTestBlock(chain, cfg, priv, 0, txHeightRange, 30)
	currHeight := tc.currBlockHeight
	require.Equal(t, 29, int(currHeight))
	for i := 15; i < 34; i++ {
		_, ok := tc.txHashes[int64(i)]
		require.True(t, ok)
	}
	require.Equal(t, 20, len(tc.txHashes))
	txs := newTestTxs(cfg, priv, currHeight+1, txHeightRange)
	tc.Add(&types.Block{Txs: txs, Height: currHeight + 1})

	require.Nil(t, tc.txHashes[15])
	require.NotNil(t, tc.txHashes[35])
}

func TestTxHashCache_Del(t *testing.T) {

	var txHeightRange int64 = 5
	tc, chain, cfg := initTxHashCache(txHeightRange)
	_, priv := util.Genaddress()

	addTestBlock(chain, cfg, priv, 0, txHeightRange, 10)
	currHeight := tc.currBlockHeight
	for i := 0; i < 5; i++ {
		tc.Del(currHeight)
		currHeight--
	}
	require.True(t, currHeight == 4)
	require.Equal(t, 9, len(tc.txHashes))
	for i := 1; i < 10; i++ {
		require.NotNil(t, tc.txHashes[int64(i)])
	}
	addTestBlock(chain, cfg, priv, currHeight+1, txHeightRange, 20)
	currHeight = tc.currBlockHeight
	for i := 0; i < 5; i++ {
		tc.Del(currHeight)
		currHeight--
	}
	require.True(t, currHeight == 19)
	require.Equal(t, 20, len(tc.txHashes))
	for i := 5; i < 24; i++ {
		require.NotNil(t, tc.txHashes[int64(i)])
	}
}

func TestTxHashCache_Contains(t *testing.T) {

	var txHeightRange int64 = 5
	tc, chain, cfg := initTxHashCache(txHeightRange)
	_, priv := util.Genaddress()

	addTestBlock(chain, cfg, priv, 0, txHeightRange, 30)
	currHeight := tc.currBlockHeight
	for i := 0; i < 5; i++ {
		tc.Del(currHeight)
		currHeight--
	}
	require.True(t, currHeight == 24)
	containsRes := []bool{true, true, true, false}
	for i := 15; i < 25; i++ {
		txs := chain.blockCache.GetBlockByHeight(int64(i)).GetBlock().GetTxs()
		require.Equal(t, 4, len(txs))
		for idx, tx := range txs {
			exist := tc.Contains(tx.Hash(), types.GetTxHeight(cfg, tx.Expire, tc.currBlockHeight))
			require.Equalf(t, containsRes[idx], exist, "height=%d, txIndex=%d", i, idx)
		}
	}

	for i := 25; i < 30; i++ {
		txs := chain.blockCache.GetBlockByHeight(int64(i)).GetBlock().GetTxs()
		for _, tx := range txs {
			exist := tc.Contains(tx.Hash(), types.GetTxHeight(cfg, tx.Expire, tc.currBlockHeight))
			require.False(t, exist)
		}
	}
}

func TestNoneCache(t *testing.T) {
	c := &noneCache{}
	c.Add(nil)
	c.Del(0)
	require.False(t, c.Contains([]byte("test"), 10))
}
