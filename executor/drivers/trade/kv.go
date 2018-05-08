package trade

import (
	"fmt"
	"gitlab.33.cn/chain33/chain33/types"
	"strconv"
)

const (
	sellOrderSHTAS       = "token-sellorder-shtas:"
	sellOrderASTS        = "token-sellorder-asts:"
	sellOrderATSS        = "token-sellorder-atss:"
	sellOrderTSPAS       = "token-sellorder-tspas:"
	buyOrderAHTSB        = "token-buyorder-ahtsb:"
	buyOrderTAHSB        = "token-buyorder-tahsb:"
	buyLimitOrderSHTAS   = "token-buyorder-shtas:"
	buyLimitOrderASTS    = "token-buyorder-asts:"
	buyLimitOrderATSS    = "token-buyorder-atss:"
	buyLimitOrderTSPAS   = "token-buyorder-tspas:"
	SellMarketOrderAHTSB = "token-sellmarketorder-ahtsb:"
	SellMarketOrderTAHSB = "token-sellmarketorder-tahsb:"
	sellIDPrefix         = "mavl-trade-sell-"
	buyIDPrefix          = "mavl-trade-buy-"
	sellOrderPrefix      = "token-sellorder"
	buyOrderPrefix       = "token-buyorder"
)

// sell order 4 key, 4prefix
// 特定状态下的卖单
func calcTokenSellOrderKey(token string, addr string, status int32, sellOrderID string, height int64) []byte {
	key := fmt.Sprintf(sellOrderSHTAS+"%d:%d:%s:%s:%s", status, height, token, addr, sellOrderID)
	return []byte(key)
}

// 特定账户下特定状态的卖单
func calcOnesSellOrderKeyStatus(token string, addr string, status int32, sellOrderID string) []byte {
	key := fmt.Sprintf(sellOrderASTS+"%s:%d:%s:%s", addr, status, token, sellOrderID)
	return []byte(key)
}

// 特定账户下特定token的卖单
func calcOnesSellOrderKeyToken(token string, addr string, status int32, sellOrderID string) []byte {
	key := fmt.Sprintf(sellOrderATSS+"%s:%s:%d:%s", addr, token, status, sellOrderID)
	return []byte(key)
}

// 指定token的卖单， 带上价格方便排序
func calcTokensSellOrderKeyStatus(token string, status int32, price int64, addr string, sellOrderID string) []byte {
	key := fmt.Sprintf(sellOrderTSPAS+"%s:%d:%016d:%s:%s", token, status, price, addr, sellOrderID)
	return []byte(key)
}

func calcTokensSellOrderPrefixStatus(token string, status int32) []byte {
	prefix := fmt.Sprintf(sellOrderTSPAS+"%s:%d:", token, status)
	return []byte(prefix)
}

// 特定账户下指定token的卖单
func calcOnesSellOrderPrefixToken(token string, addr string) []byte {
	key := fmt.Sprintf(sellOrderATSS+"%s:%s", addr, token)
	return []byte(key)
}

// 特定账户下的卖单
func calcOnesSellOrderPrefixAddr(addr string) []byte {
	return []byte(fmt.Sprintf(sellOrderASTS+"%s", addr))
}

// 特定状态下的卖单
func calcTokenSellOrderPrefixStatus(status int32) []byte {
	return []byte(fmt.Sprintf(sellOrderSHTAS+"%d", status))
}

// buy order 2 key, 1 prefix
// 考虑到购买者可能在同一个区块时间针对相同的卖单(sellorder)发起购买操作，所以使用sellid：buytxhash组合的方式生成key
func calcOnesBuyOrderKey(addr string, height int64, token string, sellOrderID string, buyTxHash string) []byte {
	return []byte(fmt.Sprintf(buyOrderAHTSB+"%s:%d:%s:%s:%s", addr, height, token, sellOrderID, buyTxHash))
}

// 用于快速查询某个token下的所有成交的买单
func calcBuyOrderKey(addr string, height int64, token string, sellOrderID string, buyTxHash string) []byte {
	return []byte(fmt.Sprintf(buyOrderTAHSB+"%s:%s:%d:%s:%s", token, addr, height, sellOrderID, buyTxHash))
}

// 特定账户下的买单
func calcOnesBuyOrderPrefixAddr(addr string) []byte {
	return []byte(fmt.Sprintf(buyOrderAHTSB+"%s", addr))
}

// ids
func calcTokenSellID(hash string) string {
	return sellIDPrefix + hash
}

