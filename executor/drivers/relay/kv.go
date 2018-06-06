package relay

import (
	"fmt"

	"gitlab.33.cn/chain33/chain33/types"
)

const (
	relayOrderSCAIH    = "relay-sellorder-scaih:"
	relayOrderCSAIH    = "relay-sellorder-csaih:"
	relayOrderASCIH    = "relay-sellorder-ascih:"
	relayOrderACSIH    = "relay-sellorder-acsih:"
	relayBuyOrderACSIH = "relay-buyorder-acsih:"
	orderIDPrefix      = "mavl-relay-orderid-"

	relayBTCHeaderHash       = "relay-btcheader-hash"
	relayBTCHeaderHeight     = "relay-btcheader-height"
	relayBTCHeaderHeightList = "relay-btcheader-height-list"
	relayBTCHeaderLastHeight = "relay-btcheader-last-height"
	relayBTCHeaderBaseHeight = "relay-btcheader-base-height"
	relayRcvBTCHighestHead   = "relay-rcv-btcheader-ht"
)

func getBtcHeaderKeyHash(hash string) []byte {
	key := fmt.Sprintf(relayBTCHeaderHash+"%s", hash)
	return []byte(key)
}

func getBtcHeaderKeyHeight(height int64) []byte {
	key := fmt.Sprintf(relayBTCHeaderHeight+"%d", height)
	return []byte(key)
}

func getBtcHeaderKeyHeightList(height int64) []byte {
	key := fmt.Sprintf(relayBTCHeaderHeightList+"%d", height)
	return []byte(key)
}

func getBtcHeaderKeyLastHeight() []byte {
	return []byte(relayBTCHeaderLastHeight)
}

func getBtcHeaderKeyBaseHeight() []byte {
	return []byte(relayBTCHeaderBaseHeight)
}

func getBtcHeightListKey() []byte {
	return []byte(relayBTCHeaderHeightList)
}

//特定状态下的定单
func getOrderKeyStatus(order *types.RelayOrder, status int32) []byte {
	key := fmt.Sprintf(relayOrderSCAIH+"%d:%s:%s:%s:%d",
		status, order.Coin, order.CreaterAddr, order.Id, order.Height)
	return []byte(key)
}

//特定coin+状态下的卖单
func getOrderKeyCoin(order *types.RelayOrder, status int32) []byte {
	key := fmt.Sprintf(relayOrderCSAIH+"%s:%d:%s:%s:%d",
		order.Coin, status, order.CreaterAddr, order.Id, order.Height)
	return []byte(key)
}

//特定账户下特定状态的卖单
func getOrderKeyAddrStatus(order *types.RelayOrder, status int32) []byte {
	key := fmt.Sprintf(relayOrderASCIH+"%s:%d:%s:%s:%d",
		order.CreaterAddr, status, order.Coin, order.Id, order.Height)
	return []byte(key)
}

//特定账户下特定coin的卖单
func getOrderKeyAddrCoin(order *types.RelayOrder, status int32) []byte {
	key := fmt.Sprintf(relayOrderACSIH+"%s:%s:%d:%s:%d",
		order.CreaterAddr, order.Coin, status, order.Id, order.Height)
	return []byte(key)
}

func getOrderPrefixStatus(status int32) []byte {
	prefix := fmt.Sprintf(relayOrderSCAIH+"%d:", status)
	return []byte(prefix)
}

func getOrderPrefixCoinStatus(coin string, status int32) []byte {
	prefix := fmt.Sprintf(relayOrderCSAIH+"%s:%d:", coin, status)
	return []byte(prefix)
}

//特定账户下指定token的卖单
func getOrderPrefixAddrCoin(addr string, coin string) []byte {
	key := fmt.Sprintf(relayOrderACSIH+"%s:%s", addr, coin)
	return []byte(key)
}

//特定账户下的卖单
func getOrderPrefixAddr(addr string) []byte {
	return []byte(fmt.Sprintf(relayOrderACSIH+"%s", addr))
}

func getBuyOrderKeyAddr(order *types.RelayOrder, status int32) []byte {
	return []byte(fmt.Sprintf(relayBuyOrderACSIH+"%s:%s:%d:%s:%d",
		order.AcceptAddr, order.Coin, status, order.Id, order.Height))
}

//特定账户下的买单
func getBuyOrderPrefixAddr(addr string) []byte {
	return []byte(fmt.Sprintf(relayBuyOrderACSIH+"%s", addr))
}

func getBuyOrderPrefixAddrCoin(addr, coin string) []byte {
	return []byte(fmt.Sprintf(relayBuyOrderACSIH+"%s:%s", addr, coin))
}

func calcRelayOrderID(hash string) string {
	return orderIDPrefix + hash
}
