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
	if chain.cfg.RollbackBlock <= 0 {
		return
	}
	if tipnode.height <= chain.cfg.RollbackBlock {
		// Nothing to do, and the node can start from where it is: exit 0, so a service
		// manager does not report a failure for a no-op.
		chainlog.Info("curHeight is small than rollback height, no need rollback")
		syscall.Exit(0)
	}
	if !chain.NeedRollback(tipnode.height, chain.cfg.RollbackBlock) {
		// NeedRollback logged why. Exit non-zero so the caller cannot mistake a refusal for
		// a completed rollback.
		syscall.Exit(1)
	}
	chainlog.Info("chain rollback start")
	chain.Rollback()
	chainlog.Info("chain rollback end")
	syscall.Exit(0)
}

// NeedRollback need Rollback
func (chain *BlockChain) NeedRollback(curHeight, rollHeight int64) bool {
	if curHeight <= rollHeight {
		chainlog.Info("curHeight is small than rollback height, no need rollback")
		return false
	}
	cfg := chain.client.GetConfig()
	kvmvccMavlFork := cfg.GetDappFork("store-kvmvccmavl", "ForkKvmvccmavl")
	if curHeight >= kvmvccMavlFork+10000 && rollHeight <= kvmvccMavlFork {
		chainlog.Info("because ForkKvmvccmavl", "current height", curHeight, "not support rollback to", rollHeight)
		return false
	}
	// The chunk archiver deletes block *bodies*, and a rollback needs two things: every body
	// it walks over on the way down, and -- once the tip stands at rollHeight -- the bodies
	// the next start reads above that tip. A target below either one cannot be recovered
	// from, because by the time the failure shows up the tip has already moved: the deletion
	// loop panics on the first missing body, and a node that gets past it panics in
	// InitCache on the next start instead. Refuse up front.
	bodyFloor, archived, err := chain.archivedBodyFloor()
	if err != nil {
		return false
	}
	if !archived {
		// Nothing has been archived, so no body is missing whatever the target.
		return true
	}
	lowest := bodyFloor
	if cfg.IsEnable("TxHeight") {
		// InitCache reads [tip-DefCacheSize, tip] and [tip-HighAllowPackHeight-LowAllowPackHeight+1, tip]
		// and panics on the first block it cannot load, so the new tip has to clear the
		// archived range by the larger of the two windows.
		window := types.HighAllowPackHeight + types.LowAllowPackHeight - 1
		if chain.cfg.DefCacheSize > window {
			window = chain.cfg.DefCacheSize
		}
		lowest = bodyFloor + window
	}
	if rollHeight < lowest {
		chainlog.Error("rollback target is below the archived block bodies, refusing",
			"target", rollHeight, "lowest usable target", lowest,
			"lowest stored block body", bodyFloor, "txHeight", cfg.IsEnable("TxHeight"))
		return false
	}
	return true
}

// archivedBodyFloor returns the lowest height whose block body is still stored, and whether
// anything has been archived at all. The chunk records decide it, not the current
// configuration: chunkShardHandle writes each record with the ChunkblockNum in effect at the
// time, and DeleteBlockBody deletes exactly the range that record names, so a node whose
// ChunkblockNum was changed between two starts would otherwise compute a floor that is too
// low. The cursor is the last chunk the archiver *attempted*: it advances even when a chunk
// has no record or a height inside it is skipped, which can only make this floor too high,
// never too low.
func (chain *BlockChain) archivedBodyFloor() (floor int64, archived bool, err error) {
	maxDeletedChunk, err := chain.blockStore.GetMaxDeletedChunkNumWithErr()
	if err != nil {
		if err == db.ErrNotFoundInDb {
			return 0, false, nil
		}
		// A read failure must not be mistaken for "nothing was archived".
		chainlog.Error("cannot read the archived chunk cursor, refusing to roll back", "err", err)
		return 0, false, err
	}
	if chunk, err := chain.blockStore.GetChunkInfo(maxDeletedChunk); err == nil {
		return chunk.End + 1, true, nil
	}
	return (maxDeletedChunk + 1) * chain.cfg.ChunkblockNum, true, nil
}

// Rollback chain Rollback
func (chain *BlockChain) Rollback() {
	cfg := chain.client.GetConfig()
	//获取当前的tip节点
	tipnode := chain.bestChain.Tip()
	startHeight := tipnode.height
	for i := startHeight; i > chain.cfg.RollbackBlock; i-- {
		blockdetail, err := chain.blockStore.LoadBlock(i, nil)
		if err != nil {
			panic(fmt.Sprintln("rollback LoadBlock err :", err))
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
			sequence, err = chain.ProcGetMainSeqByHash(blockdetail.Block.Hash(cfg))
			if err != nil {
				chainlog.Error("chain rollback get main seq fail", "height: ", i, "err", err, "hash", common.ToHex(blockdetail.Block.Hash(cfg)))
			}
		}
		err = chain.disBlock(blockdetail, sequence)
		if err != nil {
			panic(fmt.Sprintln("rollback block fail ", "height", blockdetail.Block.Height, "blockHash:", common.ToHex(blockdetail.Block.Hash(cfg))))
		}
		// 删除storedb中的状态高度
		chain.sendDelStore(blockdetail.Block.StateHash, blockdetail.Block.Height)
		chainlog.Info("chain rollback ", "height: ", i, "blockheight", blockdetail.Block.Height, "hash", common.ToHex(blockdetail.Block.Hash(cfg)), "state hash", common.ToHex(blockdetail.Block.StateHash))
	}
}

// 删除blocks
func (chain *BlockChain) disBlock(blockdetail *types.BlockDetail, sequence int64) error {
	var lastSequence int64
	cfg := chain.client.GetConfig()

	//批量删除block的信息从磁盘中
	newbatch := chain.blockStore.NewBatch(true)

	//从db中删除tx相关的信息
	err := chain.blockStore.DelTxs(newbatch, blockdetail)
	if err != nil {
		chainlog.Error("disBlock DelTxs:", "height", blockdetail.Block.Height, "err", err)
		return err
	}

	//优先删除缓存中的block信息
	chain.DelCacheBlock(blockdetail.Block.Height, blockdetail.Block.Hash(cfg))

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

	//目前非平行链并开启isRecordBlockSequence功能和enablePushSubscribe
	if chain.isRecordBlockSequence && chain.enablePushSubscribe {
		chain.push.UpdateSeq(lastSequence)
		chainlog.Debug("isRecordBlockSequence", "lastSequence", lastSequence, "height", blockdetail.Block.Height)
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
