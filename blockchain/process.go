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

//ProcessBlock 处理共识模块过来的blockdetail，peer广播过来的block，以及从peer同步过来的block
// 共识模块和peer广播过来的block需要广播出去
//共识模块过来的Receipts不为空,广播和同步过来的Receipts为空
// 返回参数说明：是否主链，是否孤儿节点，具体err
func (b *BlockChain) ProcessBlock(broadcast bool, block *types.BlockDetail, pid string, addBlock bool, sequence int64) (*types.BlockDetail, bool, bool, error) {
	chainlog.Debug("ProcessBlock:Processing", "height", block.Block.Height, "blockHash", common.ToHex(block.Block.Hash(b.client.GetConfig())))

	//blockchain close 时不再处理block
	if atomic.LoadInt32(&b.isclosed) == 1 {
		return nil, false, false, types.ErrIsClosed
	}
	cfg := b.client.GetConfig()
	if block.Block.Height > 0 {
		var lastBlockHash []byte
		if addBlock {
			lastBlockHash = block.Block.GetParentHash()
		} else {
			lastBlockHash = block.Block.Hash(cfg)
		}
		if pid == "self" && !bytes.Equal(lastBlockHash, b.bestChain.Tip().hash) {
			chainlog.Error("addBlockDetail parent hash no match", "err", types.ErrBlockHashNoMatch,
				"bestHash", common.ToHex(b.bestChain.Tip().hash), "blockHash", common.ToHex(lastBlockHash),
				"addBlock", addBlock, "height", block.Block.Height)
			return nil, false, false, types.ErrBlockHashNoMatch
		}
	}
	blockHash := block.Block.Hash(cfg)

	//目前只支持删除平行链的block处理,主链不支持删除block的操作
	if !addBlock {
		if b.isParaChain {
			return b.ProcessDelParaChainBlock(broadcast, block, pid, sequence)
		}
		return nil, false, false, types.ErrNotSupport
	}
	//判断本block是否已经存在主链或者侧链中
	//如果此block已经存在，并且已经被记录执行不过，
	//将此block的源peer节点添加到故障peerlist中
	exists := b.blockExists(blockHash)
	if exists {
		is, err := b.IsErrExecBlock(block.Block.Height, blockHash)
		if is {
			b.RecordFaultPeer(pid, block.Block.Height, blockHash, err)
		}
		chainlog.Debug("ProcessBlock already have block", "blockHash", common.ToHex(blockHash))
		return nil, false, false, types.ErrBlockExist
	}

	// 判断本区块是否已经存在孤儿链中
	exists = b.orphanPool.IsKnownOrphan(blockHash)
	if exists {
		//本区块已经存在孤儿链中，但是自己的父区块也已经存在主链中
		//此时可能是上次加载父区块的过程中，刚好子区块过来导致子区块被存入到孤儿链中了没有及时处理
		//本次需要删除孤儿连中本区块的信息，尝试将此区块添加到主链上
		if b.blockExists(block.Block.GetParentHash()) {
			b.orphanPool.RemoveOrphanBlockByHash(blockHash)
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
		prevHashExists = b.blockExists(prevHash)
	}
	if !prevHashExists {
		chainlog.Debug("ProcessBlock:AddOrphanBlock", "height", block.Block.GetHeight(), "blockHash", common.ToHex(blockHash), "prevHash", common.ToHex(prevHash))
		b.orphanPool.AddOrphanBlock(broadcast, block.Block, pid, sequence)
		return nil, false, true, nil
	}

	// 基本检测通过之后尝试添加block到主链上
	return b.maybeAddBestChain(broadcast, block, pid, sequence)
}

//基本检测通过之后尝试将此block添加到主链上
func (b *BlockChain) maybeAddBestChain(broadcast bool, block *types.BlockDetail, pid string, sequence int64) (*types.BlockDetail, bool, bool, error) {
	b.chainLock.Lock()
	defer b.chainLock.Unlock()

	blockHash := block.Block.Hash(b.client.GetConfig())
	exists := b.blockExists(blockHash)
	if exists {
		return nil, false, false, types.ErrBlockExist
	}
	chainlog.Debug("maybeAddBestChain", "height", block.Block.GetHeight(), "blockHash", common.ToHex(blockHash))
	blockdetail, isMainChain, err := b.maybeAcceptBlock(broadcast, block, pid, sequence)

	if err != nil {
		return nil, false, false, err
	}
	// 尝试处理blockHash对应的孤儿子节点
	err = b.orphanPool.ProcessOrphans(blockHash, b)
	if err != nil {
		return nil, false, false, err
	}
	return blockdetail, isMainChain, false, nil
}

//检查block是否已经存在index或者数据库中
func (b *BlockChain) blockExists(hash []byte) bool {
	// Check block index first (could be main chain or side chain blocks).
	if b.index.HaveBlock(hash) {
		return true
	}

	// 检测数据库中是否存在，通过hash获取blockheader，不存在就返回false。
	blockheader, err := b.blockStore.GetBlockHeaderByHash(hash)
	if blockheader == nil || err != nil {
		return false
	}
	//block存在数据库中时，需要确认是否在主链上。不在主链上返回false
	height, err := b.blockStore.GetHeightByBlockHash(hash)
	if err != nil {
		return false
	}
	return height != -1
}

// 尝试接受此block
func (b *BlockChain) maybeAcceptBlock(broadcast bool, block *types.BlockDetail, pid string, sequence int64) (*types.BlockDetail, bool, error) {
	// 首先判断本block的Parent block是否存在index中
	prevHash := block.Block.GetParentHash()
	prevNode := b.index.LookupNode(prevHash)
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
	if atomic.LoadInt32(&b.isbatchsync) == 0 {
		sync = false
	}

	err := b.blockStore.dbMaybeStoreBlock(block, sync)
	if err != nil {
		if err == types.ErrDataBaseDamage {
			chainlog.Error("dbMaybeStoreBlock newbatch.Write", "err", err)
			go util.ReportErrEventToFront(chainlog, b.client, "blockchain", "wallet", types.ErrDataBaseDamage)
		}
		return nil, false, err
	}
	// 创建一个node并添加到内存中index
	cfg := b.client.GetConfig()
	newNode := newBlockNode(cfg, broadcast, block.Block, pid, sequence)
	if prevNode != nil {
		newNode.parent = prevNode
	}
	b.index.AddNode(newNode)

	// 将本block添加到主链中
	var isMainChain bool
	block, isMainChain, err = b.connectBestChain(newNode, block)
	if err != nil {
		return nil, false, err
	}

	return block, isMainChain, nil
}

//将block添加到主链中
func (b *BlockChain) connectBestChain(node *blockNode, block *types.BlockDetail) (*types.BlockDetail, bool, error) {

	enBestBlockCmp := b.client.GetConfig().GetModuleConfig().Consensus.EnableBestBlockCmp
	parentHash := block.Block.GetParentHash()
	tip := b.bestChain.Tip()
	cfg := b.client.GetConfig()

	// 将此block添加到主链中,tip节点刚好是插入block的父节点.
	if bytes.Equal(parentHash, tip.hash) {
		var err error
		block, err = b.connectBlock(node, block)
		if err != nil {
			return nil, false, err
		}
		return block, true, nil
	}
	chainlog.Debug("connectBestChain", "parentHash", common.ToHex(parentHash), "bestChain.Tip().hash", common.ToHex(tip.hash))

	// 获取tip节点的block总难度tipid
	tiptd, Err := b.blockStore.GetTdByBlockHash(tip.hash)
	if tiptd == nil || Err != nil {
		chainlog.Error("connectBestChain tiptd is not exits!", "height", tip.height, "b.bestChain.Tip().hash", common.ToHex(tip.hash))
		return nil, false, Err
	}
	parenttd, Err := b.blockStore.GetTdByBlockHash(parentHash)
	if parenttd == nil || Err != nil {
		chainlog.Error("connectBestChain parenttd is not exits!", "height", block.Block.Height, "parentHash", common.ToHex(parentHash), "block.Block.hash", common.ToHex(block.Block.Hash(cfg)))
		return nil, false, types.ErrParentTdNoExist
	}
	blocktd := new(big.Int).Add(node.Difficulty, parenttd)

	chainlog.Debug("connectBestChain tip:", "hash", common.ToHex(tip.hash), "height", tip.height, "TD", difficulty.BigToCompact(tiptd))
	chainlog.Debug("connectBestChain node:", "hash", common.ToHex(node.hash), "height", node.height, "TD", difficulty.BigToCompact(blocktd))

	//优先选择总难度系数大的区块
	//总难度系数，区块高度，出块时间以及父区块一致并开启最优区块比较功能时，通过共识模块来确定最优区块
	iSideChain := blocktd.Cmp(tiptd) <= 0
	if enBestBlockCmp && blocktd.Cmp(tiptd) == 0 && node.height == tip.height && util.CmpBestBlock(b.client, block.Block, tip.hash) {
		iSideChain = false
	}
	if iSideChain {
		fork := b.bestChain.FindFork(node)
		if fork != nil && bytes.Equal(parentHash, fork.hash) {
			chainlog.Info("connectBestChain FORK:", "Block hash", common.ToHex(node.hash), "fork.height", fork.height, "fork.hash", common.ToHex(fork.hash))
		} else {
			chainlog.Info("connectBestChain extends a side chain:", "Block hash", common.ToHex(node.hash), "fork.height", fork.height, "fork.hash", common.ToHex(fork.hash))
		}
		return nil, false, nil
	}

	//print
	chainlog.Debug("connectBestChain tip", "height", tip.height, "hash", common.ToHex(tip.hash))
	chainlog.Debug("connectBestChain node", "height", node.height, "hash", common.ToHex(node.hash), "parentHash", common.ToHex(parentHash))
	chainlog.Debug("connectBestChain block", "height", block.Block.Height, "hash", common.ToHex(block.Block.Hash(cfg)))

	// 获取需要重组的block node
	detachNodes, attachNodes := b.getReorganizeNodes(node)

	// Reorganize the chain.
	err := b.reorganizeChain(detachNodes, attachNodes)
	if err != nil {
		return nil, false, err
	}

	return nil, true, nil
}

//将本block信息存储到数据库中，并更新bestchain的tip节点
func (b *BlockChain) connectBlock(node *blockNode, blockdetail *types.BlockDetail) (*types.BlockDetail, error) {
	//blockchain close 时不再处理block
	if atomic.LoadInt32(&b.isclosed) == 1 {
		return nil, types.ErrIsClosed
	}

	// Make sure it's extending the end of the best chain.
	parentHash := blockdetail.Block.GetParentHash()
	if !bytes.Equal(parentHash, b.bestChain.Tip().hash) {
		chainlog.Error("connectBlock hash err", "height", blockdetail.Block.Height, "Tip.height", b.bestChain.Tip().height)
		return nil, types.ErrBlockHashNoMatch
	}

	sync := true
	if atomic.LoadInt32(&b.isbatchsync) == 0 {
		sync = false
	}

	var err error
	var lastSequence int64

	block := blockdetail.Block
	prevStateHash := b.bestChain.Tip().statehash
	errReturn := (node.pid != "self")
	blockdetail, _, err = execBlock(b.client, prevStateHash, block, errReturn, sync)
	if err != nil {
		//记录执行出错的block信息,需要过滤掉一些特殊的错误，不计入故障中，尝试再次执行
		if IsRecordFaultErr(err) {
			b.RecordFaultPeer(node.pid, block.Height, node.hash, err)
		} else if node.pid == "self" {
			// 本节点产生的block由于api或者queue导致执行失败需要删除block在index中的记录，
			// 返回错误信息给共识模块，由共识模块尝试再次发起block的执行
			// 同步或者广播过来的情况会再下了一个区块过来后重新触发此block的执行
			chainlog.Debug("connectBlock DelNode!", "height", block.Height, "node.hash", common.ToHex(node.hash), "err", err)
			b.index.DelNode(node.hash)
		}
		chainlog.Error("connectBlock ExecBlock is err!", "height", block.Height, "err", err)
		return nil, err
	}
	cfg := b.client.GetConfig()
	//要更新node的信息
	if node.pid == "self" {
		prevhash := node.hash
		node.statehash = blockdetail.Block.GetStateHash()
		node.hash = blockdetail.Block.Hash(cfg)
		b.index.UpdateNode(prevhash, node)
	}

	beg := types.Now()
	// 写入磁盘 批量将block信息写入磁盘
	newbatch := b.blockStore.NewBatch(sync)

	//保存tx信息到db中
	err = b.blockStore.AddTxs(newbatch, blockdetail)
	if err != nil {
		chainlog.Error("connectBlock indexTxs:", "height", block.Height, "err", err)
		return nil, err
	}

	//保存block信息到db中
	lastSequence, err = b.blockStore.SaveBlock(newbatch, blockdetail, node.sequence)
	if err != nil {
		chainlog.Error("connectBlock SaveBlock:", "height", block.Height, "err", err)
		return nil, err
	}
	//cache new add block
	b.cache.cacheBlock(blockdetail)

	//保存block的总难度到db中
	difficulty := difficulty.CalcWork(block.Difficulty)
	var blocktd *big.Int
	if block.Height == 0 {
		blocktd = difficulty
	} else {
		parenttd, err := b.blockStore.GetTdByBlockHash(parentHash)
		if err != nil {
			chainlog.Error("connectBlock GetTdByBlockHash", "height", block.Height, "parentHash", common.ToHex(parentHash))
			return nil, err
		}
		blocktd = new(big.Int).Add(difficulty, parenttd)
	}

	err = b.blockStore.SaveTdByBlockHash(newbatch, blockdetail.Block.Hash(cfg), blocktd)
	if err != nil {
		chainlog.Error("connectBlock SaveTdByBlockHash:", "height", block.Height, "err", err)
		return nil, err
	}
	err = newbatch.Write()
	if err != nil {
		chainlog.Error("connectBlock newbatch.Write", "err", err)
		panic(err)
	}
	chainlog.Debug("connectBlock write db", "height", block.Height, "batchsync", sync, "cost", types.Since(beg), "hash", common.ToHex(blockdetail.Block.Hash(cfg)))

	// 更新最新的高度和header
	b.blockStore.UpdateHeight2(blockdetail.GetBlock().GetHeight())
	b.blockStore.UpdateLastBlock2(blockdetail.Block)

	// 更新 best chain的tip节点
	b.bestChain.SetTip(node)

	b.query.updateStateHash(blockdetail.GetBlock().GetStateHash())

	err = b.SendAddBlockEvent(blockdetail)
	if err != nil {
		chainlog.Debug("connectBlock SendAddBlockEvent", "err", err)
	}
	// 通知此block已经处理完，主要处理孤儿节点时需要设置
	b.syncTask.Done(blockdetail.Block.GetHeight())

	//广播此block到全网络
	if node.broadcast {
		if blockdetail.Block.BlockTime-types.Now().Unix() > FutureBlockDelayTime {
			//将此block添加到futureblocks中延时广播
			b.futureBlocks.Add(string(blockdetail.Block.Hash(cfg)), blockdetail)
			chainlog.Debug("connectBlock futureBlocks.Add", "height", block.Height, "hash", common.ToHex(blockdetail.Block.Hash(cfg)), "blocktime", blockdetail.Block.BlockTime, "curtime", types.Now().Unix())
		} else {
			b.SendBlockBroadcast(blockdetail)
		}
	}

	// 使能分片存储情况下进行分片处理
	if b.cfg.EnableShard {
		isNeed, chunkInfo := b.IsNeedChunk(block.Height)
		if isNeed {
			b.ShardChunkHandle(chunkInfo, node.pid == "self")
		}
	}
	//目前非平行链并开启isRecordBlockSequence功能
	if b.isRecordBlockSequence {
		b.pushseq.UpdateSeq(lastSequence)
	}
	return blockdetail, nil
}

//从主链中删除blocks
func (b *BlockChain) disconnectBlock(node *blockNode, blockdetail *types.BlockDetail, sequence int64) error {
	var lastSequence int64
	// 只能从 best chain tip节点开始删除
	if !bytes.Equal(node.hash, b.bestChain.Tip().hash) {
		chainlog.Error("disconnectBlock:", "height", blockdetail.Block.Height, "node.hash", common.ToHex(node.hash), "bestChain.top.hash", common.ToHex(b.bestChain.Tip().hash))
		return types.ErrBlockHashNoMatch
	}

	//批量删除block的信息从磁盘中
	newbatch := b.blockStore.NewBatch(true)

	//从db中删除tx相关的信息
	err := b.blockStore.DelTxs(newbatch, blockdetail)
	if err != nil {
		chainlog.Error("disconnectBlock DelTxs:", "height", blockdetail.Block.Height, "err", err)
		return err
	}

	//从db中删除block相关的信息
	lastSequence, err = b.blockStore.DelBlock(newbatch, blockdetail, sequence)
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
	b.blockStore.UpdateHeight()
	b.blockStore.UpdateLastBlock(blockdetail.Block.ParentHash)

	// 删除主链的tip节点，将其父节点升级成tip节点
	b.bestChain.DelTip(node)

	//通知共识，mempool和钱包删除block
	err = b.SendDelBlockEvent(blockdetail)
	if err != nil {
		chainlog.Error("disconnectBlock SendDelBlockEvent", "err", err)
	}
	b.query.updateStateHash(node.parent.statehash)

	//确定node的父节点升级成tip节点
	newtipnode := b.bestChain.Tip()

	//删除缓存中的block信息
	b.cache.delBlockFromCache(blockdetail.Block.Height)

	if newtipnode != node.parent {
		chainlog.Error("disconnectBlock newtipnode err:", "newtipnode.height", newtipnode.height, "node.parent.height", node.parent.height)
	}
	if !bytes.Equal(blockdetail.Block.GetParentHash(), b.bestChain.Tip().hash) {
		chainlog.Error("disconnectBlock", "newtipnode.height", newtipnode.height, "node.parent.height", node.parent.height)
		chainlog.Error("disconnectBlock", "newtipnode.hash", common.ToHex(newtipnode.hash), "delblock.parent.hash", common.ToHex(blockdetail.Block.GetParentHash()))
	}

	chainlog.Debug("disconnectBlock success", "newtipnode.height", newtipnode.height, "node.parent.height", node.parent.height)
	chainlog.Debug("disconnectBlock success", "newtipnode.hash", common.ToHex(newtipnode.hash), "delblock.parent.hash", common.ToHex(blockdetail.Block.GetParentHash()))

	//目前非平行链并开启isRecordBlockSequence功能
	if b.isRecordBlockSequence {
		b.pushseq.UpdateSeq(lastSequence)
	}
	return nil
}

//获取重组blockchain需要删除和添加节点
func (b *BlockChain) getReorganizeNodes(node *blockNode) (*list.List, *list.List) {
	attachNodes := list.New()
	detachNodes := list.New()

	// 查找到分叉的节点，并将分叉之后的block从index链push到attachNodes中
	forkNode := b.bestChain.FindFork(node)
	for n := node; n != nil && n != forkNode; n = n.parent {
		attachNodes.PushFront(n)
	}

	// 查找到分叉的节点，并将分叉之后的block从bestchain链push到attachNodes中
	for n := b.bestChain.Tip(); n != nil && n != forkNode; n = n.parent {
		detachNodes.PushBack(n)
	}

	return detachNodes, attachNodes
}

//LoadBlockByHash 根据hash值从缓存中查询区块
func (b *BlockChain) LoadBlockByHash(hash []byte) (block *types.BlockDetail, err error) {
	block = b.cache.GetCacheBlock(hash)
	if block == nil {
		block, err = b.blockStore.LoadBlockByHash(hash)
	}
	return block, err
}

//重组blockchain
func (b *BlockChain) reorganizeChain(detachNodes, attachNodes *list.List) error {
	detachBlocks := make([]*types.BlockDetail, 0, detachNodes.Len())
	attachBlocks := make([]*types.BlockDetail, 0, attachNodes.Len())

	//通过node中的blockhash获取block信息从db中
	cfg := b.client.GetConfig()
	for e := detachNodes.Front(); e != nil; e = e.Next() {
		n := e.Value.(*blockNode)
		block, err := b.LoadBlockByHash(n.hash)

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
		block, err := b.LoadBlockByHash(n.hash)

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
		err := b.disconnectBlock(n, block, n.sequence)
		if err != nil {
			return err
		}
	}

	// Connect the new best chain blocks.
	for i, e := 0, attachNodes.Front(); e != nil; i, e = i+1, e.Next() {
		n := e.Value.(*blockNode)
		block := attachBlocks[i]

		// Update the database and chain state.
		_, err := b.connectBlock(n, block)
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

//ProcessDelParaChainBlock 只能从 best chain tip节点开始删除，目前只提供给平行链使用
func (b *BlockChain) ProcessDelParaChainBlock(broadcast bool, blockdetail *types.BlockDetail, pid string, sequence int64) (*types.BlockDetail, bool, bool, error) {

	//获取当前的tip节点
	tipnode := b.bestChain.Tip()
	blockHash := blockdetail.Block.Hash(b.client.GetConfig())

	if !bytes.Equal(blockHash, b.bestChain.Tip().hash) {
		chainlog.Error("ProcessDelParaChainBlock:", "delblockheight", blockdetail.Block.Height, "delblockHash", common.ToHex(blockHash), "bestChain.top.hash", common.ToHex(b.bestChain.Tip().hash))
		return nil, false, false, types.ErrBlockHashNoMatch
	}
	err := b.disconnectBlock(tipnode, blockdetail, sequence)
	if err != nil {
		return nil, false, false, err
	}
	//平行链回滚可能出现 向同一高度写哈希相同的区块，
	// 主链中对应的节点信息已经在disconnectBlock处理函数中删除了
	// 这里还需要删除index链中对应的节点信息
	b.index.DelNode(blockHash)

	return nil, true, false, nil
}

// IsRecordFaultErr 检测此错误是否要记录到故障错误中
func IsRecordFaultErr(err error) bool {
	return err != types.ErrFutureBlock && !api.IsGrpcError(err) && !api.IsQueueError(err)
}
