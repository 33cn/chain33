package relay

import (
	log "github.com/inconshreveable/log15"

	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/types"
)

/*
relay执行器支持跨链token的创建和交易，

主要提供操作有以下几种：
1）挂BTY单出售购买BTC 或者已有BTC买别人的BTY(未实现)；
2）挂TOKEN单出售购买BTC 或者已有BTC买别人的TOKEN（未实现）；
3）买家购买指定的卖单；
4）买家交易相应数额的BTC给卖家，并且提取交易信息在chain33上通过relay验证，验证通过获取相应BTY
5）撤销卖单；
6）插销买单

*/

///////executor.go////////
//_ "gitlab.33.cn/chain33/chain33/executor/drivers/relay"

var relaylog = log.New("module", "execs.relay")

func Init() {
	r := newRelay()
	drivers.Register(r.GetName(), r, types.ForkV7AddRelay) //TODO: ForkV7AddRelay
}

type relay struct {
	drivers.DriverBase
	btcstore relayBTCStore
}

func newRelay() *relay {
	r := &relay{}
	r.SetChild(r)
	r.btcstore.r = r
	return r
}

func (r *relay) GetName() string {
	return "relay"
}

func (r *relay) Clone() drivers.Driver {
	clone := &relay{}
	clone.DriverBase = *(r.DriverBase.Clone().(*drivers.DriverBase))
	clone.SetChild(clone)
	clone.btcstore.r = clone

	return clone
}

func (r *relay) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	var action types.RelayAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return nil, err
	}
	relaylog.Info("exec relay tx", "tx hash", common.Bytes2Hex(tx.Hash()), "Ty", action.GetTy())

	actiondb := newRelayAction(r, tx)
	switch action.GetTy() {
	case types.RelayActionSell:
		return actiondb.relaySell(action.GetRsell())

	case types.RelayActionRevokeSell:
		return actiondb.relayRevokeSell(action.GetRrevokesell())

	case types.RelayActionBuy:
		return actiondb.relayBuy(action.GetRbuy())

	case types.RelayActionRevokeBuy:
		return actiondb.relayRevokeBuy(action.GetRrevokebuy())

	//orderid, txhash
	case types.RelayActionConfirmTx:
		return actiondb.relayConfirmTx(action.GetRconfirmtx())

	// orderid, rawTx, index sibling, blockhash
	case types.RelayActionVerifyTx:
		return actiondb.relayVerifyTx(action.GetRverify(), r)

	// orderid, rawTx, index sibling, blockhash
	case types.RelayActionVerifyBTCTx:
		return actiondb.relayVerifyBTCTx(action.GetRverifybtc(), r)

	case types.RelayActionRcvBTCHeaders:
		return actiondb.relaySaveBTCHeader(action.GetBtcHeaders())
	default:
		return nil, types.ErrActionNotSupport
	}
}

//获取运行状态名
func (r *relay) GetActionName(tx *types.Transaction) string {
	return tx.ActionName()
}

func (r *relay) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := r.DriverBase.ExecLocal(tx, receipt, index)
	if err != nil {
		return nil, err
	}

	if receipt.GetTy() != types.ExecOk {
		return set, nil
	}

	for i := 0; i < len(receipt.Logs); i++ {
		item := receipt.Logs[i]
		if item.Ty == types.TyLogRelaySell || item.Ty == types.TyLogRelayRevokeSell {
			var receipt types.ReceiptRelaySell
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err)
			}
			kv := r.saveSell([]byte(receipt.Base.Orderid), item.Ty)
			set.KV = append(set.KV, kv...)
		} else if item.Ty == types.TyLogRelayBuy || item.Ty == types.TyLogRelayRevokeBuy {
			var receipt types.ReceiptRelayBuy
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err)
			}

			kv := r.saveBuy([]byte(receipt.Orderid), item.Ty)
			set.KV = append(set.KV, kv...)
		} else if item.Ty == types.TyLogRelayRcvBTCHead {

			var receipt types.ReceiptRelayRcvBTCHeaders
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err)
			}

			kv := r.btcstore.saveBlockHead(receipt.Base)
			set.KV = append(set.KV, kv...)
		}
	}

	return set, nil
}

