package executor

import (
	"fmt"

	rTy "gitlab.33.cn/chain33/chain33/plugin/dapp/relay/types"
)

const (
	relayOrderSCAIH          = "relay-sellorder-scaih:"
	relayOrderCSAIH          = "relay-sellorder-csaih:"
	relayOrderASCIH          = "relay-sellorder-ascih:"
	relayOrderACSIH          = "relay-sellorder-acsih:"
	relayBuyOrderACSIH       = "relay-buyorder-acsih:"
	orderIDPrefix            = "mavl-relay-orderid-"
	coinHashPrefix           = "mavl-relay-coinhash-"
	btcLastHead              = "mavl-relay-btclasthead"
	relayBTCHeaderHash       = "relay-btcheader-hash"
	relayBTCHeaderHeight     = "relay-btcheader-height"
	relayBTCHeaderHeightList = "relay-btcheader-height-list"
)

var (
	relayBTCHeaderLastHeight = []byte("relay-btcheader-last-height")
	relayBTCHeaderBaseHeight = []byte("relay-btcheader-base-height")
)

func calcBtcHeaderKeyHash(hash string) []byte {
	key := fmt.Sprintf(relayBTCHeaderHash+"%s", hash)
	return []byte(key)
}

func calcBtcHeaderKeyHeight(height int64) []byte {
	key := fmt.Sprintf(relayBTCHeaderHeight+"%d", height)
	return []byte(key)
}

func calcBtcHeaderKeyHeightList(height int64) []byte {
	key := fmt.Sprintf(relayBTCHeaderHeightList+"%d", height)
	return []byte(key)
}

func calcOrderKeyStatus(order *rTy.RelayOrder, status int32) []byte {
	key := fmt.Sprintf(relayOrderSCAIH+"%d:%s:%s:%s:%d",
		status, order.Coin, order.CreaterAddr, order.Id, order.Height)
	return []byte(key)
}

func calcOrderKeyCoin(order *rTy.RelayOrder, status int32) []byte {
	key := fmt.Sprintf(relayOrderCSAIH+"%s:%d:%s:%s:%d",
		order.Coin, status, order.CreaterAddr, order.Id, order.Height)
	return []byte(key)
}

func calcOrderKeyAddrStatus(order *rTy.RelayOrder, status int32) []byte {
	key := fmt.Sprintf(relayOrderASCIH+"%s:%d:%s:%s:%d",
		order.CreaterAddr, status, order.Coin, order.Id, order.Height)
	return []byte(key)
}

func calcOrderKeyAddrCoin(order *rTy.RelayOrder, status int32) []byte {
	key := fmt.Sprintf(relayOrderACSIH+"%s:%s:%d:%s:%d",
		order.CreaterAddr, order.Coin, status, order.Id, order.Height)
	return []byte(key)
}

func calcOrderPrefixStatus(status int32) []byte {
	prefix := fmt.Sprintf(relayOrderSCAIH+"%d:", status)
	return []byte(prefix)
}

func calcOrderPrefixCoinStatus(coin string, status int32) []byte {
	prefix := fmt.Sprintf(relayOrderCSAIH+"%s:%d:", coin, status)
	return []byte(prefix)
}

func calcOrderPrefixAddrCoin(addr string, coin string) []byte {
	key := fmt.Sprintf(relayOrderACSIH+"%s:%s", addr, coin)
	return []byte(key)
}

func calcOrderPrefixAddr(addr string) []byte {
	return []byte(fmt.Sprintf(relayOrderACSIH+"%s", addr))
}

func calcAcceptKeyAddr(order *rTy.RelayOrder, status int32) []byte {
	if order.AcceptAddr != "" {
		return []byte(fmt.Sprintf(relayBuyOrderACSIH+"%s:%s:%d:%s:%d",
			order.AcceptAddr, order.Coin, status, order.Id, order.Height))
	}
	return nil

}

func calcAcceptPrefixAddr(addr string) []byte {
	return []byte(fmt.Sprintf(relayBuyOrderACSIH+"%s", addr))
}

func calcAcceptPrefixAddrCoin(addr, coin string) []byte {
	return []byte(fmt.Sprintf(relayBuyOrderACSIH+"%s:%s", addr, coin))
}

func calcRelayOrderID(hash string) string {
	return orderIDPrefix + hash
}

func calcCoinHash(hash string) string {
	return coinHashPrefix + hash
}
