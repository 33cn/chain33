// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"sync/atomic"
	"time"

	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
)

// ReduceChain 精简chain
func (chain *BlockChain) ReduceChain() {
	height := chain.GetBlockHeight()
	cfg := chain.client.GetConfig()
	if cfg.IsEnable("reduceLocaldb") {
		// 精简localdb
		chain.InitReduceLocalDB(height)
		chain.reducewg.Add(1)
		go chain.ReduceLocalDB()
	} else {
		flagHeight, _ := chain.blockStore.loadFlag(types.ReduceLocaldbHeight) //一旦开启reduceLocaldb，后续不能关闭
		if flagHeight != 0 {
			panic("toml config disable reduce localdb, but database enable reduce localdb")
		}
	}
}

// InitReduceLocalDB 初始化启动时候执行
func (chain *BlockChain) InitReduceLocalDB(height int64) {
	flag, err := chain.blockStore.loadFlag(types.FlagReduceLocaldb)
	if err != nil {
		panic(err)
	}
	flagHeight, err := chain.blockStore.loadFlag(types.ReduceLocaldbHeight)
	if err != nil {
		panic(err)
	}
	if flag == 0 {
		endHeight := height - SafetyReduceHeight //初始化时候执行精简高度要大于最大回滚高度
		if endHeight > flagHeight {
			chainlog.Info("start reduceLocaldb", "start height", flagHeight, "end height", endHeight)
			chain.walkOver(flagHeight, endHeight, false, chain.reduceBodyInit,
				func(batch dbm.Batch, height int64) {
					height++
					batch.Set(types.ReduceLocaldbHeight, types.Encode(&types.Int64{Data: height}))
				})
			// CompactRange执行将会阻塞仅仅做一次压缩
			chainlog.Info("reduceLocaldb start compact db")
			err = chain.blockStore.db.CompactRange(nil, nil)
			chainlog.Info("reduceLocaldb end compact db", "error", err)
		}
		chain.blockStore.saveReduceLocaldbFlag()
	}
}

// walkOver walk over
func (chain *BlockChain) walkOver(start, end int64, sync bool, fn func(batch dbm.Batch, height int64),
	fnflag func(batch dbm.Batch, height int64)) {
	// 删除
	const batchDataSize = 1024 * 1024 * 10
	newbatch := chain.blockStore.NewBatch(sync)
	for i := start; i <= end; i++ {
		fn(newbatch, i)
		if newbatch.ValueSize() > batchDataSize {
			// 记录当前高度
			fnflag(newbatch, i)
			dbm.MustWrite(newbatch)
			newbatch.Reset()
			chainlog.Info("reduceLocaldb", "height", i)
		}
	}
	if newbatch.ValueSize() > 0 {
		// 记录当前高度
		fnflag(newbatch, end)
		dbm.MustWrite(newbatch)
		newbatch.Reset()
		chainlog.Info("reduceLocaldb end", "height", end)
	}
}

// reduceBodyInit 将body中的receipt进行精简；将TxHashPerfix为key的TxResult中的receipt和tx字段进行精简
func (chain *BlockChain) reduceBodyInit(batch dbm.Batch, height int64) {
	blockDetail, err := chain.blockStore.LoadBlock(height, nil)
	if err == nil {
		cfg := chain.client.GetConfig()
		kvs, err := delBlockReceiptTable(chain.blockStore.db, height, blockDetail.Block.Hash(cfg))
		if err != nil {
			chainlog.Debug("reduceBody delBlockReceiptTable", "height", height, "error", err)
			return
		}
		for _, kv := range kvs {
			if kv.GetValue() == nil {
				batch.Delete(kv.GetKey())
			}
		}
		chain.reduceIndexTx(batch, blockDetail.GetBlock())
	}
}

