// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"bytes"
	"container/list"
	"math/big"
	"sync/atomic"

	"github.com/33cn/chain33/client/api"
	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/difficulty"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
)

// ProcessBlock 处理共识模块过来的blockdetail，peer广播过来的block，以及从peer同步过来的block
// 共识模块和peer广播过来的block需要广播出去
// 共识模块过来的Receipts不为空,广播和同步过来的Receipts为空
// 返回参数说明：是否主链，是否孤儿节点，具体err
func (chain *BlockChain) ProcessBlock(broadcast bool, block *types.BlockDetail, pid string, addBlock bool, sequence int64) (*types.BlockDetail, bool, bool, error) {
	chainlog.Debug("ProcessBlock:Processing", "height", block.Block.Height, "blockHash", common.ToHex(block.Block.Hash(chain.client.GetConfig())))

	//blockchain close 时不再处理block
	if atomic.LoadInt32(&chain.isclosed) == 1 {
		return nil, false, false, types.ErrIsClosed
	}
	cfg := chain.client.GetConfig()
	if block.Block.Height > 0 {
		var lastBlockHash []byte
		if addBlock {
			lastBlockHash = block.Block.GetParentHash()
		} else {
			lastBlockHash = block.Block.Hash(cfg)
		}
		if pid == "self" && !bytes.Equal(lastBlockHash, chain.bestChain.Tip().hash) {
			chainlog.Error("addBlockDetail parent hash no match", "err", types.ErrBlockHashNoMatch,
				"bestHash", common.ToHex(chain.bestChain.Tip().hash), "blockHash", common.ToHex(lastBlockHash),
				"addBlock", addBlock, "height", block.Block.Height)
			return nil, false, false, types.ErrBlockHashNoMatch
		}
	}
	blockHash := block.Block.Hash(cfg)

	//目前只支持删除平行链的block处理,主链不支持删除block的操作
	if !addBlock {
		if chain.isParaChain {
			return chain.ProcessDelParaChainBlock(broadcast, block, pid, sequence)
		}
		return nil, false, false, types.ErrNotSupport
	}
	//判断本block是否已经存在主链或者侧链中
	//如果此block已经存在，并且已经被记录执行不过，
	//将此block的源peer节点添加到故障peerlist中
	exists := chain.blockExists(blockHash)
	if exists {
		var err error
		node := chain.index.LookupNode(blockHash)
		if node != nil && node.errLog != nil {
			err = node.errLog
			chain.RecordFaultPeer(pid, block.Block.Height, blockHash, err)
		}
		chainlog.Error("ProcessBlock block exist", "height", block.Block.Height, "hash", common.ToHex(blockHash), "err", err)
		return nil, false, false, types.ErrBlockExist
	}

	// 判断本区块是否已经存在孤儿链中
	exists = chain.orphanPool.IsKnownOrphan(blockHash)
	if exists {
		//本区块已经存在孤儿链中，但是自己的父区块也已经存在主链中
		//此时可能是上次加载父区块的过程中，刚好子区块过来导致子区块被存入到孤儿链中了没有及时处理
		//本次需要删除孤儿连中本区块的信息，尝试将此区块添加到主链上
		if chain.blockExists(block.Block.GetParentHash()) {
			chain.orphanPool.RemoveOrphanBlockByHash(blockHash)
			chainlog.Debug("ProcessBlock:maybe Accept Orphan Block", "blockHash", common.ToHex(blockHash))
		} else {
			chainlog.Debug("ProcessBlock already have block(orphan)", "blockHash", common.ToHex(blockHash))
			return nil, false, false, types.ErrBlockExist
		}
	}

	//判断本block的父block是否存在，如果不存在就将此block添加到孤儿链中
	//创世块0需要做一些特殊的判断
	var prevHashExists bool
	prevHash := block.Block.GetParentHash()
	if 0 == block.Block.GetHeight() {
		if bytes.Equal(prevHash, make([]byte, sha256Len)) {
			prevHashExists = true
		}
	} else {
		prevHashExists = chain.blockExists(prevHash)
	}
	if !prevHashExists {
		chainlog.Debug("ProcessBlock:AddOrphanBlock", "height", block.Block.GetHeight(), "blockHash", common.ToHex(blockHash), "prevHash", common.ToHex(prevHash))
		chain.orphanPool.AddOrphanBlock(broadcast, block.Block, pid, sequence)
		return nil, false, true, nil
	}

	// 基本检测通过之后尝试添加block到主链上
	return chain.maybeAddBestChain(broadcast, block, pid, sequence)
}

