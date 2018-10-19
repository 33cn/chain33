package commands

type RelayOrder2Show struct {
	OrderId       string `json:"orderid"`
	Status        string `json:"status"`
	Creator       string `json:"address"`
	Amount        string `json:"amount"`
	CoinOperation string `json:"coinoperation"`
	Coin          string `json:"coin"`
	CoinAmount    string `json:"coinamount"`
	CoinAddr      string `json:"coinaddr"`
	CoinWaits     uint32 `json:"coinwaits"`
	CreateTime    int64  `json:"createtime"`
	AcceptAddr    string `json:"acceptaddr"`
	AcceptTime    int64  `json:"accepttime"`
	ConfirmTime   int64  `json:"confirmtime"`
	FinishTime    int64  `json:"finishtime"`
	FinishTxHash  string `json:"finishtxhash"`
	Height        int64  `json:"height"`
}