func calcTokenBuyID(hash string) string {
	return buyIDPrefix + hash
}

// buy limit order 4 key, 4prefix
// 特定状态下的买单
func calcTokenBuyLimitOrderKey(token string, addr string, status int32, orderID string, height int64) []byte {
	key := fmt.Sprintf(buyLimitOrderSHTAS+"%d:%d:%s:%s:%s", status, height, token, addr, orderID)
	return []byte(key)
}

// 特定账户下特定状态的买单
func calcOnesBuyLimitOrderKeyStatus(token string, addr string, status int32, orderID string) []byte {
	key := fmt.Sprintf(buyLimitOrderASTS+"%s:%d:%s:%s", addr, status, token, orderID)
	return []byte(key)
}

// 特定账户下特定token的买单
func calcOnesBuyLimitOrderKeyToken(token string, addr string, status int32, orderID string) []byte {
	key := fmt.Sprintf(buyLimitOrderATSS+"%s:%s:%d:%s", addr, token, status, orderID)
	return []byte(key)
}

// 指定token的卖单， 带上价格方便排序
func calcTokensBuyLimitOrderKeyStatus(token string, status int32, price int64, addr string, orderID string) []byte {
	key := fmt.Sprintf(buyLimitOrderTSPAS+"%s:%d:%016d:%s:%s", token, status, price, addr, orderID)
	return []byte(key)
}

func calcTokensBuyLimitOrderPrefixStatus(token string, status int32) []byte {
	prefix := fmt.Sprintf(buyLimitOrderTSPAS+"%s:%d:", token, status)
	return []byte(prefix)
}

// 特定账户下指定token的买单
func calcOnesBuyLimitOrderPrefixToken(token string, addr string) []byte {
	key := fmt.Sprintf(buyLimitOrderATSS+"%s:%s", addr, token)
	return []byte(key)
}

// 特定账户下的买单
func calcOnesBuyLimitOrderPrefixAddr(addr string) []byte {
	return []byte(fmt.Sprintf(buyLimitOrderASTS+"%s", addr))
}

// 特定状态下的买单
func calcTokenBuyLimitOrderPrefixStatus(status int32) []byte {
	return []byte(fmt.Sprintf(buyLimitOrderSHTAS+"%d", status))
}

// sell market order 2 key, 1 prefix
// 考虑到购买者可能在同一个区块时间针对相同的buy limit order 发起操作，所以使用id：selltxhash组合的方式生成key
func calcOnesSellMarketOrderKey(addr string, height int64, token string, orderID string, sellTxHash string) []byte {
	return []byte(fmt.Sprintf(SellMarketOrderAHTSB+"%s:%d:%s:%s:%s", addr, height, token, orderID, sellTxHash))
}

// 用于快速查询某个token下的所有成交的卖单
func calcSellMarketOrderKey(addr string, height int64, token string, orderID string, sellTxHash string) []byte {
	return []byte(fmt.Sprintf(SellMarketOrderTAHSB+"%s:%s:%d:%s:%s", token, addr, height, orderID, sellTxHash))
}

//特定账户下的买单
func calcOnesSellMarketOrderPrefixAddr(addr string) []byte {
	return []byte(fmt.Sprintf(SellMarketOrderAHTSB+"%s", addr))
}

func genBuyMarketOrderKeyValue(kv []*types.KeyValue, receipt *types.ReceiptBuyBase,
	status int32, height int64, value []byte) []*types.KeyValue {

	keyId := receipt.Txhash

	newkey := calcTokenBuyLimitOrderKey(receipt.TokenSymbol, receipt.Owner, status, keyId, height)
	kv = append(kv, &types.KeyValue{newkey, value})

	newkey = calcOnesBuyLimitOrderKeyStatus(receipt.TokenSymbol, receipt.Owner, status, keyId)
	kv = append(kv, &types.KeyValue{newkey, value})

	newkey = calcOnesBuyLimitOrderKeyToken(receipt.TokenSymbol, receipt.Owner, status, keyId)
	kv = append(kv, &types.KeyValue{newkey, value})


	priceBoardlot, err := strconv.ParseFloat(receipt.PricePerBoardlot, 64)
	if err != nil {
		panic(err)
	}
	priceBoardlotInt64 := int64(priceBoardlot * float64(types.TokenPrecision))
	AmountPerBoardlot, err := strconv.ParseFloat(receipt.AmountPerBoardlot, 64)
	if err != nil {
		panic(err)
	}
	AmountPerBoardlotInt64 := int64(AmountPerBoardlot * float64(types.Coin))
	price := calcPriceOfToken(priceBoardlotInt64, AmountPerBoardlotInt64)

	newkey = calcTokensBuyLimitOrderKeyStatus(receipt.TokenSymbol, status,
		price, receipt.Owner, keyId)
	kv = append(kv, &types.KeyValue{newkey, value})

	return kv
}