// 基本检测通过之后尝试将此block添加到主链上
func (chain *BlockChain) maybeAddBestChain(broadcast bool, block *types.BlockDetail, pid string, sequence int64) (*types.BlockDetail, bool, bool, error) {
	chain.chainLock.Lock()
	defer chain.chainLock.Unlock()

	blockHash := block.Block.Hash(chain.client.GetConfig())
	exists := chain.blockExists(blockHash)
	if exists {
		return nil, false, false, types.ErrBlockExist
	}
	chainlog.Debug("maybeAddBestChain", "height", block.Block.GetHeight(), "blockHash", common.ToHex(blockHash))
	blockdetail, isMainChain, err := chain.maybeAcceptBlock(broadcast, block, pid, sequence)

	if err != nil {
		return nil, false, false, err
	}
	// 尝试处理blockHash对应的孤儿子节点
	err = chain.orphanPool.ProcessOrphans(blockHash, chain)
	if err != nil {
		return nil, false, false, err
	}
	return blockdetail, isMainChain, false, nil
}

// 检查block是否已经存在index或者数据库中
func (chain *BlockChain) blockExists(hash []byte) bool {
	// Check block index first (could be main chain or side chain blocks).
	if chain.index.HaveBlock(hash) {
		return true
	}

	// 检测数据库中是否存在，通过hash获取blockheader，不存在就返回false。
	blockheader, err := chain.blockStore.GetBlockHeaderByHash(hash)
	if blockheader == nil || err != nil {
		return false
	}
	//assert block hash(not equal for side chain)
	storeHash, err := chain.blockStore.GetBlockHashByHeight(blockheader.Height)
	return err == nil && bytes.Equal(storeHash, hash)
}

// 尝试接受此block
func (chain *BlockChain) maybeAcceptBlock(broadcast bool, block *types.BlockDetail, pid string, sequence int64) (*types.BlockDetail, bool, error) {
	// 首先判断本block的Parent block是否存在index中
	prevHash := block.Block.GetParentHash()
	prevNode := chain.index.LookupNode(prevHash)
	if prevNode == nil {
		chainlog.Debug("maybeAcceptBlock", "previous block is unknown", common.ToHex(prevHash))
		return nil, false, types.ErrParentBlockNoExist
	}

	blockHeight := block.Block.GetHeight()
	if blockHeight != prevNode.height+1 {
		chainlog.Debug("maybeAcceptBlock height err", "blockHeight", blockHeight, "prevHeight", prevNode.height)
		return nil, false, types.ErrBlockHeightNoMatch
	}

	//将此block存储到db中，方便后面blockchain重组时使用，加入到主链saveblock时通过hash重新覆盖即可
	sync := true
	if atomic.LoadInt32(&chain.isbatchsync) == 0 {
		sync = false
	}

	err := chain.blockStore.dbMaybeStoreBlock(block, sync)
	if err != nil {
		if err == types.ErrDataBaseDamage {
			chainlog.Error("dbMaybeStoreBlock newbatch.Write", "err", err)
			go util.ReportErrEventToFront(chainlog, chain.client, "blockchain", "wallet", types.ErrDataBaseDamage)
		}
		return nil, false, err
	}
	// 创建一个node并添加到内存中index
	cfg := chain.client.GetConfig()
	newNode := newBlockNode(cfg, broadcast, block.Block, pid, sequence)
	if prevNode != nil {
		newNode.parent = prevNode
	}
	chain.index.AddNode(newNode)

	// 将本block添加到主链中
	var isMainChain bool
	block, isMainChain, err = chain.connectBestChain(newNode, block)
	if err != nil {
		return nil, false, err
	}

	return block, isMainChain, nil
}

