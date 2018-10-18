package executor

import (
	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/relay/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (r *relay) Query_GetRelayOrderByStatus(addrCoins *ty.ReqRelayAddrCoins) (types.Message, error) {
	return r.GetSellOrderByStatus(addrCoins)
}

func (r *relay) Query_GetSellRelayOrder(addrCoins *ty.ReqRelayAddrCoins) (types.Message, error) {
	return r.GetSellRelayOrder(addrCoins)
}

func (r *relay) Query_GetBuyRelayOrder(addrCoins *ty.ReqRelayAddrCoins) (types.Message, error) {
	return r.GetSellOrderByStatus(addrCoins)
}

func (r *relay) Query_GetBTCHeaderList(req *ty.ReqRelayBtcHeaderHeightList) (types.Message, error) {
	db := newBtcStore(r.GetLocalDB())
	return db.getHeadHeightList(req)
}

func (r *relay) Query_GetBTCHeaderCurHeight(req *ty.ReqRelayQryBTCHeadHeight) (types.Message, error) {
	db := newBtcStore(r.GetLocalDB())
	return db.getBtcCurHeight(req)
}