func (r *relay) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := r.DriverBase.ExecDelLocal(tx, receipt, index)
	if err != nil {
		return nil, err
	}

	if receipt.GetTy() != types.ExecOk {
		return set, nil
	}

	for i := 0; i < len(receipt.Logs); i++ {
		item := receipt.Logs[i]
		if item.Ty == types.TyLogRelaySell || item.Ty == types.TyLogRelayRevokeSell {
			var receipt types.ReceiptRelaySell
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err)
			}
			kv := r.deletSell([]byte(receipt.Base.Orderid), item.Ty)
			set.KV = append(set.KV, kv...)
		} else if item.Ty == types.TyLogRelayBuy || item.Ty == types.TyLogRelayRevokeBuy {
			var receipt types.ReceiptRelayBuy
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err)
			}
			kv := r.deleteBuy([]byte(receipt.Orderid), item.Ty)
			set.KV = append(set.KV, kv...)
		}
	}

	return set, nil

}

func (r *relay) Query(funcName string, params []byte) (types.Message, error) {
	switch funcName {
	//按状态查询卖单
	case "GetRelayOrderByStatus":
		var addrCoins types.ReqRelayAddrCoins

		err := types.Decode(params, &addrCoins)
		if err != nil {
			return nil, err
		}
		return r.GetSellOrderByStatus(&addrCoins)
	//查询某个特定用户的一个或者多个coin兑换的卖单,包括所有状态的卖单
	case "GetSellRelayOrder":
		var addrCoins types.ReqRelayAddrCoins
		err := types.Decode(params, &addrCoins)
		if err != nil {
			return nil, err
		}
		return r.GetSellRelayOrder(&addrCoins)
	case "GetBuyRelayOrder":
		var addrCoins types.ReqRelayAddrCoins
		err := types.Decode(params, &addrCoins)
		if err != nil {
			return nil, err
		}
		return r.GetBuyRelayOrder(&addrCoins)
	case "GetBTCHeaderList":
		var req types.ReqRelayBtcHeaderHeightList
		err := types.Decode(params, &req)
		if err != nil {
			return nil, err
		}
		return r.btcstore.getHeadHeigtList(&req)

	default:
	}
	relaylog.Error("relay Query", "Query type not supprt with func name", funcName)
	return nil, types.ErrQueryNotSupport
}

func (r *relay) GetSellOrderByStatus(addrCoins *types.ReqRelayAddrCoins) (types.Message, error) {
	var prefixs [][]byte
	if 0 == len(addrCoins.Coins) {
		val := getRelayOrderPrefixStatus((int32)(addrCoins.Status))
		prefixs = append(prefixs, val)
	} else {
		for _, coin := range addrCoins.Coins {
			val := getRelayOrderPrefixCoinStatus(coin, (int32)(addrCoins.Status))
			prefixs = append(prefixs, val)
		}
	}

	return r.GetSellOrder(prefixs)

}

func (r *relay) GetSellRelayOrder(addrCoins *types.ReqRelayAddrCoins) (types.Message, error) {
	var prefixs [][]byte
	if 0 == len(addrCoins.Coins) {
		val := getRelayOrderPrefixAddr(addrCoins.Addr)
		prefixs = append(prefixs, val)
	} else {
		for _, coin := range addrCoins.Coins {
			val := getRelayOrderPrefixAddrCoin(addrCoins.Addr, coin)
			prefixs = append(prefixs, val)
		}
	}

	return r.GetSellOrder(prefixs)

}

func (r *relay) GetBuyRelayOrder(addrCoins *types.ReqRelayAddrCoins) (types.Message, error) {
	var prefixs [][]byte
	if 0 == len(addrCoins.Coins) {
		val := getBuyOrderPrefixAddr(addrCoins.Addr)
		prefixs = append(prefixs, val)
	} else {
		for _, coin := range addrCoins.Coins {
			val := getBuyOrderPrefixAddrCoin(addrCoins.Addr, coin)
			prefixs = append(prefixs, val)
		}
	}

	return r.GetSellOrder(prefixs)

}