// 将block添加到主链中
func (chain *BlockChain) connectBestChain(node *blockNode, block *types.BlockDetail) (*types.BlockDetail, bool, error) {

	enBestBlockCmp := chain.client.GetConfig().GetModuleConfig().Consensus.EnableBestBlockCmp
	parentHash := block.Block.GetParentHash()
	tip := chain.bestChain.Tip()
	cfg := chain.client.GetConfig()

	// 将此block添加到主链中,tip节点刚好是插入block的父节点.
	if bytes.Equal(parentHash, tip.hash) {
		var err error
		block, err = chain.connectBlock(node, block)
		if err != nil {
			return nil, false, err
		}
		return block, true, nil
	}
	chainlog.Debug("connectBestChain", "parentHash", common.ToHex(parentHash), "bestChain.Tip().hash", common.ToHex(tip.hash))

	// 获取tip节点的block总难度tipid
	tiptd, Err := chain.blockStore.GetTdByBlockHash(tip.hash)
	if tiptd == nil || Err != nil {
		chainlog.Error("connectBestChain tiptd is not exits!", "height", tip.height, "b.bestChain.Tip().hash", common.ToHex(tip.hash))
		return nil, false, Err
	}
	parenttd, Err := chain.blockStore.GetTdByBlockHash(parentHash)
	if parenttd == nil || Err != nil {
		chainlog.Error("connectBestChain parenttd is not exits!", "height", block.Block.Height, "parentHash", common.ToHex(parentHash), "block.Block.hash", common.ToHex(block.Block.Hash(cfg)))
		return nil, false, types.ErrParentTdNoExist
	}
	blocktd := new(big.Int).Add(node.Difficulty, parenttd)

	chainlog.Debug("connectBestChain difficulty:", "tipHash", common.ToHex(tip.hash), "tipHeight", tip.height, "tipTD", difficulty.BigToCompact(tiptd),
		"nodeHash", common.ToHex(node.hash), "nodeHeight", node.height, "nodeTD", difficulty.BigToCompact(blocktd))

	//优先选择总难度系数大的区块
	//总难度系数，区块高度，出块时间以及父区块一致并开启最优区块比较功能时，通过共识模块来确定最优区块
	iSideChain := blocktd.Cmp(tiptd) <= 0
	if enBestBlockCmp && blocktd.Cmp(tiptd) == 0 && node.height == tip.height && util.CmpBestBlock(chain.client, block.Block, tip.hash) {
		iSideChain = false
	}
	fork := chain.bestChain.FindFork(node)
	finalized, hash := chain.finalizer.getLastFinalized()
	if iSideChain || node.height < finalized+12 {

		chainlog.Debug("connectBestChain side chain:", "nodeHeight", node.height, "nodeHash", common.ToHex(node.hash),
			"fork.height", fork.height, "fork.hash", common.ToHex(fork.hash),
			"finalized", finalized, "fHash", common.ToHex(hash))
		return nil, false, nil
	}

	if fork != nil && fork.height < finalized {

		chainlog.Debug("connectBestChain reset finalizer",
			"finalized", finalized, "fHash", common.ToHex(hash),
			"forkHeight", fork.height, "forkHash", common.ToHex(fork.hash),
			"tipHeight", tip.height, "tipHash", common.ToHex(tip.hash),
			"nodeHeight", node.height, "nodeHash", common.ToHex(node.hash))
		_ = chain.finalizer.reset(fork.height, fork.hash)
	}

	//print
	chainlog.Debug("connectBestChain reorganize", "tipHeight", tip.height, "tipHash", common.ToHex(tip.hash),
		"nodeHeight", node.height, "nodeHash", common.ToHex(node.hash), "parentHash", common.ToHex(parentHash))

	// 获取需要重组的block node
	detachNodes, attachNodes := chain.getReorganizeNodes(node, fork)

	// Reorganize the chain.
	err := chain.reorganizeChain(detachNodes, attachNodes)
	if err != nil {
		return nil, false, err
	}

	return nil, true, nil
}

