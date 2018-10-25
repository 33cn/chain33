package types

type TradeSellTx struct {
	TokenSymbol       string `json:"tokenSymbol"`
	AmountPerBoardlot int64  `json:"amountPerBoardlot"`
	MinBoardlot       int64  `json:"minBoardlot"`
	PricePerBoardlot  int64  `json:"pricePerBoardlot"`
	TotalBoardlot     int64  `json:"totalBoardlot"`
	Fee               int64  `json:"fee"`
	AssetExec         string `json:"assetExec"`
}

type TradeBuyTx struct {
	SellID      string `json:"sellID"`
	BoardlotCnt int64  `json:"boardlotCnt"`
	Fee         int64  `json:"fee"`
}

type TradeRevokeTx struct {
	SellID string `json:"sellID,"`
	Fee    int64  `json:"fee"`
}

type TradeBuyLimitTx struct {
	TokenSymbol       string `json:"tokenSymbol"`
	AmountPerBoardlot int64  `json:"amountPerBoardlot"`
	MinBoardlot       int64  `json:"minBoardlot"`
	PricePerBoardlot  int64  `json:"pricePerBoardlot"`
	TotalBoardlot     int64  `json:"totalBoardlot"`
	Fee               int64  `json:"fee"`
	AssetExec         string `json:"assetExec"`
}

type TradeSellMarketTx struct {
	BuyID       string `json:"buyID"`
	BoardlotCnt int64  `json:"boardlotCnt"`
	Fee         int64  `json:"fee"`
}

type TradeRevokeBuyTx struct {
	BuyID string `json:"buyID,"`
	Fee   int64  `json:"fee"`
}
