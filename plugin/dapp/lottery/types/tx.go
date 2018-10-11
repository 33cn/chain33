package types

type LotteryCreateTx struct {
	PurBlockNum  int64 `json:"purBlockNum"`
	DrawBlockNum int64 `json:"drawBlockNum"`
	Fee          int64 `json:"fee"`
}

type LotteryBuyTx struct {
	LotteryId string `json:"lotteryId"`
	Amount    int64  `json:"amount"`
	Number    int64  `json:"number"`
	Way       int64  `json:"way"`
	Fee       int64  `json:"fee"`
}

type LotteryDrawTx struct {
	LotteryId string `json:"lotteryId"`
	Fee       int64  `json:"fee"`
}

type LotteryCloseTx struct {
	LotteryId string `json:"lotteryId"`
	Fee       int64  `json:"fee"`
}
