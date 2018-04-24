package relay

import "fmt"

const (
	relaySellOrderTSHSES = "relay-sellorder-tshses:"
	relaySellOrderASTESS = "relay-sellorder-astess:"
	relaySellOrderATSESS = "relay-sellorder-atsess:"
	relayBuyOrderATSSE   = "relay-buyorder-atsse:"
	sellIDPrefix         = "mavl-relay-sell-"
)

//特定token+状态下的卖单
func getSellOrderKeyToken(token string, status int32, sellOrderID string, sellamount, exchgamount int64, height int64) []byte {
	key := fmt.Sprintf(relaySellOrderTSHSES+"%s:%d:%d:%d:%d:%s", token, status, height, sellamount, exchgamount, sellOrderID)
	return []byte(key)
}

//特定账户下特定状态的卖单
func getSellOrderKeyAddrStatus(token string, addr string, status int32, sellOrderID string, sellamount, exchgamount int64) []byte {
	key := fmt.Sprintf(relaySellOrderASTESS+"%s:%d:%s:%d:%d:%s", addr, status, token, exchgamount, sellamount, sellOrderID)
	return []byte(key)
}

//特定账户下特定token的卖单
func getSellOrderKeyAddrToken(token string, addr string, status int32, sellOrderID string, sellamount, exchgamount int64) []byte {
	key := fmt.Sprintf(relaySellOrderATSESS+"%s:%s:%d:%d:%d:%s", addr, token, status, exchgamount, sellamount, sellOrderID)
	return []byte(key)
}

func getSellOrderPrefixTokenStatus(token string, status int32) []byte {
	prefix := fmt.Sprintf(relaySellOrderTSHSES+"%s:%d:", token, status)
	return []byte(prefix)
}

//特定账户下指定token的卖单
func getSellOrderPrefixAddrToken(token string, addr string) []byte {
	key := fmt.Sprintf(relaySellOrderATSESS+"%s:%s", addr, token)
	return []byte(key)
}

//特定账户下的卖单
func getSellOrderPrefixAddr(addr string) []byte {
	return []byte(fmt.Sprintf(relaySellOrderATSESS+"%s", addr))
}

func getBuyOrderKeyAddr(addr string, token string, sellOrderID string, sellamount, exchgamount int64) []byte {
	return []byte(fmt.Sprintf(relayBuyOrderATSSE+"%s:%s:%s:%d:%d", addr, token, sellOrderID, sellamount, exchgamount))
}

//特定账户下的买单
func getBuyOrderPrefixAddr(addr string) []byte {
	return []byte(fmt.Sprintf(relayBuyOrderATSSE+"%s", addr))
}

func calcRelaySellID(hash string) string {
	return sellIDPrefix + hash
}
