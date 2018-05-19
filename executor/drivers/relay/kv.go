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

	relayBTCHeaderTHP        = "relay-btcheader-thp"
	relayBTCHeaderHT         = "relay-btcheader-ht"
	relayBTCHeaderHash       = "relay-btcheader-hash"
	relayBTCHeaderHeight     = "relay-btcheader-height"
	relayBTCHeaderHeightList = "relay-btcheader-height-list"
	relayBTCHeaderLastHeight = "relay-btcheader-last-height"
	relayRcvBTCHighestHead   = "relay-rcv-btcheader-ht"
)

func getRelayBtcHeaderKeyHash(hash string) []byte {
	key := fmt.Sprintf(relayBTCHeaderHash+"%s", hash)
	return []byte(key)
}

func getRelayBtcHeaderKeyHeight(height int64) []byte {
	key := fmt.Sprintf(relayBTCHeaderHeight+"%d-", height)
	return []byte(key)
}

func getRelayBtcHeaderKeyHeightList(height int64) []byte {
	key := fmt.Sprintf(relayBTCHeaderHeightList+"%d-", height)
	return []byte(key)
}

func getRelayBtcHeaderKeyLastHeight() []byte {
	return []byte(relayBTCHeaderHeightList)
}

func getRelayBtcHeightListKey() []byte {
	return []byte(relayBTCHeaderHeightList)
}

//特定状态下的定单
func getRelayOrderKeyStatus(order *types.RelayOrder, status int32) []byte {
	key := fmt.Sprintf(relayOrderSCAIH+"%d:%s:%s:%s:%d",
		status, order.Exchgcoin, order.Selladdr, order.Orderid, order.Height)
	return []byte(key)
}

//特定coin+状态下的卖单
func getRelayOrderKeyCoin(order *types.RelayOrder, status int32) []byte {
	key := fmt.Sprintf(relayOrderCSAIH+"%s:%d:%s:%s:%d",
		order.Exchgcoin, status, order.Selladdr, order.Orderid, order.Height)
	return []byte(key)
}

//特定账户下特定状态的卖单
func getRelayOrderKeyAddrStatus(order *types.RelayOrder, status int32) []byte {
	key := fmt.Sprintf(relayOrderASCIH+"%s:%d:%s:%s:%d",
		order.Selladdr, status, order.Exchgcoin, order.Orderid, order.Height)
	return []byte(key)
}

//特定账户下特定coin的卖单
func getRelayOrderKeyAddrCoin(order *types.RelayOrder, status int32) []byte {
	key := fmt.Sprintf(relayOrderACSIH+"%s:%s:%d:%s:%d",
		order.Selladdr, order.Exchgcoin, status, order.Orderid, order.Height)
	return []byte(key)
}

func getRelayOrderPrefixStatus(status int32) []byte {
	prefix := fmt.Sprintf(relayOrderSCAIH+"%d:", status)
	return []byte(prefix)
}

func getRelayOrderPrefixCoinStatus(coin string, status int32) []byte {
	prefix := fmt.Sprintf(relayOrderCSAIH+"%s:%d:", coin, status)
	return []byte(prefix)
}

//特定账户下指定token的卖单
func getRelayOrderPrefixAddrCoin(addr string, coin string) []byte {
	key := fmt.Sprintf(relayOrderACSIH+"%s:%s", addr, coin)
	return []byte(key)
}

//特定账户下的卖单
func getRelayOrderPrefixAddr(addr string) []byte {
	return []byte(fmt.Sprintf(relayOrderACSIH+"%s", addr))
}

func getBuyOrderKeyAddr(order *types.RelayOrder, status int32) []byte {
	return []byte(fmt.Sprintf(relayBuyOrderACSIH+"%s:%s:%d:%s:%d",
		order.Buyeraddr, order.Exchgcoin, status, order.Orderid, order.Height))
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
