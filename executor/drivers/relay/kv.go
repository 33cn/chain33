package relay

import "fmt"

const (
	relayOrderTSHSES   = "relay-sellorder-tshses:"
	relayOrderASTESS   = "relay-sellorder-astess:"
	relayOrderATSESS   = "relay-sellorder-atsess:"
	relayBuyOrderATSSE = "relay-buyorder-atsse:"
	sellIDPrefix       = "mavl-relay-sell-"
)

//特定token+状态下的卖单
func getRelayOrderKeyToken(token string, status int32, sellOrderID string, sellamount, exchgamount int64, height int64) []byte {
	key := fmt.Sprintf(relayOrderTSHSES+"%s:%d:%d:%d:%d:%s", token, status, height, sellamount, exchgamount, sellOrderID)
	return []byte(key)
}

//特定账户下特定状态的卖单
func getRelayOrderKeyAddrStatus(token string, addr string, status int32, sellOrderID string, sellamount, exchgamount int64) []byte {
	key := fmt.Sprintf(relayOrderASTESS+"%s:%d:%s:%d:%d:%s", addr, status, token, exchgamount, sellamount, sellOrderID)
	return []byte(key)
}

//特定账户下特定token的卖单
func getRelayOrderKeyAddrToken(token string, addr string, status int32, sellOrderID string, sellamount, exchgamount int64) []byte {
	key := fmt.Sprintf(relayOrderATSESS+"%s:%s:%d:%d:%d:%s", addr, token, status, exchgamount, sellamount, sellOrderID)
	return []byte(key)
}

func getRelayOrderPrefixTokenStatus(token string, status int32) []byte {
	prefix := fmt.Sprintf(relayOrderTSHSES+"%s:%d:", token, status)
	return []byte(prefix)
}

//特定账户下指定token的卖单
func getRelayOrderPrefixAddrToken(token string, addr string) []byte {
	key := fmt.Sprintf(relayOrderATSESS+"%s:%s", addr, token)
	return []byte(key)
}

//特定账户下的卖单
func getRelayOrderPrefixAddr(addr string) []byte {
	return []byte(fmt.Sprintf(relayOrderATSESS+"%s", addr))
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