// 将本block信息存储到数据库中，并更新bestchain的tip节点
func (chain *BlockChain) connectBlock(node *blockNode, blockdetail *types.BlockDetail) (*types.BlockDetail, error) {
	//blockchain close 时不再处理block
	if atomic.LoadInt32(&chain.isclosed) == 1 {
		return nil, types.ErrIsClosed
	}

	// Make sure it's extending the end of the best chain.
	parentHash := blockdetail.Block.GetParentHash()
	if !bytes.Equal(parentHash, chain.bestChain.Tip().hash) {
		chainlog.Error("connectBlock hash err", "height", blockdetail.Block.Height, "Tip.height", chain.bestChain.Tip().height)
		return nil, types.ErrBlockHashNoMatch
	}

	sync := true
	if atomic.LoadInt32(&chain.isbatchsync) == 0 {
		sync = false
	}

	var err error
	var lastSequence int64

	block := blockdetail.Block
	prevStateHash := chain.bestChain.Tip().statehash
	errReturn := (node.pid != "self")
	blockdetail, _, err = execBlock(chain.client, prevStateHash, block, errReturn, sync)
	handleErrBlk := func(err error) {
		//记录执行出错的block信息,需要过滤掉一些特殊的错误，不计入故障中，尝试再次执行
		//快速下载时执行失败的区块不需要记录错误信息，并删除index中此区块的信息尝试通过普通模式再次下载执行
		ok := IsRecordFaultErr(err)

		if node.pid == "download" || (!ok && node.pid == "self") {
			// 本节点产生的block由于api或者queue导致执行失败需要删除block在index中的记录，
			// 返回错误信息给共识模块，由共识模块尝试再次发起block的执行
			// 同步或者广播过来的情况会在下一个区块过来后重新触发此block的执行
			chainlog.Debug("connectBlock DelNode!", "height", block.Height, "node.hash", common.ToHex(node.hash), "err", err)
			chain.index.DelNode(node.hash)
		} else {
			chain.RecordFaultPeer(node.pid, block.Height, node.hash, err)
			node.errLog = err
		}
	}
	if err != nil {
		handleErrBlk(err)
		chainlog.Error("connectBlock ExecBlock is err!", "height", block.GetHeight(),
			"hash", common.ToHex(node.hash), "err", err)
		return nil, err
	}
	blkHash := block.Hash(chain.client.GetConfig())

	//要更新node的信息
	if node.pid == "self" {
		prevhash := node.hash
		node.statehash = blockdetail.Block.GetStateHash()
		node.hash = blkHash
		chain.index.UpdateNode(prevhash, node)
	}

	// 写入磁盘 批量将block信息写入磁盘
	newbatch := chain.blockStore.batch
	newbatch.Reset()
	newbatch.UpdateWriteSync(sync)
	//保存tx信息到db中
	beg := types.Now()
	// exec local
	err = chain.blockStore.AddTxs(newbatch, blockdetail)
	if err != nil {
		handleErrBlk(err)
		chainlog.Error("connectBlock indexTxs:", "height", block.Height, "hash", common.ToHex(node.hash), "err", err)
		return nil, err
	}
	txCost := types.Since(beg)
	beg = types.Now()
	//保存block信息到db中
	lastSequence, err = chain.blockStore.SaveBlock(newbatch, blockdetail, node.sequence)
	if err != nil {
		chainlog.Error("connectBlock SaveBlock:", "height", block.Height, "err", err)
		return nil, err
	}
	saveBlkCost := types.Since(beg)
	//hashCache new add block
	beg = types.Now()
	chain.AddCacheBlock(blockdetail)
	cacheCost := types.Since(beg)

	//保存block的总难度到db中
	difficulty := difficulty.CalcWork(block.Difficulty)
	var blocktd *big.Int
	if block.Height == 0 {
		blocktd = difficulty
	} else {
		parenttd, err := chain.blockStore.GetTdByBlockHash(parentHash)
		if err != nil {
			chainlog.Error("connectBlock GetTdByBlockHash", "height", block.Height, "parentHash", common.ToHex(parentHash))
			return nil, err
		}
		blocktd = new(big.Int).Add(difficulty, parenttd)
	}

	err = chain.blockStore.SaveTdByBlockHash(newbatch, blkHash, blocktd)
	if err != nil {
		chainlog.Error("connectBlock SaveTdByBlockHash:", "height", block.Height, "err", err)
		return nil, err
	}
	beg = types.Now()
	err = newbatch.Write()
	if err != nil {
		chainlog.Error("connectBlock newbatch.Write", "err", err)
		panic(err)
	}
	writeCost := types.Since(beg)
	chainlog.Info("ConnectBlock", "execLocal", txCost, "saveBlk", saveBlkCost, "cacheBlk", cacheCost, "writeBatch", writeCost)
	chainlog.Debug("connectBlock info", "height", block.Height, "batchsync", sync, "hash", common.ToHex(blkHash))

	// 更新最新的高度和header
	chain.blockStore.UpdateHeight2(blockdetail.GetBlock().GetHeight())
	chain.blockStore.UpdateLastBlock2(blockdetail.Block)

	// 更新 best chain的tip节点
	chain.bestChain.SetTip(node)

	chain.query.updateStateHash(blockdetail.GetBlock().GetStateHash())

	err = chain.SendAddBlockEvent(blockdetail)
	if err != nil {
		chainlog.Debug("connectBlock SendAddBlockEvent", "err", err)
	}
	// 通知此block已经处理完，主要处理孤儿节点时需要设置
	chain.syncTask.Done(blockdetail.Block.GetHeight())

	//广播此block到全网络
	if node.broadcast && !chain.cfg.DisableBlockBroadcast {
		if blockdetail.Block.BlockTime-types.Now().Unix() > FutureBlockDelayTime {
			//将此block添加到futureblocks中延时广播
			chain.futureBlocks.Add(string(blkHash), blockdetail)
			chainlog.Debug("connectBlock futureBlocks.Add", "height", block.Height, "hash", common.ToHex(blkHash), "blocktime", blockdetail.Block.BlockTime, "curtime", types.Now().Unix())
		} else {
			chain.SendBlockBroadcast(blockdetail)
		}
	}

	//目前非平行链并开启isRecordBlockSequence功能和enablePushSubscribe
	if chain.isRecordBlockSequence && chain.enablePushSubscribe {
		chain.push.UpdateSeq(lastSequence)
		chainlog.Debug("isRecordBlockSequence", "lastSequence", lastSequence, "height", block.Height)
	}
	return blockdetail, nil
}

