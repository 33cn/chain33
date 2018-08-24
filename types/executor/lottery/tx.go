package lottery

type LotteryCreateTx struct {
	PurchasePeriod int64 `json:"purchasePeriod"`
	ShowPeriod     int64 `json:"showPeriod"`
	MaxPurchaseNum int64 `json:"maxPurchaseNum"`
	Fee            int64 `json:"fee"`
}

type LotteryBuyTx struct {
	LotteryId string `json:"lotteryId"`
	Amount    int64  `json:"amount"`
	HashValue string `json:"hashValue"`
	Way       int64  `json:"way"`
	Fee       int64  `json:"fee"`
}

type LotteryShowTx struct {
	LotteryId string `json:"lotteryId"`
	Secret    string `json:"secret"`
	TxHash    string `json:"txHash"`
	Number    int64  `json:"number`
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
