package executor

import (
	pty "gitlab.33.cn/chain33/chain33/plugin/dapp/lottery/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (l *Lottery) Query_GetLotteryNormalInfo(param *pty.ReqLotteryInfo) (types.Message, error) {
	lottery, err := findLottery(l.GetStateDB(), param.GetLotteryId())
	if err != nil {
		return nil, err
	}
	return &pty.ReplyLotteryNormalInfo{lottery.CreateHeight,
		lottery.PurBlockNum,
		lottery.DrawBlockNum,
		lottery.CreateAddr}, nil
}

func (l *Lottery) Query_GetLotteryPurchaseAddr(param *pty.ReqLotteryInfo) (types.Message, error) {
	lottery, err := findLottery(l.GetStateDB(), param.GetLotteryId())
	if err != nil {
		return nil, err
	}
	reply := &pty.ReplyLotteryPurchaseAddr{}
	for addr := range lottery.Records {
		reply.Address = append(reply.Address, addr)
	}
	//lottery.Records
	return reply, nil
}

func (l *Lottery) Query_GetLotteryCurrentInfo(param *pty.ReqLotteryInfo) (types.Message, error) {
	lottery, err := findLottery(l.GetStateDB(), param.GetLotteryId())
	if err != nil {
		return nil, err
	}
	reply := &pty.ReplyLotteryCurrentInfo{Status: lottery.Status,
		Fund:                       lottery.Fund,
		LastTransToPurState:        lottery.LastTransToPurState,
		LastTransToDrawState:       lottery.LastTransToDrawState,
		TotalPurchasedTxNum:        lottery.TotalPurchasedTxNum,
		Round:                      lottery.Round,
		LuckyNumber:                lottery.LuckyNumber,
		LastTransToPurStateOnMain:  lottery.LastTransToPurStateOnMain,
		LastTransToDrawStateOnMain: lottery.LastTransToDrawStateOnMain,
		PurBlockNum:                lottery.PurBlockNum,
		DrawBlockNum:               lottery.DrawBlockNum,
		MissingRecords:             lottery.MissingRecords,
	}
	return reply, nil
}

func (l *Lottery) Query_GetLotteryHistoryLuckyNumber(param *pty.ReqLotteryLuckyHistory) (types.Message, error) {
	return ListLotteryLuckyHistory(l.GetLocalDB(), l.GetStateDB(), param)
}

func (l *Lottery) Query_GetLotteryRoundLuckyNumber(param *pty.ReqLotteryLuckyInfo) (types.Message, error) {
	//	var req pty.ReqLotteryLuckyInfo
	var records []*pty.LotteryDrawRecord
	//	err := types.Decode(param, &req)
	//if err != nil {
	//	return nil, err
	//}
	for _, round := range param.Round {
		key := calcLotteryDrawKey(param.LotteryId, round)
		record, err := l.findLotteryDrawRecord(key)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	return &pty.LotteryDrawRecords{Records: records}, nil
}

func (l *Lottery) Query_GetLotteryHistoryBuyInfo(param *pty.ReqLotteryBuyHistory) (types.Message, error) {
	return ListLotteryBuyRecords(l.GetLocalDB(), l.GetStateDB(), param)
}

func (l *Lottery) Query_GetLotteryBuyRoundInfo(param *pty.ReqLotteryBuyInfo) (types.Message, error) {
	key := calcLotteryBuyRoundPrefix(param.LotteryId, param.Addr, param.Round)
	record, err := l.findLotteryBuyRecords(key)
	if err != nil {
		return nil, err
	}
	return record, nil
}
