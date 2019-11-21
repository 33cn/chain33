// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"fmt"

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
func (chain *BlockChain) ProcAddBlockSeqCB(cb *types.BlockSeqCB) (interface{}, error) {
	if cb == nil {
		return nil, types.ErrInvalidParam
	}

	if !chain.isRecordBlockSequence {
		return nil, types.ErrRecordBlockSequence
	}
	if chain.blockStore.seqCBNum() >= MaxSeqCB && !chain.blockStore.isSeqCBExist(cb.Name) {
		return nil, types.ErrTooManySeqCB
	}

	// 在不指定sequence时, 和原来行为保存一直
	if cb.LastSequence == 0 {
		err := chain.blockStore.addBlockSeqCB(cb)
		if err != nil {
			return nil, err
		}
		chain.pushseq.addTask(cb)
		return nil, nil
	}

	// TODO
	if cb.LastSequence != 0 {
		// name 是否存在， 存在就继续，不用在重新注册了
		if chain.blockStore.isSeqCBExist(cb.Name) {
			return nil, nil
		}
		// name不存在：Sequence 信息匹配，添加
		req := &types.ReqBlocks{Start: cb.LastSequence, End: cb.LastSequence, IsDetail: false, Pid: []string{}}
		sequences, err := chain.GetBlockSequences(req)
		if err != nil {
			// TODO check not exist
			return nil, err
		}
		// 同一高度，不一定同一个hash，有分叉的可能；但同一个hash必定同一个高度
		reloadHash := common.ToHex(sequences.Items[0].Hash)
		if cb.LastBlockHash == reloadHash {
			// TODO 开始参数填入， 而不是从0开始
			chain.pushseq.addTask(cb)
			return nil, nil
		}
		// name不存在， 但对应的Hash/Height对不上
		start := cb.LastSequence - 100
		if start < 0 {
			start = 0
		}
		req2 := &types.ReqBlocks{Start: start, End: cb.LastSequence, IsDetail: false, Pid: []string{}}
		sequences, err = chain.GetBlockSequences(req2)
		if err != nil {
			// TODO check not exist
			return nil, err
		}
		LoadedBlocks := []types.Block{}
		return LoadedBlocks, fmt.Errorf("%s", "SequenceNotMatch")
	}

	return nil, nil
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