func (r *relay) GetSellOrder(prefixs [][]byte) (types.Message, error) {
	var orderids [][]byte

	for _, prefix := range prefixs {
		values, err := r.GetLocalDB().List(prefix, nil, 0, 0)
		if err != nil {
			return nil, err
		}

		if 0 != len(values) {
			relaylog.Debug("relay coin status Query", "get number of orderid", len(values))
			orderids = append(orderids, values...)
		}
	}

	return r.getRelayOrderReply(orderids)

}

func (r *relay) getRelayOrderReply(orderids [][]byte) (types.Message, error) {
	orderidGot := make(map[string]bool)

	var reply types.ReplyRelayOrders
	for _, orderid := range orderids {
		//因为通过db list功能获取的sellid由于条件设置宽松会出现重复sellid的情况，在此进行过滤
		if !orderidGot[string(orderid)] {
			if order, err := r.getSellOrderFromDb(orderid); err == nil {
				relaylog.Debug("relay Query", "getSellOrderFromID", string(orderid))
				reply.Relayorders = insertOrderDescending(order, reply.Relayorders)
			}
			orderidGot[string(orderid)] = true
		}
	}
	return &reply, nil
}

func insertOrderDescending(toBeInserted *types.RelayOrder, orders []*types.RelayOrder) []*types.RelayOrder {
	if 0 == len(orders) {
		orders = append(orders, toBeInserted)
	} else {
		index := len(orders)
		for i, element := range orders {
			if toBeInserted.Sellamount >= element.Sellamount {
				index = i
				break
			}
		}

		if len(orders) == index {
			orders = append(orders, toBeInserted)
		} else {
			rear := append([]*types.RelayOrder{}, orders[index:]...)
			orders = append(orders[0:index], toBeInserted)
			orders = append(orders, rear...)
		}
	}
	return orders
}

func (r *relay) getSellOrderFromDb(orderid []byte) (*types.RelayOrder, error) {
	value, err := r.GetStateDB().Get(orderid)
	if err != nil {
		return nil, err
	}
	var order types.RelayOrder
	types.Decode(value, &order)
	return &order, nil
}

func (r *relay) getBTCHeaderFromDb(hash []byte) (*types.BtcHeader, error) {
	value, err := r.GetStateDB().Get(hash)
	if err != nil {
		return nil, err
	}
	var header types.BtcHeader
	types.Decode(value, &header)
	return &header, nil
}

func getSellOrderKv(order *types.RelayOrder) []*types.KeyValue {
	status := order.Status
	var kv []*types.KeyValue
	kv = getSellOrderKeyValue(kv, order, int32(status))
	if status == types.RelayOrderStatus_locking || status == types.RelayOrderStatus_canceled {
		kv = deleteSellOrderKeyValue(kv, order, int32(types.RelayOrderStatus_pending))
		//进入pending有两个状态，要么初始，要么locking，如果是初始，删除locking不影响
	} else if status == types.RelayOrderStatus_pending && BUY_CANCLED == order.Buyeraddr {
		kv = deleteSellOrderKeyValue(kv, order, int32(types.RelayOrderStatus_locking))
	} else if status == types.RelayOrderStatus_confirming {
		kv = deleteSellOrderKeyValue(kv, order, int32(types.RelayOrderStatus_locking))
	} else if status == types.RelayOrderStatus_finished {
		kv = deleteSellOrderKeyValue(kv, order, int32(types.RelayOrderStatus_confirming))
	}

	return kv
}

func getDeleteSellOrderKv(order *types.RelayOrder) []*types.KeyValue {
	status := order.Status
	var kv []*types.KeyValue
	kv = deleteSellOrderKeyValue(kv, order, int32(status))
	if status == types.RelayOrderStatus_locking || status == types.RelayOrderStatus_canceled {
		kv = getSellOrderKeyValue(kv, order, int32(types.RelayOrderStatus_pending))
	} else if status == types.RelayOrderStatus_pending && BUY_CANCLED == order.Buyeraddr {
		kv = getSellOrderKeyValue(kv, order, int32(types.RelayOrderStatus_locking))
	} else if status == types.RelayOrderStatus_confirming {
		kv = getSellOrderKeyValue(kv, order, int32(types.RelayOrderStatus_locking))
	} else if status == types.RelayOrderStatus_finished {
		kv = getSellOrderKeyValue(kv, order, int32(types.RelayOrderStatus_confirming))
	}

	return kv
}

