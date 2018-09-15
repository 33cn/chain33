package executor

import (
	"context"
	"errors"

	"gitlab.33.cn/chain33/chain33/types"
)

//different impl on main chain and parachain
func (action *Action) getTxActions(height int64, blockNum int64) ([]*types.TicketAction, error) {
	var txActions []*types.TicketAction
	llog.Error("getTxActions", "height", height, "blockNum", blockNum)
	if !types.IsPara() {
		req := &types.ReqBlocks{height - blockNum + 1, height, false, []string{""}}

		blockDetails, err := action.api.GetBlocks(req)
		if err != nil {
			llog.Error("getTxActions", "height", height, "blockNum", blockNum, "err", err)
			return txActions, err
		}
		for _, block := range blockDetails.Items {
			llog.Error("getTxActions", "blockHeight", block.Block.Height, "blockhash", block.Block.Hash())
			ticketAction, err := action.getMinerTx(block.Block)
			if err != nil {
				return txActions, err
			}
			txActions = append(txActions, ticketAction)
		}
		return txActions, nil
	} else {
		//last block on paraChain
		req := &types.ReqBlocks{height, height, false, []string{""}}

		blockDetails, err := action.api.GetBlocks(req)
		if err != nil {
			llog.Error("getTxActions", "height", height, "blockNum", blockNum, "err", err)
			return txActions, err
		}

		var sequences []int64
		var blockedSeq int64
		block := blockDetails.Items[0]
		blockedSeq, err = action.api.GetBlockedSeq(&types.ReqHash{block.Block.Hash()})
		if err != nil {
			return txActions, err
		}

		for i := 0; i < int(blockNum); i++ {
			blockedSeq -= 1
			sequences = append(sequences, blockedSeq)
		}

		llog.Error("getTxActions", "sequences", sequences)
		savedBlocksOnMain, err := action.GetBlockOnMainBySeq(sequences)
		if err != nil {
			//fatal err sometimes
			panic("")
			llog.Error("getTxActions", "err", err)
			return txActions, err
		}

		if len(savedBlocksOnMain.Items) == 0 {
			return txActions, errors.New("No main block")
		}

		for _, block := range savedBlocksOnMain.Items {
			ticketAction, err := action.getMinerTx(block.Block)
			if err != nil {
				return txActions, err
			}
			txActions = append(txActions, ticketAction)
		}
		return txActions, nil
	}
}

func (action *Action) GetBlockHashFromMainChain(start int64, end int64) (*types.BlockSequences, error) {
	req := &types.ReqBlocks{start, end, true, []string{}}
	blockSeqs, err := action.grpcClient.GetBlockSequences(context.Background(), req)
	if err != nil {
		llog.Error("GetBlockHashFromMainChain", "Error", err.Error())
		return nil, err
	}
	return blockSeqs, nil
}

func (action *Action) GetBlocksByHashesFromMainChain(hashes [][]byte) (*types.BlockDetails, error) {
	req := &types.ReqHashes{hashes}
	blocks, err := action.grpcClient.GetBlockByHashes(context.Background(), req)
	if err != nil {
		llog.Error("GetBlocksByHashesFromMainChain", "Error", err.Error())
		return nil, err
	}
	return blocks, nil
}

//dangerous to get five blocks one time, better to select not detail, supported by main block
func (action *Action) GetBlockOnMainBySeq(seqs []int64) (*types.BlockDetails, error) {
	var hashes [][]byte
	for _, seq := range seqs {
		blockSeqs, err := action.GetBlockHashFromMainChain(seq, seq)
		if err != nil {
			llog.Error("Not found block hash on seq", "start", seq, "end", seq)
			return nil, err
		}

		for _, item := range blockSeqs.Items {
			//ADD
			//if item.Type == 1 {
			hashes = append(hashes, item.Hash)
			//}
		}
	}

	blockDetails, err := action.GetBlocksByHashesFromMainChain(hashes)
	if err != nil {
		return nil, err
	}

	return blockDetails, nil
}

//TransactionDetail
func (action *Action) GetMainHeightByTxHash(txHash []byte) int64 {
	req := &types.ReqHash{txHash}
	txDetail, err := action.grpcClient.QueryTransaction(context.Background(), req)
	if err != nil {
		return -1
	}
	return txDetail.GetHeight()
}
