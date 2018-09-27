package types

import "gitlab.33.cn/chain33/chain33/types"

type LotteryDrawLog struct {
}

func (l LotteryDrawLog) Name() string {
	return "LogLotteryDraw"
}

func (l LotteryDrawLog) Decode(msg []byte) (interface{}, error) {
	var logTmp ReceiptLottery
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type LotteryCloseLog struct {
}

func (l LotteryCloseLog) Name() string {
	return "LogLotteryClose"
}

func (l LotteryCloseLog) Decode(msg []byte) (interface{}, error) {
	var logTmp ReceiptLottery
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type LotteryCreateLog struct {
}

func (l LotteryCreateLog) Name() string {
	return "LogLotteryCreate"
}

func (l LotteryCreateLog) Decode(msg []byte) (interface{}, error) {
	var logTmp ReceiptLottery
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type LotteryBuyLog struct {
}

func (l LotteryBuyLog) Name() string {
	return "LogLotteryBuy"
}

func (l LotteryBuyLog) Decode(msg []byte) (interface{}, error) {
	var logTmp ReceiptLottery
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}
