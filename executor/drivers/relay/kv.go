package relay

import "fmt"

const (
	sellOrderSHTAS = "token-sellorder-shtas:"
	sellOrderASTS  = "token-sellorder-asts:"
	sellOrderATSS  = "token-sellorder-atss:"
	sellOrderTSPAS = "token-sellorder-tspas:"
	buyOrderAHTSB  = "token-buyorder-ahtsb:"
	buyOrderTAHSB  = "token-buyorder-tahsb:"
	sellIDPrefix   = "mavl-relay-sell-"
)

//特定状态下的卖单
func calcTokenSellOrderKey(token string, addr string, status int32, sellOrderID string, height int64) []byte {
	key := fmt.Sprintf(sellOrderSHTAS+"%d:%d:%s:%s:%s", status, height, token, addr, sellOrderID)
	return []byte(key)
}

func calcRelaySellID(hash string) string {
	return sellIDPrefix + hash
}
