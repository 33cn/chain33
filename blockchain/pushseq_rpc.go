// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/types"
)

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
		err = chain.GetStore().setSeqCBLastNum([]byte(cb.Name), cb.LastSequence)
		if err != nil {
			chainlog.Error("ProcAddBlockSeqCB", "setSeqCBLastNum", err)
			return nil, err
		}
		err = chain.blockStore.addBlockSeqCB(cb)
		if err != nil {
			chainlog.Error("ProcAddBlockSeqCB", "addBlockSeqCB", err)
			return nil, err
		}
		chain.pushseq.addTask(cb)
		return nil, nil
	}

	// 注册点，在节点上不存在， 即分叉上
	// name不存在， 但对应的Hash/Height对不上
	return loadSequanceForAddCallback(chain.blockStore, cb)
}

// add callback时， name不存在， 但对应的Hash/Height对不上, 加载推荐的开始点
// 1. 在接近的sequence推荐，解决分叉问题
// 2. 跳跃的sequence推荐，解决在极端情况下， 有比较深的分叉， 减少交互的次数
func loadSequanceForAddCallback(store *BlockStore, cb *types.BlockSeqCB) ([]*types.Sequence, error) {
	seqsNumber := recommendSeqs(cb.LastSequence, types.MaxBlockCountPerTime)

	seqs := make([]*types.Sequence, 0)
	for _, i := range seqsNumber {
		seq, err := loadOneSeq(store, i)
		if err != nil {
			continue
		}
		seqs = append(seqs, seq)
	}
	return seqs, types.ErrSequenceNotMatch
}

func recommendSeqs(lastSequence, max int64) []int64 {
	count := int64(100)
	skip := int64(100)
	skipTimes := int64(100)
	if count+skipTimes > max {
		count = max / 2
		skipTimes = max / 2
	}

	seqs := make([]int64, 0)

	start := lastSequence - count
	if start < 0 {
		start = 0
	}
	cur := lastSequence
	for ; cur > start; cur-- {
		seqs = append(seqs, cur)
	}

	cur = start + 1 - skip
	for ; cur > 0; cur = cur - skip {
		skipTimes--
		if skipTimes < 0 {
			break
		}
		seqs = append(seqs, cur)
	}
	if cur <= 0 {
		seqs = append(seqs, 0)
	}

	return seqs
}

func loadOneSeq(store *BlockStore, cur int64) (*types.Sequence, error) {
	seq, err := store.GetBlockSequence(cur)
	if err != nil || seq == nil {
		chainlog.Warn("ProcAddBlockSeqCB continue-seq-push", "load-2", err, "seq", cur)
		return nil, err
	}
	header, err := store.GetBlockHeaderByHash(seq.Hash)
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
