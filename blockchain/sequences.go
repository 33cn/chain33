package blockchain

import (
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/types"
)

//通过记录的block序列号获取blockd序列存储的信息
func (chain *BlockChain) GetBlockSequences(requestblock *types.ReqBlocks) (*types.BlockSequences, error) {
	blockLastSeq, _ := chain.blockStore.LoadBlockLastSequence()
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

//处理共识过来的删除block的消息，目前只提供给平行链使用
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

//处理共识过来的add block的消息，目前只提供给平行链使用
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

//处理共识过来的通过blockhash获取seq的消息，只提供add block时的seq，用于平行链block回退
func (chain *BlockChain) ProcGetSeqByHash(hash []byte) (int64, error) {
	if len(hash) == 0 {
		chainlog.Error("ProcGetSeqByHash input hash is null")
		return -1, types.ErrInvalidParam
	}
	seq, err := chain.blockStore.GetSequenceByHash(hash)
	chainlog.Debug("ProcGetSeqByHash", "blockhash", common.ToHex(hash), "seq", seq, "err", err)

	return seq, err
}
