package executor

import (
	"fmt"
	"strconv"

	pty "gitlab.33.cn/chain33/chain33/plugin/dapp/trade/types"
	"gitlab.33.cn/chain33/chain33/types"
)

const (
	sellOrderSHTAS = "LODB-trade-sellorder-shtas:"
	sellOrderASTS  = "LODB-trade-sellorder-asts:"
	sellOrderATSS  = "LODB-trade-sellorder-atss:"
	sellOrderTSPAS = "LODB-trade-sellorder-tspas:"
	buyOrderSHTAS  = "LODB-trade-buyorder-shtas:"
	buyOrderASTS   = "LODB-trade-buyorder-asts:"
	buyOrderATSS   = "LODB-trade-buyorder-atss:"
	buyOrderTSPAS  = "LODB-trade-buyorder-tspas:"
	sellIDPrefix   = "mavl-trade-sell-"
	buyIDPrefix    = "mavl-trade-buy-"
	// Addr-Status-Type-Height-Key
	orderASTHK = "LODB-trade-order-asthk:"
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

func calcOnesSellOrderPrefixStatus(addr string, status int32) []byte {
	return []byte(fmt.Sprintf(sellOrderASTS+"%s:%d", addr, status))
}

// 特定状态下的卖单
//func calcTokenSellOrderPrefixStatus(status int32) []byte {
//	return []byte(fmt.Sprintf(sellOrderSHTAS+"%d", status))
//}

// ids
func calcTokenSellID(hash string) string {
	return sellIDPrefix + hash
}

func calcTokenBuyID(hash string) string {
	return buyIDPrefix + hash
}

// buy limit order 4 key, 4prefix
// 特定状态下的买单
func calcTokenBuyOrderKey(token string, addr string, status int32, orderID string, height int64) []byte {
	key := fmt.Sprintf(buyOrderSHTAS+"%d:%d:%s:%s:%s", status, height, token, addr, orderID)
	return []byte(key)
}

// 特定账户下特定状态的买单
func calcOnesBuyOrderKeyStatus(token string, addr string, status int32, orderID string) []byte {
	key := fmt.Sprintf(buyOrderASTS+"%s:%d:%s:%s", addr, status, token, orderID)
	return []byte(key)
}

// 特定账户下特定token的买单
func calcOnesBuyOrderKeyToken(token string, addr string, status int32, orderID string) []byte {
	key := fmt.Sprintf(buyOrderATSS+"%s:%s:%d:%s", addr, token, status, orderID)
	return []byte(key)
}

// 指定token的卖单， 带上价格方便排序
func calcTokensBuyOrderKeyStatus(token string, status int32, price int64, addr string, orderID string) []byte {
	key := fmt.Sprintf(buyOrderTSPAS+"%s:%d:%016d:%s:%s", token, status, price, addr, orderID)
	return []byte(key)
}

func calcTokensBuyOrderPrefixStatus(token string, status int32) []byte {
	prefix := fmt.Sprintf(buyOrderTSPAS+"%s:%d:", token, status)
	return []byte(prefix)
}

// 特定账户下指定token的买单
func calcOnesBuyOrderPrefixToken(token string, addr string) []byte {
	key := fmt.Sprintf(buyOrderATSS+"%s:%s", addr, token)
	return []byte(key)
}

// 特定账户下的买单
func calcOnesBuyOrderPrefixAddr(addr string) []byte {
	return []byte(fmt.Sprintf(buyOrderASTS+"%s", addr))
}

func calcOnesBuyOrderPrefixStatus(addr string, status int32) []byte {
	return []byte(fmt.Sprintf(buyOrderASTS+"%s:%d", addr, status))
}

// 特定帐号下的订单
// 这里状态进行转化, 分成 状态和类型， 状态三种， 类型 两种
//  on:  Onsale Onbuy
//  done:  Soldout boughtOut
//  revoke:  RevokeSell RevokeBuy
// buy/sell 两种类型
//  目前页面是按addr， 状态来
//
func calcOnesOrderKey(addr string, status int32, ty int32, height int64, key string) []byte {
	return []byte(fmt.Sprintf(orderASTHK+"%s:%d:%d:%010d:%s", addr, status, ty, height, key))
}

func calcOnesOrderPrefixStatus(addr string, status int32) []byte {
	return []byte(fmt.Sprintf(orderASTHK+"%s:%d:", addr, status))
}

// 特定状态下的买单
//func calcTokenBuyOrderPrefixStatus(status int32) []byte {
//	return []byte(fmt.Sprintf(buyOrderSHTAS+"%d", status))
//}

func genBuyMarketOrderKeyValue(kv []*types.KeyValue, receipt *pty.ReceiptBuyBase,
	status int32, height int64, value []byte) []*types.KeyValue {

	keyId := receipt.TxHash

	newkey := calcTokenBuyOrderKey(receipt.TokenSymbol, receipt.Owner, status, keyId, height)
	kv = append(kv, &types.KeyValue{newkey, value})

	newkey = calcOnesBuyOrderKeyStatus(receipt.TokenSymbol, receipt.Owner, status, keyId)
	kv = append(kv, &types.KeyValue{newkey, value})

	newkey = calcOnesBuyOrderKeyToken(receipt.TokenSymbol, receipt.Owner, status, keyId)
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

	newkey = calcTokensBuyOrderKeyStatus(receipt.TokenSymbol, status,
		price, receipt.Owner, keyId)
	kv = append(kv, &types.KeyValue{newkey, value})

	st, ty := fromStatus(status)
	newkey = calcOnesOrderKey(receipt.Owner, st, ty, height, keyId)
	kv = append(kv, &types.KeyValue{newkey, value})

	return kv
}

func genSellMarketOrderKeyValue(kv []*types.KeyValue, receipt *pty.ReceiptSellBase, status int32,
	height int64, value []byte) []*types.KeyValue {

	keyID := receipt.TxHash

	newkey := calcTokenSellOrderKey(receipt.TokenSymbol, receipt.Owner, status, keyID, height)
	kv = append(kv, &types.KeyValue{newkey, value})

	newkey = calcOnesSellOrderKeyStatus(receipt.TokenSymbol, receipt.Owner, status, keyID)
	kv = append(kv, &types.KeyValue{newkey, value})

	newkey = calcOnesSellOrderKeyToken(receipt.TokenSymbol, receipt.Owner, status, keyID)
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

	newkey = calcTokensSellOrderKeyStatus(receipt.TokenSymbol, status,
		price, receipt.Owner, keyID)
	kv = append(kv, &types.KeyValue{newkey, value})

	st, ty := fromStatus(status)
	newkey = calcOnesOrderKey(receipt.Owner, st, ty, height, keyID)
	kv = append(kv, &types.KeyValue{newkey, value})

	return kv
}

// make a number as token's price whether cheap or dear
// support 1e8 bty pre token or 1/1e8 bty pre token, [1Coins, 1e16Coins]
// the number in key is used to sort buy orders and pages
func calcPriceOfToken(priceBoardlot, AmountPerBoardlot int64) int64 {
	return 1e8 * priceBoardlot / AmountPerBoardlot
}

func genBuyLimitOrderKeyValue(kv []*types.KeyValue, buyOrder *pty.BuyLimitOrder, status int32, value []byte) []*types.KeyValue {
	newkey := calcTokenBuyOrderKey(buyOrder.TokenSymbol, buyOrder.Address, status, buyOrder.BuyID, buyOrder.Height)
	kv = append(kv, &types.KeyValue{newkey, value})

	newkey = calcOnesBuyOrderKeyStatus(buyOrder.TokenSymbol, buyOrder.Address, status, buyOrder.BuyID)
	kv = append(kv, &types.KeyValue{newkey, value})

	newkey = calcOnesBuyOrderKeyToken(buyOrder.TokenSymbol, buyOrder.Address, status, buyOrder.BuyID)
	kv = append(kv, &types.KeyValue{newkey, value})

	newkey = calcTokensBuyOrderKeyStatus(buyOrder.TokenSymbol, status,
		calcPriceOfToken(buyOrder.PricePerBoardlot, buyOrder.AmountPerBoardlot), buyOrder.Address, buyOrder.BuyID)
	kv = append(kv, &types.KeyValue{newkey, value})

	st, ty := fromStatus(status)
	newkey = calcOnesOrderKey(buyOrder.Address, st, ty, buyOrder.Height, buyOrder.BuyID)
	kv = append(kv, &types.KeyValue{newkey, value})

	return kv
}

func genSellOrderKeyValue(kv []*types.KeyValue, sellorder *pty.SellOrder, status int32, value []byte) []*types.KeyValue {
	newkey := calcTokenSellOrderKey(sellorder.TokenSymbol, sellorder.Address, status, sellorder.SellID, sellorder.Height)
	kv = append(kv, &types.KeyValue{newkey, value})

	newkey = calcOnesSellOrderKeyStatus(sellorder.TokenSymbol, sellorder.Address, status, sellorder.SellID)
	kv = append(kv, &types.KeyValue{newkey, value})

	newkey = calcOnesSellOrderKeyToken(sellorder.TokenSymbol, sellorder.Address, status, sellorder.SellID)
	kv = append(kv, &types.KeyValue{newkey, value})

	newkey = calcTokensSellOrderKeyStatus(sellorder.TokenSymbol, status,
		calcPriceOfToken(sellorder.PricePerBoardlot, sellorder.AmountPerBoardlot), sellorder.Address, sellorder.SellID)
	kv = append(kv, &types.KeyValue{newkey, value})

	st, ty := fromStatus(status)
	newkey = calcOnesOrderKey(sellorder.Address, st, ty, sellorder.Height, sellorder.SellID)
	kv = append(kv, &types.KeyValue{newkey, value})

	return kv
}
