// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/types"
)

//GetBlockSequences 通过记录的block序列号获取blockd序列存储的信息
func (chain *BlockChain) GetBlockSequences(requestblock *types.ReqBlocks) (*types.BlockSequences, error) {
	blockLastSeq, err := chain.blockStore.LoadBlockLastSequence()
	if err != nil {
		chainlog.Debug("GetBlockSequences LoadBlockLastSequence", "blockLastSeq", blockLastSeq, "err", err)
	}
	if requestblock.Start > blockLastSeq {
		chainlog.Error("GetBlockSequences StartSeq err", "startSeq", requestblock.Start, "lastSeq", blockLastSeq)
		return nil, types.ErrStartHeight
	}
	if requestblock.GetStart() > requestblock.GetEnd() {
		chainlog.Error("GetBlockSequences input must Start <= End:", "startSeq", requestblock.Start, "endSeq", requestblock.End)
		return nil, types.ErrEndLessThanStartHeight
	}
	if requestblock.End-requestblock.Start >= types.MaxBlockCountPerTime {
		return nil, types.ErrMaxCountPerTime
	}
	end := requestblock.End
	if requestblock.End > blockLastSeq {
		end = blockLastSeq
	}
	start := requestblock.Start
	count := end - start + 1

	chainlog.Debug("GetBlockSequences", "Start", requestblock.Start, "End", requestblock.End, "lastSeq", blockLastSeq, "counts", count)

	var blockSequences types.BlockSequences

	for i := start; i <= end; i++ {
		blockSequence, err := chain.blockStore.GetBlockSequence(i)
		if err == nil && blockSequence != nil {
			blockSequences.Items = append(blockSequences.Items, blockSequence)
		} else {
			blockSequences.Items = append(blockSequences.Items, nil)
		}
	}
	return &blockSequences, nil
}

//ProcDelParaChainBlockMsg 处理共识过来的删除block的消息，目前只提供给平行链使用
func (chain *BlockChain) ProcDelParaChainBlockMsg(broadcast bool, ParaChainblockdetail *types.ParaChainBlockDetail, pid string) (err error) {
	cfg := chain.client.GetConfig()
	if ParaChainblockdetail == nil || ParaChainblockdetail.GetBlockdetail() == nil || ParaChainblockdetail.GetBlockdetail().GetBlock() == nil {
		chainlog.Error("ProcDelParaChainBlockMsg input block is null")
		return types.ErrInvalidParam
	}
	blockdetail := ParaChainblockdetail.GetBlockdetail()
	block := ParaChainblockdetail.GetBlockdetail().GetBlock()
	sequence := ParaChainblockdetail.GetSequence()

	_, ismain, isorphan, err := chain.ProcessBlock(broadcast, blockdetail, pid, false, sequence)
	chainlog.Debug("ProcDelParaChainBlockMsg result:", "height", block.Height, "sequence", sequence, "ismain", ismain, "isorphan", isorphan, "hash", common.ToHex(block.Hash(cfg)), "err", err)

	return err
}

//ProcAddParaChainBlockMsg 处理共识过来的add block的消息，目前只提供给平行链使用
func (chain *BlockChain) ProcAddParaChainBlockMsg(broadcast bool, ParaChainblockdetail *types.ParaChainBlockDetail, pid string) (*types.BlockDetail, error) {
	cfg := chain.client.GetConfig()
	if ParaChainblockdetail == nil || ParaChainblockdetail.GetBlockdetail() == nil || ParaChainblockdetail.GetBlockdetail().GetBlock() == nil {
		chainlog.Error("ProcAddParaChainBlockMsg input block is null")
		return nil, types.ErrInvalidParam
	}
	blockdetail := ParaChainblockdetail.GetBlockdetail()
	block := ParaChainblockdetail.GetBlockdetail().GetBlock()
	sequence := ParaChainblockdetail.GetSequence()

	fullBlockDetail, ismain, isorphan, err := chain.ProcessBlock(broadcast, blockdetail, pid, true, sequence)
	chainlog.Debug("ProcAddParaChainBlockMsg result:", "height", block.Height, "sequence", sequence, "ismain", ismain, "isorphan", isorphan, "hash", common.ToHex(block.Hash(cfg)), "err", err)

	return fullBlockDetail, err
}

//ProcGetSeqByHash 处理共识过来的通过blockhash获取seq的消息，只提供add block时的seq，用于平行链block回退
func (chain *BlockChain) ProcGetSeqByHash(hash []byte) (int64, error) {
	if len(hash) == 0 {
		chainlog.Error("ProcGetSeqByHash input hash is null")
		return -1, types.ErrInvalidParam
	}
	seq, err := chain.blockStore.GetSequenceByHash(hash)
	chainlog.Debug("ProcGetSeqByHash", "blockhash", common.ToHex(hash), "seq", seq, "err", err)

	return seq, err
}

//ProcGetMainSeqByHash 处理共识过来的通过blockhash获取seq的消息，只提供add block时的seq，用于平行链block回退
func (chain *BlockChain) ProcGetMainSeqByHash(hash []byte) (int64, error) {
	if len(hash) == 0 {
		chainlog.Error("ProcGetMainSeqByHash input hash is null")
		return -1, types.ErrInvalidParam
	}
	seq, err := chain.blockStore.GetMainSequenceByHash(hash)
	chainlog.Debug("ProcGetMainSeqByHash", "blockhash", common.ToHex(hash), "seq", seq, "err", err)

	return seq, err
}

