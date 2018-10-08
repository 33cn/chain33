package types

// trade op
const (
	TradeSellLimit = iota
	TradeBuyMarket
	TradeRevokeSell
	TradeSellMarket
	TradeBuyLimit
	TradeRevokeBuy
)