// 从主链中删除blocks
func (chain *BlockChain) disconnectBlock(node *blockNode, blockdetail *types.BlockDetail, sequence int64) error {
	var lastSequence int64
	// 只能从 best chain tip节点开始删除
	if !bytes.Equal(node.hash, chain.bestChain.Tip().hash) {
		chainlog.Error("disconnectBlock:", "height", blockdetail.Block.Height, "node.hash", common.ToHex(node.hash), "bestChain.top.hash", common.ToHex(chain.bestChain.Tip().hash))
		return types.ErrBlockHashNoMatch
	}

	//批量删除block的信息从磁盘中
	newbatch := chain.blockStore.NewBatch(true)

	//从db中删除tx相关的信息
	err := chain.blockStore.DelTxs(newbatch, blockdetail)
	if err != nil {
		chainlog.Error("disconnectBlock DelTxs:", "height", blockdetail.Block.Height, "err", err)
		return err
	}
	//优先删除缓存中的block信息
	chain.DelCacheBlock(blockdetail.Block.Height, node.hash)

	//从db中删除block相关的信息
	lastSequence, err = chain.blockStore.DelBlock(newbatch, blockdetail, sequence)
	if err != nil {
		chainlog.Error("disconnectBlock DelBlock:", "height", blockdetail.Block.Height, "err", err)
		return err
	}
	err = newbatch.Write()
	if err != nil {
		chainlog.Error("disconnectBlock newbatch.Write", "err", err)
		panic(err)
	}
	//更新最新的高度和header为上一个块
	chain.blockStore.UpdateHeight()
	chain.blockStore.UpdateLastBlock(blockdetail.Block.ParentHash)

	// 删除主链的tip节点，将其父节点升级成tip节点
	chain.bestChain.DelTip(node)

	//通知共识，mempool和钱包删除block
	err = chain.SendDelBlockEvent(blockdetail)
	if err != nil {
		chainlog.Error("disconnectBlock SendDelBlockEvent", "err", err)
	}
	chain.query.updateStateHash(node.parent.statehash)

	//确定node的父节点升级成tip节点
	newtipnode := chain.bestChain.Tip()

	if newtipnode != node.parent {
		chainlog.Error("disconnectBlock newtipnode err:", "newtipnode.height", newtipnode.height, "node.parent.height", node.parent.height)
	}
	if !bytes.Equal(blockdetail.Block.GetParentHash(), chain.bestChain.Tip().hash) {
		chainlog.Error("disconnectBlock", "newtipnode.height", newtipnode.height, "node.parent.height", node.parent.height)
		chainlog.Error("disconnectBlock", "newtipnode.hash", common.ToHex(newtipnode.hash), "delblock.parent.hash", common.ToHex(blockdetail.Block.GetParentHash()))
	}

	chainlog.Debug("disconnectBlock success", "newtipnode.height", newtipnode.height, "node.parent.height", node.parent.height)
	chainlog.Debug("disconnectBlock success", "newtipnode.hash", common.ToHex(newtipnode.hash), "delblock.parent.hash", common.ToHex(blockdetail.Block.GetParentHash()))

	//目前非平行链并开启isRecordBlockSequence功能和enablePushSubscribe
	if chain.isRecordBlockSequence && chain.enablePushSubscribe {
		chain.push.UpdateSeq(lastSequence)
		chainlog.Debug("isRecordBlockSequence", "lastSequence", lastSequence, "height", blockdetail.Block.Height)
	}

	return nil
}