//ProcAddBlockSeqCB 添加seq callback
func (chain *BlockChain) ProcAddBlockSeqCB(cb *types.BlockSeqCB) ([]*types.Sequence, error) {
	if cb == nil {
		chainlog.Error("ProcAddBlockSeqCB input hash is null")
		return nil, types.ErrInvalidParam
	}

	if !chain.isRecordBlockSequence {
		chainlog.Error("ProcAddBlockSeqCB not support sequence")
		return nil, types.ErrRecordBlockSequence
	}
	if chain.blockStore.seqCBNum() >= MaxSeqCB && !chain.blockStore.isSeqCBExist(cb.Name) {
		chainlog.Error("ProcAddBlockSeqCB too many seq callback")
		return nil, types.ErrTooManySeqCB
	}

	// 在不指定sequence时, 和原来行为保存一直
	if cb.LastSequence == 0 {
		err := chain.blockStore.addBlockSeqCB(cb)
		if err != nil {
			chainlog.Error("ProcAddBlockSeqCB", "addBlockSeqCB", err)
			return nil, err
		}
		chain.pushseq.addTask(cb)
		return nil, nil
	}

	// 处理带 last sequence, 推送续传的情况
	chainlog.Debug("ProcAddBlockSeqCB continue-seq-push", "name", cb.Name, "seq", cb.LastSequence,
		"hash", cb.LastBlockHash, "height", cb.LastHeight)
	// name 是否存在， 存在就继续，不需要重新注册了
	if chain.blockStore.isSeqCBExist(cb.Name) {
		chainlog.Info("ProcAddBlockSeqCB continue-seq-push", "exist", cb.Name)
		return nil, nil
	}

	lastSeq, err := chain.blockStore.LoadBlockLastSequence()
	if err != nil {
		chainlog.Error("ProcAddBlockSeqCB continue-seq-push", "load-last-seq", err)
		return nil, err
	}

	// 续传的情况下， 最好等节点同步过了原先的点， 不然同步好的删除了， 等于重新同步
	if lastSeq < cb.LastSequence {
		chainlog.Error("ProcAddBlockSeqCB continue-seq-push", "last-seq", lastSeq, "input-seq", cb.LastSequence, "err", types.ErrSequenceTooBig)
		return nil, types.ErrSequenceTooBig
	}
	// name不存在：Sequence 信息匹配，添加
	sequence, err := chain.blockStore.GetBlockSequence(cb.LastSequence)
	if err != nil {
		chainlog.Error("ProcAddBlockSeqCB continue-seq-push", "load-1", err)
		return nil, err
	}

	// 注册点，在节点上存在
	// 同一高度，不一定同一个hash，有分叉的可能；但同一个hash必定同一个高度
	reloadHash := common.ToHex(sequence.Hash)
	if cb.LastBlockHash == reloadHash {
		// 先填入last seq， 而不是从0开始
		chain.GetStore().setSeqCBLastNum([]byte(cb.Name), cb.LastSequence)
		chain.pushseq.addTask(cb)
		return nil, nil
	}

	// 注册点，在节点上不存在， 即分叉上
	// name不存在， 但对应的Hash/Height对不上
	return loadSequanceForAddCallback(chain, cb)
}

// add callback时， name不存在， 但对应的Hash/Height对不上
// 加载推荐的开始点
func loadSequanceForAddCallback(chain *BlockChain, cb *types.BlockSeqCB) ([]*types.Sequence, error) {
	count := int64(100)
	skip := int64(100)
	skipTimes := int64(100)
	if count+skipTimes > types.MaxBlockCountPerTime {
		count = types.MaxBlockCountPerTime / 2
		skipTimes = types.MaxBlockCountPerTime / 2
	}
	start := cb.LastSequence - count
	if start < 0 {
		start = 0
	}

	seqs := make([]*types.Sequence, 0)
	for i := cb.LastSequence; i >= start; i-- {
		seq, err := loadOneSeq(chain, i)
		if err != nil {
			continue
		}
		seqs = append(seqs, seq)
	}
	for cur := start - skip; cur > 0; cur = cur - skip {
		skipTimes--
		if skipTimes <= 0 {
			break
		}
		seq, err := loadOneSeq(chain, cur)
		if err != nil {
			continue
		}
		seqs = append(seqs, seq)
	}
	return seqs, types.ErrSequenceNotMatch
}

func loadOneSeq(chain *BlockChain, cur int64) (*types.Sequence, error) {
	seq, err := chain.blockStore.GetBlockSequence(cur)
	if err != nil || seq == nil {
		chainlog.Warn("ProcAddBlockSeqCB continue-seq-push", "load-2", err, "seq", cur)
		return nil, err
	}
	header, err := chain.blockStore.GetBlockHeaderByHash(seq.Hash)
	if err != nil || header == nil {
		chainlog.Warn("ProcAddBlockSeqCB continue-seq-push", "load-2", err, "seq", cur, "hash", common.ToHex(seq.Hash))
		return nil, err
	}
	return &types.Sequence{Hash: seq.Hash, Type: seq.Type, Sequence: cur, Height: header.Height}, nil
}

//ProcListBlockSeqCB 列出所有已经设置的seq callback
func (chain *BlockChain) ProcListBlockSeqCB() (*types.BlockSeqCBs, error) {
	cbs, err := chain.blockStore.listSeqCB()
	if err != nil {
		chainlog.Error("ProcListBlockSeqCB", "err", err.Error())
		return nil, err
	}
	var listSeqCBs types.BlockSeqCBs

	listSeqCBs.Items = append(listSeqCBs.Items, cbs...)

	return &listSeqCBs, nil
}

//ProcGetSeqCBLastNum 获取指定name的callback已经push的最新seq num
func (chain *BlockChain) ProcGetSeqCBLastNum(name string) int64 {
	num := chain.blockStore.getSeqCBLastNum([]byte(name))
	return num
}
