// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"bytes"

	"github.com/33cn/chain33/common/merkle"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
)

//执行区块将变成一个私有的函数
func execBlock(client queue.Client, prevStateRoot []byte, block *types.Block, errReturn bool, sync bool) (*types.BlockDetail, []*types.Transaction, error) {
	//发送执行交易给execs模块
	//通过consensus module 再次检查
	chainlog.Debug("ExecBlock", "height------->", block.Height, "ntx", len(block.Txs))
	beg := types.Now()
	defer func() {
		chainlog.Info("ExecBlock", "height", block.Height, "ntx", len(block.Txs), "writebatchsync", sync, "cost", types.Since(beg))
	}()

	if errReturn && block.Height > 0 && !block.CheckSign() {
		//block的来源不是自己的mempool，而是别人的区块
		return nil, nil, types.ErrSign
	}
	//tx交易去重处理, 这个地方要查询数据库，需要一个更快的办法
	cacheTxs := types.TxsToCache(block.Txs)
	oldtxscount := len(cacheTxs)
	var err error
	cacheTxs, err = util.CheckTxDup(client, cacheTxs, block.Height)
	if err != nil {
		return nil, nil, err
	}
	newtxscount := len(cacheTxs)
	if oldtxscount != newtxscount && errReturn {
		return nil, nil, types.ErrTxDup
	}
	chainlog.Debug("ExecBlock", "prevtx", oldtxscount, "newtx", newtxscount)
	block.TxHash = merkle.CalcMerkleRootCache(cacheTxs)
	block.Txs = types.CacheToTxs(cacheTxs)
	//println("1")
	receipts := util.ExecTx(client, prevStateRoot, block)
	var maplist = make(map[string]*types.KeyValue)
	var kvset []*types.KeyValue
	var deltxlist = make(map[int]bool)
	var rdata []*types.ReceiptData //save to db receipt log
	for i := 0; i < len(receipts.Receipts); i++ {
		receipt := receipts.Receipts[i]
		if receipt.Ty == types.ExecErr {
			chainlog.Error("exec tx err", "err", receipt)
			if errReturn { //认为这个是一个错误的区块
				return nil, nil, types.ErrBlockExec
			}
			deltxlist[i] = true
			continue
		}

		rdata = append(rdata, &types.ReceiptData{Ty: receipt.Ty, Logs: receipt.Logs})
		//处理KV
		kvs := receipt.KV
		for _, kv := range kvs {
			if item, ok := maplist[string(kv.Key)]; ok {
				item.Value = kv.Value //更新item 的value
			} else {
				maplist[string(kv.Key)] = kv
				kvset = append(kvset, kv)
			}
		}
	}
	//check TxHash
	calcHash := merkle.CalcMerkleRoot(block.Txs)
	if errReturn && !bytes.Equal(calcHash, block.TxHash) {
		return nil, nil, types.ErrCheckTxHash
	}
	block.TxHash = calcHash
	//删除无效的交易
	var deltx []*types.Transaction
	if len(deltxlist) > 0 {
		var newtx []*types.Transaction
		for i := 0; i < len(block.Txs); i++ {
			if deltxlist[i] {
				deltx = append(deltx, block.Txs[i])
			} else {
				newtx = append(newtx, block.Txs[i])
			}
		}
		block.Txs = newtx
		block.TxHash = merkle.CalcMerkleRoot(block.Txs)
	}

	var detail types.BlockDetail

	calcHash = util.ExecKVMemSet(client, prevStateRoot, block.Height, kvset, sync)
	//println("2")
	if errReturn && !bytes.Equal(block.StateHash, calcHash) {
		util.ExecKVSetRollback(client, calcHash)
		if len(rdata) > 0 {
			for i, rd := range rdata {
				rd.OutputReceiptDetails(block.Txs[i].Execer, chainlog)
			}
		}
		return nil, nil, types.ErrCheckStateHash
	}
	block.StateHash = calcHash
	detail.Block = block
	detail.Receipts = rdata
	if detail.Block.Height > 0 {
		err := util.CheckBlock(client, &detail)
		if err != nil {
			chainlog.Debug("CheckBlock-->", "err=", err)
			return nil, deltx, err
		}
	}
	//println("3")
	//save to db
	// 写数据库失败时需要及时返回错误，防止错误数据被写入localdb中CHAIN33-567
	err = util.ExecKVSetCommit(client, block.StateHash)
	if err != nil {
		return nil, nil, err
	}
	detail.KV = kvset
	detail.PrevStatusHash = prevStateRoot
	//get receipts
	//save kvset and get state hash
	//ulog.Debug("blockdetail-->", "detail=", detail)
	//println("4")
	return &detail, deltx, nil
}