// 获取重组blockchain需要删除和添加节点
func (chain *BlockChain) getReorganizeNodes(node, forkNode *blockNode) (*list.List, *list.List) {
	attachNodes := list.New()
	detachNodes := list.New()

	// 查找到分叉的节点，并将分叉之后的block从index链push到attachNodes中
	//forkNode := chain.bestChain.FindFork(node)
	for n := node; n != nil && n != forkNode; n = n.parent {
		attachNodes.PushFront(n)
	}

	// 查找到分叉的节点，并将分叉之后的block从bestchain链push到attachNodes中
	for n := chain.bestChain.Tip(); n != nil && n != forkNode; n = n.parent {
		detachNodes.PushBack(n)
	}

	return detachNodes, attachNodes
}

// LoadBlockByHash 根据hash值从缓存中查询区块
func (chain *BlockChain) LoadBlockByHash(hash []byte) (block *types.BlockDetail, err error) {

	//从缓存的最新区块中获取
	block = chain.blockCache.GetBlockByHash(hash)
	if block != nil {
		return block, err
	}

	//从缓存的活跃区块中获取
	block, _ = chain.blockStore.GetActiveBlock(string(hash))
	if block != nil {
		return block, err
	}

	//从数据库中获取
	block, err = chain.blockStore.LoadBlockByHash(hash)

	//如果是主链区块需要添加到活跃区块的缓存中
	if block != nil {
		mainHash, _ := chain.blockStore.GetBlockHashByHeight(block.Block.GetHeight())
		if mainHash != nil && bytes.Equal(mainHash, hash) {
			chain.blockStore.AddActiveBlock(string(hash), block)
		}
	}
	return block, err
}

