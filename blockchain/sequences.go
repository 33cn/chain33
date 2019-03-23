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
	if ParaChainblockdetail == nil || ParaChainblockdetail.GetBlockdetail() == nil || ParaChainblockdetail.GetBlockdetail().GetBlock() == nil {
		chainlog.Error("ProcDelParaChainBlockMsg input block is null")
		return types.ErrInvalidParam
	}
	blockdetail := ParaChainblockdetail.GetBlockdetail()
	block := ParaChainblockdetail.GetBlockdetail().GetBlock()
	sequence := ParaChainblockdetail.GetSequence()

	_, ismain, isorphan, err := chain.ProcessBlock(broadcast, blockdetail, pid, false, sequence)
	chainlog.Debug("ProcDelParaChainBlockMsg result:", "height", block.Height, "sequence", sequence, "ismain", ismain, "isorphan", isorphan, "hash", common.ToHex(block.Hash()), "err", err)

	return err
}

//ProcAddParaChainBlockMsg 处理共识过来的add block的消息，目前只提供给平行链使用
func (chain *BlockChain) ProcAddParaChainBlockMsg(broadcast bool, ParaChainblockdetail *types.ParaChainBlockDetail, pid string) (*types.BlockDetail, error) {
	if ParaChainblockdetail == nil || ParaChainblockdetail.GetBlockdetail() == nil || ParaChainblockdetail.GetBlockdetail().GetBlock() == nil {
		chainlog.Error("ProcAddParaChainBlockMsg input block is null")
		return nil, types.ErrInvalidParam
	}
	blockdetail := ParaChainblockdetail.GetBlockdetail()
	block := ParaChainblockdetail.GetBlockdetail().GetBlock()
	sequence := ParaChainblockdetail.GetSequence()

	fullBlockDetail, ismain, isorphan, err := chain.ProcessBlock(broadcast, blockdetail, pid, true, sequence)
	chainlog.Debug("ProcAddParaChainBlockMsg result:", "height", block.Height, "sequence", sequence, "ismain", ismain, "isorphan", isorphan, "hash", common.ToHex(block.Hash()), "err", err)

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

//ProcAddBlockSeqCB 添加seq callback
func (chain *BlockChain) ProcAddBlockSeqCB(cb *types.BlockSeqCB) error {
	if cb == nil {
		return types.ErrInvalidParam
	}

	if !chain.isRecordBlockSequence {
		return types.ErrRecordBlockSequence
	}
	if chain.blockStore.seqCBNum() >= MaxSeqCB && !chain.blockStore.isSeqCBExist(cb.Name) {
		return types.ErrTooManySeqCB
	}
	err := chain.blockStore.addBlockSeqCB(cb)
	if err != nil {
		return err
	}
	chain.pushseq.addTask(cb)
	return nil
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
