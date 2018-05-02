package trade

import "fmt"

const (
	sellOrderSHTAS = "token-sellorder-shtas:"
	sellOrderASTS  = "token-sellorder-asts:"
	sellOrderATSS  = "token-sellorder-atss:"
	sellOrderTSPAS = "token-sellorder-tspas:"
	buyOrderAHTSB  = "token-buyorder-ahtsb:"
	buyOrderTAHSB  = "token-buyorder-tahsb:"
	sellIDPrefix   = "mavl-trade-sell-"
	buyIDPrefix    = "mavl-trade-buy-"
)

//特定状态下的卖单
func calcTokenSellOrderKey(token string, addr string, status int32, sellOrderID string, height int64) []byte {
	key := fmt.Sprintf(sellOrderSHTAS+"%d:%d:%s:%s:%s", status, height, token, addr, sellOrderID)
	return []byte(key)
}

//特定账户下特定状态的卖单
func calcOnesSellOrderKeyStatus(token string, addr string, status int32, sellOrderID string) []byte {
	key := fmt.Sprintf(sellOrderASTS+"%s:%d:%s:%s", addr, status, token, sellOrderID)
	return []byte(key)
}

//特定账户下特定token的卖单
func calcOnesSellOrderKeyToken(token string, addr string, status int32, sellOrderID string) []byte {
	key := fmt.Sprintf(sellOrderATSS+"%s:%s:%d:%s", addr, token, status, sellOrderID)
	return []byte(key)
}

//指定token的卖单， 带上价格方便排序
func calcTokensSellOrderKeyStatus(token string, status int32, price int64, addr string, sellOrderID string) []byte {
	key := fmt.Sprintf(sellOrderTSPAS+"%s:%d:%016d:%s:%s", token, status, price, addr, sellOrderID)
	return []byte(key)
}

func calcTokensSellOrderPrefixStatus(token string, status int32) []byte {
	prefix := fmt.Sprintf(sellOrderTSPAS+"%s:%d:", token, status)
	return []byte(prefix)
}

//特定账户下指定token的卖单
func calcOnesSellOrderPrefixToken(token string, addr string) []byte {
	key := fmt.Sprintf(sellOrderATSS+"%s:%s", addr, token)
	return []byte(key)
}

//特定账户下的卖单
func calcOnesSellOrderPrefixAddr(addr string) []byte {
	return []byte(fmt.Sprintf(sellOrderASTS+"%s", addr))
}

//特定状态下的卖单
func calcTokenSellOrderPrefixStatus(status int32) []byte {
	return []byte(fmt.Sprintf(sellOrderSHTAS+"%d", status))
}

//考虑到购买者可能在同一个区块时间针对相同的卖单(sellorder)发起购买操作，所以使用sellid：buytxhash组合的方式生成key
func calcOnesBuyOrderKey(addr string, height int64, token string, sellOrderID string, buyTxHash string) []byte {
	return []byte(fmt.Sprintf(buyOrderAHTSB+"%s:%d:%s:%s:%s", addr, height, token, sellOrderID, buyTxHash))
}

//用于快速查询某个token下的所有成交的买单
func calcBuyOrderKey(addr string, height int64, token string, sellOrderID string, buyTxHash string) []byte {
	return []byte(fmt.Sprintf(buyOrderTAHSB+"%s:%s:%d:%s:%s", token, addr, height, sellOrderID, buyTxHash))
}

//特定账户下的买单
func calcOnesBuyOrderPrefixAddr(addr string) []byte {
	return []byte(fmt.Sprintf(buyOrderAHTSB+"%s", addr))
}

func calcTokenSellID(hash string) string {
	return sellIDPrefix + hash
}

func calcTokenBuyID(hash string) string {
	return buyIDPrefix + hash
}