// 重组blockchain
func (chain *BlockChain) reorganizeChain(detachNodes, attachNodes *list.List) error {
	detachBlocks := make([]*types.BlockDetail, 0, detachNodes.Len())
	attachBlocks := make([]*types.BlockDetail, 0, attachNodes.Len())

	//通过node中的blockhash获取block信息从db中
	cfg := chain.client.GetConfig()
	for e := detachNodes.Front(); e != nil; e = e.Next() {
		n := e.Value.(*blockNode)
		block, err := chain.LoadBlockByHash(n.hash)

		// 需要删除的blocks
		if block != nil && err == nil {
			detachBlocks = append(detachBlocks, block)
			chainlog.Debug("reorganizeChain detachBlocks ", "height", block.Block.Height, "hash", common.ToHex(block.Block.Hash(cfg)))
		} else {
			chainlog.Error("reorganizeChain detachBlocks fail", "height", n.height, "hash", common.ToHex(n.hash), "err", err)
			return err
		}
	}

	for e := attachNodes.Front(); e != nil; e = e.Next() {
		n := e.Value.(*blockNode)
		block, err := chain.LoadBlockByHash(n.hash)

		// 需要加载到db的blocks
		if block != nil && err == nil {
			attachBlocks = append(attachBlocks, block)
			chainlog.Debug("reorganizeChain attachBlocks ", "height", block.Block.Height, "hash", common.ToHex(block.Block.Hash(cfg)))
		} else {
			chainlog.Error("reorganizeChain attachBlocks fail", "height", n.height, "hash", common.ToHex(n.hash), "err", err)
			return err
		}
	}

	// Disconnect blocks from the main chain.
	for i, e := 0, detachNodes.Front(); e != nil; i, e = i+1, e.Next() {
		n := e.Value.(*blockNode)
		block := detachBlocks[i]

		// Update the database and chain state.
		err := chain.disconnectBlock(n, block, n.sequence)
		if err != nil {
			return err
		}
	}

	// Connect the new best chain blocks.
	for i, e := 0, attachNodes.Front(); e != nil; i, e = i+1, e.Next() {
		n := e.Value.(*blockNode)
		block := attachBlocks[i]

		// Update the database and chain state.
		_, err := chain.connectBlock(n, block)
		if err != nil {
			return err
		}
	}

	// Log the point where the chain forked and old and new best chain
	// heads.
	if attachNodes.Front() != nil {
		firstAttachNode := attachNodes.Front().Value.(*blockNode)
		chainlog.Debug("REORGANIZE: Chain forks at hash", "hash", common.ToHex(firstAttachNode.parent.hash), "height", firstAttachNode.parent.height)

	}
	if detachNodes.Front() != nil {
		firstDetachNode := detachNodes.Front().Value.(*blockNode)
		chainlog.Debug("REORGANIZE: Old best chain head was hash", "hash", common.ToHex(firstDetachNode.hash), "height", firstDetachNode.parent.height)

	}
	if attachNodes.Back() != nil {
		lastAttachNode := attachNodes.Back().Value.(*blockNode)
		chainlog.Debug("REORGANIZE: New best chain head is hash", "hash", common.ToHex(lastAttachNode.hash), "height", lastAttachNode.parent.height)
	}
	return nil
}

// ProcessDelParaChainBlock 只能从 best chain tip节点开始删除，目前只提供给平行链使用
func (chain *BlockChain) ProcessDelParaChainBlock(broadcast bool, blockdetail *types.BlockDetail, pid string, sequence int64) (*types.BlockDetail, bool, bool, error) {

	//获取当前的tip节点
	tipnode := chain.bestChain.Tip()
	blockHash := blockdetail.Block.Hash(chain.client.GetConfig())

	if !bytes.Equal(blockHash, chain.bestChain.Tip().hash) {
		chainlog.Error("ProcessDelParaChainBlock:", "delblockheight", blockdetail.Block.Height, "delblockHash", common.ToHex(blockHash), "bestChain.top.hash", common.ToHex(chain.bestChain.Tip().hash))
		return nil, false, false, types.ErrBlockHashNoMatch
	}
	err := chain.disconnectBlock(tipnode, blockdetail, sequence)
	if err != nil {
		return nil, false, false, err
	}
	//平行链回滚可能出现 向同一高度写哈希相同的区块，
	// 主链中对应的节点信息已经在disconnectBlock处理函数中删除了
	// 这里还需要删除index链中对应的节点信息
	chain.index.DelNode(blockHash)

	return nil, true, false, nil
}

// IsRecordFaultErr 检测此错误是否要记录到故障错误中
func IsRecordFaultErr(err error) bool {
	return err != types.ErrFutureBlock && !api.IsGrpcError(err) && !api.IsQueueError(err)
}
