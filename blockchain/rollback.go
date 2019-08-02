// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"fmt"
	"syscall"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
)

// Rollbackblock chain Rollbackblock
func (chain *BlockChain) Rollbackblock() {
	tipnode := chain.bestChain.Tip()
	if chain.cfg.RollbackBlock > 0 {
		if chain.NeedRollback(tipnode.height, chain.cfg.RollbackBlock) {
			chainlog.Info("chain rollback start")
			chain.Rollback()
			chainlog.Info("chain rollback end")
		}
		syscall.Exit(0)
	}
}

// NeedRollback need Rollback
func (chain *BlockChain) NeedRollback(curHeight, rollHeight int64) bool {
	if curHeight <= rollHeight {
		chainlog.Info("curHeight is small than rollback height, no need rollback")
		return false
	}
	kvmvccMavlFork := types.GetDappFork("store-kvmvccmavl", "ForkKvmvccmavl")
	if curHeight >= kvmvccMavlFork+10000 && rollHeight <= kvmvccMavlFork {
		chainlog.Info("because ForkKvmvccmavl", "current height", curHeight, "not support rollback to", rollHeight)
		return false
	}
	return true
}

// Rollback chain Rollback
func (chain *BlockChain) Rollback() {
	//获取当前的tip节点
	tipnode := chain.bestChain.Tip()
	startHeight := tipnode.height
	for i := startHeight; i > chain.cfg.RollbackBlock; i-- {
		blockdetail, err := chain.blockStore.LoadBlockByHeight(i)
		if err != nil {
			panic(fmt.Sprintln("rollback LoadBlockByHeight err :", err))
		}
		if chain.cfg.RollbackSave { //本地保存临时区块
			lastHeightSave := false
			if i == startHeight {
				lastHeightSave = true
			}
			err = chain.WriteBlockToDbTemp(blockdetail.Block, lastHeightSave)
			if err != nil {
				panic(fmt.Sprintln("rollback WriteBlockToDbTemp fail", "height", blockdetail.Block.Height, "error ", err))
			}
		}
		sequence := int64(-1)
		if chain.isParaChain {
			// 获取平行链的seq
			sequence, err = chain.ProcGetMainSeqByHash(blockdetail.Block.Hash())
			if err != nil {
				chainlog.Error("chain rollback get main seq fail", "height: ", i, "err", err, "hash", common.ToHex(blockdetail.Block.Hash()))
			}
		}
		err = chain.disBlock(blockdetail, sequence)
		if err != nil {
			panic(fmt.Sprintln("rollback block fail ", "height", blockdetail.Block.Height, "blockHash:", common.ToHex(blockdetail.Block.Hash())))
		}
		// 删除storedb中的状态高度
		chain.sendDelStore(blockdetail.Block.StateHash, blockdetail.Block.Height)
		chainlog.Info("chain rollback ", "height: ", i, "blockheight", blockdetail.Block.Height, "hash", common.ToHex(blockdetail.Block.Hash()), "state hash", common.ToHex(blockdetail.Block.StateHash))
	}
}

// 删除blocks
func (chain *BlockChain) disBlock(blockdetail *types.BlockDetail, sequence int64) error {
	var lastSequence int64

	//批量删除block的信息从磁盘中
	newbatch := chain.blockStore.NewBatch(true)

	//从db中删除tx相关的信息
	err := chain.blockStore.DelTxs(newbatch, blockdetail)
	if err != nil {
		chainlog.Error("disBlock DelTxs:", "height", blockdetail.Block.Height, "err", err)
		return err
	}

	//从db中删除block相关的信息
	lastSequence, err = chain.blockStore.DelBlock(newbatch, blockdetail, sequence)
	if err != nil {
		chainlog.Error("disBlock DelBlock:", "height", blockdetail.Block.Height, "err", err)
		return err
	}
	db.MustWrite(newbatch)

	//更新最新的高度和header为上一个块
	chain.blockStore.UpdateHeight()
	chain.blockStore.UpdateLastBlock(blockdetail.Block.ParentHash)

	//通知共识，mempool和钱包删除block
	err = chain.SendDelBlockEvent(blockdetail)
	if err != nil {
		chainlog.Error("disBlock SendDelBlockEvent", "err", err)
	}

	//删除缓存中的block信息
	chain.cache.delBlockFromCache(blockdetail.Block.Height)

	//目前非平行链并开启isRecordBlockSequence功能
	if chain.isRecordBlockSequence {
		chain.pushseq.updateSeq(lastSequence)
	}
	return nil
}

// 通知store删除区块，主要针对kvmvcc
func (chain *BlockChain) sendDelStore(hash []byte, height int64) {
	storeDel := &types.StoreDel{StateHash: hash, Height: height}
	msg := chain.client.NewMessage("store", types.EventStoreDel, storeDel)
	err := chain.client.Send(msg, true)
	if err != nil {
		chainlog.Debug("sendDelStoreEvent -->>store", "err", err)
	}
	_, err = chain.client.Wait(msg)
	if err != nil {
		panic(fmt.Sprintln("sendDelStore", err))
	}
}
