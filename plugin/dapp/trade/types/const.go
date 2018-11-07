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

// log
const (
	TyLogTradeSellLimit  = 310
	TyLogTradeBuyMarket  = 311
	TyLogTradeSellRevoke = 312

	TyLogTradeSellMarket = 330
	TyLogTradeBuyLimit   = 331
	TyLogTradeBuyRevoke  = 332
)

// 0->not start, 1->on sale, 2->sold out, 3->revoke, 4->expired
const (
	TradeOrderStatusNotStart = iota
	TradeOrderStatusOnSale
	TradeOrderStatusSoldOut
	TradeOrderStatusRevoked
	TradeOrderStatusExpired
	TradeOrderStatusOnBuy
	TradeOrderStatusBoughtOut
	TradeOrderStatusBuyRevoked
)

var SellOrderStatus = map[int32]string{
	TradeOrderStatusNotStart:   "NotStart",
	TradeOrderStatusOnSale:     "OnSale",
	TradeOrderStatusSoldOut:    "SoldOut",
	TradeOrderStatusRevoked:    "Revoked",
	TradeOrderStatusExpired:    "Expired",
	TradeOrderStatusOnBuy:      "OnBuy",
	TradeOrderStatusBoughtOut:  "BoughtOut",
	TradeOrderStatusBuyRevoked: "BuyRevoked",
}

var SellOrderStatus2Int = map[string]int32{
	"NotStart":   TradeOrderStatusNotStart,
	"OnSale":     TradeOrderStatusOnSale,
	"SoldOut":    TradeOrderStatusSoldOut,
	"Revoked":    TradeOrderStatusRevoked,
	"Expired":    TradeOrderStatusExpired,
	"OnBuy":      TradeOrderStatusOnBuy,
	"BoughtOut":  TradeOrderStatusBoughtOut,
	"BuyRevoked": TradeOrderStatusBuyRevoked,
}

var MapSellOrderStatusStr2Int = map[string]int32{
	"onsale":  TradeOrderStatusOnSale,
	"soldout": TradeOrderStatusSoldOut,
	"revoked": TradeOrderStatusRevoked,
}

var MapBuyOrderStatusStr2Int = map[string]int32{
	"onbuy":      TradeOrderStatusOnBuy,
	"boughtout":  TradeOrderStatusBoughtOut,
	"buyrevoked": TradeOrderStatusBuyRevoked,
}

const (
	InvalidStartTime = 0
)