func getBuyOrderKv(order *types.RelayOrder) []*types.KeyValue {
	status := order.Status
	var kv []*types.KeyValue
	kv = getBuyOrderKeyValue(kv, order, int32(status))
	if status == types.RelayOrderStatus_confirming || status == types.RelayOrderStatus_pending {
		kv = deleteBuyOrderKeyValue(kv, order, int32(types.RelayOrderStatus_locking))
	} else if status == types.RelayOrderStatus_finished {
		kv = deleteBuyOrderKeyValue(kv, order, int32(types.RelayOrderStatus_confirming))
	}

	return kv
}

func getDeleteBuyOrderKv(order *types.RelayOrder) []*types.KeyValue {
	status := order.Status
	var kv []*types.KeyValue
	kv = deleteBuyOrderKeyValue(kv, order, int32(status))
	if status == types.RelayOrderStatus_confirming || status == types.RelayOrderStatus_pending {
		kv = getBuyOrderKeyValue(kv, order, int32(types.RelayOrderStatus_locking))
	} else if status == types.RelayOrderStatus_finished {
		kv = getBuyOrderKeyValue(kv, order, int32(types.RelayOrderStatus_confirming))
	}

	return kv
}

func (r *relay) saveSell(orderid []byte, ty int32) []*types.KeyValue {
	order, _ := r.getSellOrderFromDb(orderid)
	return getSellOrderKv(order)
}

func (r *relay) saveBuy(orderid []byte, ty int32) []*types.KeyValue {
	order, _ := r.getSellOrderFromDb(orderid)
	return getBuyOrderKv(order)
}

func (r *relay) deletSell(orderid []byte, ty int32) []*types.KeyValue {
	order, _ := r.getSellOrderFromDb(orderid)
	return getDeleteSellOrderKv(order)
}

func (r *relay) deleteBuy(orderid []byte, ty int32) []*types.KeyValue {
	order, _ := r.getSellOrderFromDb(orderid)
	return getDeleteBuyOrderKv(order)
}

func getSellOrderKeyValue(kv []*types.KeyValue, order *types.RelayOrder, status int32) []*types.KeyValue {
	orderid := []byte(order.Orderid)

	key := getRelayOrderKeyStatus(order, status)
	kv = append(kv, &types.KeyValue{key, orderid})

	key = getRelayOrderKeyCoin(order, status)
	kv = append(kv, &types.KeyValue{key, orderid})

	key = getRelayOrderKeyAddrStatus(order, status)
	kv = append(kv, &types.KeyValue{key, orderid})

	key = getRelayOrderKeyAddrCoin(order, status)
	kv = append(kv, &types.KeyValue{key, orderid})

	return kv

}

func deleteSellOrderKeyValue(kv []*types.KeyValue, order *types.RelayOrder, status int32) []*types.KeyValue {

	key := getRelayOrderKeyStatus(order, status)
	kv = append(kv, &types.KeyValue{key, nil})

	key = getRelayOrderKeyCoin(order, status)
	kv = append(kv, &types.KeyValue{key, nil})

	key = getRelayOrderKeyAddrStatus(order, status)
	kv = append(kv, &types.KeyValue{key, nil})

	key = getRelayOrderKeyAddrCoin(order, status)
	kv = append(kv, &types.KeyValue{key, nil})

	return kv
}

func getBuyOrderKeyValue(kv []*types.KeyValue, order *types.RelayOrder, status int32) []*types.KeyValue {
	orderid := []byte(order.Orderid)

	key := getBuyOrderKeyAddr(order, status)
	kv = append(kv, &types.KeyValue{key, orderid})

	return kv
}

func deleteBuyOrderKeyValue(kv []*types.KeyValue, order *types.RelayOrder, status int32) []*types.KeyValue {

	key := getBuyOrderKeyAddr(order, status)
	kv = append(kv, &types.KeyValue{key, nil})

	return kv
}
