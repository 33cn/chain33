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

func (l *Lottery) Query_GetLotteryCurrentInfo(param *pty.ReqLotteryInfo) (types.Message, error) {
	lottery, err := findLottery(l.GetStateDB(), param.GetLotteryId())
	if err != nil {
		return nil, err
	}
	return &pty.ReplyLotteryCurrentInfo{lottery.Status,
		lottery.Fund,
		lottery.LastTransToPurState,
		lottery.LastTransToDrawState,
		lottery.TotalPurchasedTxNum,
		lottery.Round,
		lottery.LuckyNumber,
		lottery.LastTransToPurStateOnMain,
		lottery.LastTransToDrawStateOnMain}, nil
}

func (l *Lottery) Query_GetLotteryHistoryLuckyNumber(param *pty.ReqLotteryLuckyHistory) (types.Message, error) {
	return ListLotteryLuckyHistory(l.GetLocalDB(), l.GetStateDB(), param)
}

func (l *Lottery) Query_GetLotteryRoundLuckyNumber(param *pty.ReqLotteryLuckyInfo) (types.Message, error) {
	key := calcLotteryDrawKey(param.LotteryId, param.Round)
	record, err := l.findLotteryDrawRecord(key)
	if err != nil {
		return nil, err
	}
	return record, nil
}

func (l *Lottery) Query_GetLotteryHistoryBuyInfo(param *pty.ReqLotteryBuyHistory) (types.Message, error) {
	return ListLotteryBuyRecords(l.GetLocalDB(), l.GetStateDB(), param)
}

func (l *Lottery) Query_GetLotteryBuyRoundInfo(param *pty.ReqLotteryBuyInfo) (types.Message, error) {
	key := calcLotteryBuyRoundPrefix(param.LotteryId, param.Addr, param.Round)
	record, err := l.findLotteryBuyRecord(key)
	if err != nil {
		return nil, err
	}
	return record, nil
}
