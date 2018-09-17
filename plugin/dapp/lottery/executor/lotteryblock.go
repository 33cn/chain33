package executor

import (
	"context"

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
		//block height on main
		mainHeight := action.GetMainHeightByTxHash(action.txhash)
		if mainHeight < 0 {
			llog.Error("LotteryCreate", "mainHeight", mainHeight)
			panic("")
		}

		blockDetails, err := action.GetBlocksOnMain(mainHeight-blockNum, mainHeight-1)
		if err != nil {
			llog.Error("LotteryCreate", "mainHeight", mainHeight)
			panic("")
		}

		for _, block := range blockDetails.Items {
			ticketAction, err := action.getMinerTx(block.Block)
			if err != nil {
				return txActions, err
			}
			txActions = append(txActions, ticketAction)
		}
		return txActions, nil
	}
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

func (action *Action) GetBlocksOnMain(start int64, end int64) (*types.BlockDetails, error) {
	req := &types.ReqBlocks{start, end, false, []string{""}}

	reply, err := action.grpcClient.GetBlocks(context.Background(), req)
	if err != nil {
		llog.Error("GetBlocksOnMain", "start", start, "end", end, "err", err)
		return nil, err
	}

	var blockDetails types.BlockDetails

	err = types.Decode(reply.Msg, &blockDetails)
	if err != nil {
		llog.Error("GetBlocksOnMain", "err", err)
		return nil, err
	}

	return &blockDetails, nil
}

func (action *Action) getMinerTx(current *types.Block) (*types.TicketAction, error) {
	//检查第一个笔交易的execs, 以及执行状态
	if len(current.Txs) == 0 {
		return nil, types.ErrEmptyTx
	}
	baseTx := current.Txs[0]
	//判断交易类型和执行情况
	var ticketAction types.TicketAction
	err := types.Decode(baseTx.GetPayload(), &ticketAction)
	if err != nil {
		return nil, err
	}
	if ticketAction.GetTy() != types.TicketActionMiner {
		return nil, types.ErrCoinBaseTxType
	}
	//判断交易执行是否OK
	if ticketAction.GetMiner() == nil {
		return nil, types.ErrEmptyMinerTx
	}
	return &ticketAction, nil
}
