package commands

type TradeOrderResult struct {
	TokenSymbol       string `json:"tokenSymbol"`
	Owner             string `json:"owner"`
	AmountPerBoardlot string `json:"amountPerBoardlot"`
	MinBoardlot       int64  `json:"minBoardlot"`
	PricePerBoardlot  string `json:"pricePerBoardlot"`
	TotalBoardlot     int64  `json:"totalBoardlot"`
	TradedBoardlot    int64  `json:"tradedBoardlot"`
	BuyID             string `json:"buyID"`
	Status            int32  `json:"status"`
	SellID            string `json:"sellID"`
	TxHash            string `json:"txHash"`
	Height            int64  `json:"height"`
	Key               string `json:"key"`
	BlockTime         int64  `json:"blockTime"`
	IsSellOrder       bool   `json:"isSellOrder"`
}

type ReplySellOrdersResult struct {
	SellOrders []*TradeOrderResult `json:"sellOrders"`
}

type ReplyBuyOrdersResult struct {
	BuyOrders []*TradeOrderResult `json:"buyOrders"`
}

type ReplyTradeOrdersResult struct {
	Orders []*TradeOrderResult `json:"orders"`
}