func genSellMarketOrderKeyValue(kv []*types.KeyValue, receipt *types.ReceiptTradeBase, status int32,
	height int64, value []byte) []*types.KeyValue {

	keyID := receipt.Txhash

	newkey := calcTokenSellOrderKey(receipt.Tokensymbol, receipt.Owner, status, keyID, height)
	kv = append(kv, &types.KeyValue{newkey, value})

	newkey = calcOnesSellOrderKeyStatus(receipt.Tokensymbol, receipt.Owner, status, keyID)
	kv = append(kv, &types.KeyValue{newkey, value})

	newkey = calcOnesSellOrderKeyToken(receipt.Tokensymbol, receipt.Owner, status, keyID)
	kv = append(kv, &types.KeyValue{newkey, value})

	priceBoardlot, err := strconv.ParseFloat(receipt.Priceperboardlot, 64)
	if err != nil {
		panic(err)
	}
	priceBoardlotInt64 := int64(priceBoardlot * float64(types.TokenPrecision))
	AmountPerBoardlot, err := strconv.ParseFloat(receipt.Amountperboardlot, 64)
	if err != nil {
		panic(err)
	}
	AmountPerBoardlotInt64 := int64(AmountPerBoardlot * float64(types.Coin))
	price := calcPriceOfToken(priceBoardlotInt64, AmountPerBoardlotInt64)

	newkey = calcTokensSellOrderKeyStatus(receipt.Tokensymbol, status,
		price, receipt.Owner, keyID)
	kv = append(kv, &types.KeyValue{newkey, value})

	return kv
}

// make a number as token's price whether cheap or dear
// support 1e8 bty pre token or 1/1e8 bty pre token, [1Coins, 1e16Coins]
// the number in key is used to sort buy orders and pages
func calcPriceOfToken(priceBoardlot, AmountPerBoardlot int64) int64 {
	return 1e8 * priceBoardlot / AmountPerBoardlot
}

func genBuyLimitOrderKeyValue(kv []*types.KeyValue, buyOrder *types.BuyLimitOrder, status int32, value []byte) []*types.KeyValue {
	newkey := calcTokenBuyLimitOrderKey(buyOrder.TokenSymbol, buyOrder.Address, status, buyOrder.Buyid, buyOrder.Height)
	kv = append(kv, &types.KeyValue{newkey, value})

	newkey = calcOnesBuyLimitOrderKeyStatus(buyOrder.TokenSymbol, buyOrder.Address, status, buyOrder.Buyid)
	kv = append(kv, &types.KeyValue{newkey, value})

	newkey = calcOnesBuyLimitOrderKeyToken(buyOrder.TokenSymbol, buyOrder.Address, status, buyOrder.Buyid)
	kv = append(kv, &types.KeyValue{newkey, value})

	newkey = calcTokensBuyLimitOrderKeyStatus(buyOrder.TokenSymbol, status,
		calcPriceOfToken(buyOrder.PricePerBoardlot, buyOrder.AmountPerBoardlot), buyOrder.Address, buyOrder.Buyid)
	kv = append(kv, &types.KeyValue{newkey, value})

	return kv
}

func genSellOrderKeyValue(kv []*types.KeyValue, sellorder *types.SellOrder, status int32, value []byte) []*types.KeyValue {
	newkey := calcTokenSellOrderKey(sellorder.Tokensymbol, sellorder.Address, status, sellorder.Sellid, sellorder.Height)
	kv = append(kv, &types.KeyValue{newkey, value})

	newkey = calcOnesSellOrderKeyStatus(sellorder.Tokensymbol, sellorder.Address, status, sellorder.Sellid)
	kv = append(kv, &types.KeyValue{newkey, value})

	newkey = calcOnesSellOrderKeyToken(sellorder.Tokensymbol, sellorder.Address, status, sellorder.Sellid)
	kv = append(kv, &types.KeyValue{newkey, value})

	newkey = calcTokensSellOrderKeyStatus(sellorder.Tokensymbol, status,
		calcPriceOfToken(sellorder.Priceperboardlot, sellorder.Amountperboardlot), sellorder.Address, sellorder.Sellid)
	kv = append(kv, &types.KeyValue{newkey, value})

	return kv
}