// reduceIndexTx 对数据库中的 hash-TX进行精简
func (chain *BlockChain) reduceIndexTx(batch dbm.Batch, block *types.Block) {
	cfg := chain.client.GetConfig()
	Txs := block.GetTxs()
	for index, tx := range Txs {
		hash := tx.Hash()
		// 之前执行quickIndex时候未对无用hash做删除处理，占用空间，因此这里删除
		if cfg.IsEnable("quickIndex") {
			batch.Delete(hash)
		}

		// 为了提高效率组合生成txresult,不从数据库中读取已有txresult
		txresult := &types.TxResult{
			Height:     block.Height,
			Index:      int32(index),
			Blocktime:  block.BlockTime,
			ActionName: tx.ActionName(),
		}
		batch.Set(cfg.CalcTxKey(hash), cfg.CalcTxKeyValue(txresult))
	}
}

func (chain *BlockChain) deleteTx(batch dbm.Batch, block *types.Block) {
	cfg := chain.client.GetConfig()
	Txs := block.GetTxs()
	for _, tx := range Txs {
		batch.Delete(cfg.CalcTxKey(tx.Hash()))
	}
}

// reduceReceipts 精简receipts
func reduceReceipts(src *types.BlockBody) []*types.ReceiptData {
	receipts := types.CloneReceipts(src.Receipts)
	for i := 0; i < len(receipts); i++ {
		for j := 0; j < len(receipts[i].Logs); j++ {
			if receipts[i].Logs[j] != nil {
				if receipts[i].Logs[j].Ty == types.TyLogErr { // 为了匹配界面显示
					continue
				}
				receipts[i].Logs[j].Log = nil
			}
		}
	}
	return receipts
}

// ReduceLocalDB 实时精简localdb
func (chain *BlockChain) ReduceLocalDB() {
	defer chain.reducewg.Done()

	flagHeight, err := chain.blockStore.loadFlag(types.ReduceLocaldbHeight)
	if err != nil {
		panic(err)
	}
	if flagHeight < 0 {
		flagHeight = 0
	}
	// 10s检测一次是否可以进行reduce localdb
	checkTicker := time.NewTicker(10 * time.Second)
	defer checkTicker.Stop()

	for {
		select {
		case <-chain.quit:
			return
		case <-checkTicker.C:
			flagHeight = chain.TryReduceLocalDB(flagHeight, 100)
		}
	}
}

// TryReduceLocalDB TryReduce try reduce
func (chain *BlockChain) TryReduceLocalDB(flagHeight int64, rangeHeight int64) (newHeight int64) {
	if rangeHeight <= 0 {
		rangeHeight = 100
	}
	height := chain.GetBlockHeight()
	safetyHeight := height - ReduceHeight
	if safetyHeight/rangeHeight > flagHeight/rangeHeight { // 每隔rangeHeight区块进行一次精简
		sync := true
		if atomic.LoadInt32(&chain.isbatchsync) == 0 {
			sync = false
		}
		chain.walkOver(flagHeight, safetyHeight, sync, chain.reduceBody,
			func(batch dbm.Batch, height int64) {
				// 记录的时候记录下一个，中断开始执行的就是下一个
				height++
				batch.Set(types.ReduceLocaldbHeight, types.Encode(&types.Int64{Data: height}))
			})
		flagHeight = safetyHeight + 1
		chainlog.Debug("reduceLocaldb ticker", "current height", flagHeight)
		return flagHeight
	}
	return flagHeight
}

// reduceBody 将body中的receipt进行精简；
func (chain *BlockChain) reduceBody(batch dbm.Batch, height int64) {
	hash, err := chain.blockStore.GetBlockHashByHeight(height)
	if err == nil {
		kvs, err := delBlockReceiptTable(chain.blockStore.db, height, hash)
		if err != nil {
			chainlog.Debug("reduceBody delBlockReceiptTable", "height", height, "error", err)
			return
		}
		for _, kv := range kvs {
			if kv.GetValue() == nil {
				batch.Delete(kv.GetKey())
			}
		}
	}
}
