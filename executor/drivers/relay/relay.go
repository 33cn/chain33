package relay

import (
	log "github.com/inconshreveable/log15"

	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/types"
)

var relaylog = log.New("module", "execs.relay")

func Init() {
	r := newRelay()
	drivers.Register(r.GetName(), r, types.ForkV7AddRelay) //TODO: ForkV7AddRelay
}

type relay struct {
	drivers.DriverBase
	btcStore relayBTCStore
}

func newRelay() *relay {
	r := &relay{}
	r.SetChild(r)
	r.btcStore.new(r)

	return r
}

func (r *relay) GetName() string {
	return "relay"
}

func (r *relay) Clone() drivers.Driver {
	clone := &relay{}
	clone.DriverBase = *(r.DriverBase.Clone().(*drivers.DriverBase))
	clone.SetChild(clone)
	clone.btcStore.r = clone

	return clone
}

func (r *relay) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	var action types.RelayAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return nil, err
	}
	relaylog.Debug("exec relay tx", "tx hash", common.Bytes2Hex(tx.Hash()), "Ty", action.GetTy())

	actiondb := newRelayAction(r, tx)
	switch action.GetTy() {
	case types.RelayActionCreate:
		return actiondb.relayCreate(action.GetCreate())

	case types.RelayActionRevokeCreate:
		return actiondb.relayRevokeCreate(action.GetRevokeCreate())

	case types.RelayActionAccept:
		return actiondb.relayAccept(action.GetAccept())

	case types.RelayActionRevokeAccept:
		return actiondb.relayRevokeAccept(action.GetRevokeAccept())

	//OrderId, txHash
	case types.RelayActionConfirmTx:
		return actiondb.relayConfirmTx(action.GetConfirmTx())

	// OrderId, rawTx, index sibling, blockhash
	case types.RelayActionVerifyTx:
		return actiondb.relayVerifyTx(action.GetVerify(), r)

	// OrderId, rawTx, index sibling, blockhash
	case types.RelayActionVerifyBTCTx:
		return actiondb.relayVerifyBTCTx(action.GetVerifyBtc(), r)

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
		if item.Ty == types.TyLogRelayCreate || item.Ty == types.TyLogRelayRevokeCreate {
			var receipt types.ReceiptRelayCreate
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err)
			}
			kv := r.saveSell([]byte(receipt.Base.OrderId), item.Ty)
			set.KV = append(set.KV, kv...)
		} else if item.Ty == types.TyLogRelayAccept || item.Ty == types.TyLogRelayRevokeAccept {
			var receipt types.ReceiptRelayAccept
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err)
			}

			kv := r.saveBuy([]byte(receipt.OrderId), item.Ty)
			set.KV = append(set.KV, kv...)
		} else if item.Ty == types.TyLogRelayRcvBTCHead {

			var receipt types.ReceiptRelayRcvBTCHeaders
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err)
			}

			kv := r.btcStore.saveBlockHead(receipt.Base)
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
		if item.Ty == types.TyLogRelayCreate || item.Ty == types.TyLogRelayRevokeCreate {
			var receipt types.ReceiptRelayCreate
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err)
			}
			kv := r.deletSell([]byte(receipt.Base.OrderId), item.Ty)
			set.KV = append(set.KV, kv...)
		} else if item.Ty == types.TyLogRelayAccept || item.Ty == types.TyLogRelayRevokeAccept {
			var receipt types.ReceiptRelayAccept
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err)
			}
			kv := r.deleteBuy([]byte(receipt.OrderId), item.Ty)
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
		return r.btcStore.getHeadHeightList(&req)

	case "GetBTCHeaderCurHeight":
		var req types.ReqRelayQryBTCHeadHeight
		err := types.Decode(params, &req)
		if err != nil {
			return nil, err
		}
		return r.btcStore.getBTCHeadDbCurHeight(&req)
	default:
	}
	relaylog.Error("relay Query", "Query type not supprt with func name", funcName)
	return nil, types.ErrQueryNotSupport
}

func (r *relay) GetSellOrderByStatus(addrCoins *types.ReqRelayAddrCoins) (types.Message, error) {
	var prefixs [][]byte
	if 0 == len(addrCoins.Coins) {
		val := getOrderPrefixStatus((int32)(addrCoins.Status))
		prefixs = append(prefixs, val)
	} else {
		for _, coin := range addrCoins.Coins {
			val := getOrderPrefixCoinStatus(coin, (int32)(addrCoins.Status))
			prefixs = append(prefixs, val)
		}
	}

	return r.GetSellOrder(prefixs)

}

func (r *relay) GetSellRelayOrder(addrCoins *types.ReqRelayAddrCoins) (types.Message, error) {
	var prefixs [][]byte
	if 0 == len(addrCoins.Coins) {
		val := getOrderPrefixAddr(addrCoins.Addr)
		prefixs = append(prefixs, val)
	} else {
		for _, coin := range addrCoins.Coins {
			val := getOrderPrefixAddrCoin(addrCoins.Addr, coin)
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
	var OrderIds [][]byte

	for _, prefix := range prefixs {
		values, err := r.GetLocalDB().List(prefix, nil, 0, 0)
		if err != nil {
			return nil, err
		}

		if 0 != len(values) {
			relaylog.Debug("relay coin status Query", "get number of OrderId", len(values))
			OrderIds = append(OrderIds, values...)
		}
	}

	return r.getRelayOrderReply(OrderIds)

}

func (r *relay) getRelayOrderReply(OrderIds [][]byte) (types.Message, error) {
	OrderIdGot := make(map[string]bool)

	var reply types.ReplyRelayOrders
	for _, OrderId := range OrderIds {
		//因为通过db list功能获取的sellid由于条件设置宽松会出现重复sellid的情况，在此进行过滤
		if !OrderIdGot[string(OrderId)] {
			if order, err := r.getSellOrderFromDb(OrderId); err == nil {
				relaylog.Debug("relay Query", "getSellOrderFromID", string(OrderId))
				reply.Relayorders = insertOrderDescending(order, reply.Relayorders)
			}
			OrderIdGot[string(OrderId)] = true
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
			if toBeInserted.Amount >= element.Amount {
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

func (r *relay) getSellOrderFromDb(OrderId []byte) (*types.RelayOrder, error) {
	value, err := r.GetStateDB().Get(OrderId)
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

func getSellOrderKv(order *types.RelayOrder, ty int32) []*types.KeyValue {
	status := order.Status
	var kv []*types.KeyValue
	kv = getSellOrderKeyValue(kv, order, int32(status))
	if status == types.RelayOrderStatus_locking {
		kv = deleteSellOrderKeyValue(kv, order, int32(types.RelayOrderStatus_pending))
	} else if status == types.RelayOrderStatus_canceled {
		if order.ConfirmTime != 0 {
			kv = deleteSellOrderKeyValue(kv, order, int32(types.RelayOrderStatus_confirming))
		} else if order.AcceptTime != 0 {
			kv = deleteSellOrderKeyValue(kv, order, int32(types.RelayOrderStatus_locking))
		} else {
			kv = deleteSellOrderKeyValue(kv, order, int32(types.RelayOrderStatus_pending))
		}

	} else if status == types.RelayOrderStatus_pending {
		if ty == types.TyLogRelayRevokeCreate {
			if order.ConfirmTime != 0 {
				kv = deleteSellOrderKeyValue(kv, order, int32(types.RelayOrderStatus_confirming))
			} else if order.AcceptTime != 0 {
				kv = deleteSellOrderKeyValue(kv, order, int32(types.RelayOrderStatus_locking))
			}
		} else if ty == types.TyLogRelayRevokeAccept {
			if order.ConfirmTime != 0 {
				kv = deleteSellOrderKeyValue(kv, order, int32(types.RelayOrderStatus_confirming))
			} else {
				kv = deleteSellOrderKeyValue(kv, order, int32(types.RelayOrderStatus_locking))
			}
		}

	} else if status == types.RelayOrderStatus_confirming {
		kv = deleteSellOrderKeyValue(kv, order, int32(types.RelayOrderStatus_locking))
	} else if status == types.RelayOrderStatus_finished {
		kv = deleteSellOrderKeyValue(kv, order, int32(types.RelayOrderStatus_confirming))
	}

	return kv
}

func getDeleteSellOrderKv(order *types.RelayOrder, ty int32) []*types.KeyValue {
	status := order.Status
	var kv []*types.KeyValue
	kv = deleteSellOrderKeyValue(kv, order, int32(status))
	if status == types.RelayOrderStatus_locking {
		kv = getSellOrderKeyValue(kv, order, int32(types.RelayOrderStatus_pending))
	} else if status == types.RelayOrderStatus_canceled {
		if order.ConfirmTime != 0 {
			kv = getSellOrderKeyValue(kv, order, int32(types.RelayOrderStatus_confirming))
		} else if order.AcceptTime != 0 {
			kv = getSellOrderKeyValue(kv, order, int32(types.RelayOrderStatus_locking))
		} else {
			kv = getSellOrderKeyValue(kv, order, int32(types.RelayOrderStatus_pending))
		}
	} else if ty == types.TyLogRelayRevokeCreate && status == types.RelayOrderStatus_pending {
		if order.ConfirmTime != 0 {
			kv = getSellOrderKeyValue(kv, order, int32(types.RelayOrderStatus_confirming))
		} else if order.AcceptTime != 0 {
			kv = getSellOrderKeyValue(kv, order, int32(types.RelayOrderStatus_locking))
		}
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

func (r *relay) saveSell(OrderId []byte, ty int32) []*types.KeyValue {
	order, _ := r.getSellOrderFromDb(OrderId)
	return getSellOrderKv(order, ty)
}

func (r *relay) saveBuy(OrderId []byte, ty int32) []*types.KeyValue {
	order, _ := r.getSellOrderFromDb(OrderId)
	return getBuyOrderKv(order)
}

func (r *relay) deletSell(OrderId []byte, ty int32) []*types.KeyValue {
	order, _ := r.getSellOrderFromDb(OrderId)
	return getDeleteSellOrderKv(order, ty)
}

func (r *relay) deleteBuy(OrderId []byte, ty int32) []*types.KeyValue {
	order, _ := r.getSellOrderFromDb(OrderId)
	return getDeleteBuyOrderKv(order)
}

func getSellOrderKeyValue(kv []*types.KeyValue, order *types.RelayOrder, status int32) []*types.KeyValue {
	OrderId := []byte(order.Id)

	key := getOrderKeyStatus(order, status)
	kv = append(kv, &types.KeyValue{key, OrderId})

	key = getOrderKeyCoin(order, status)
	kv = append(kv, &types.KeyValue{key, OrderId})

	key = getOrderKeyAddrStatus(order, status)
	kv = append(kv, &types.KeyValue{key, OrderId})

	key = getOrderKeyAddrCoin(order, status)
	kv = append(kv, &types.KeyValue{key, OrderId})

	return kv

}

func deleteSellOrderKeyValue(kv []*types.KeyValue, order *types.RelayOrder, status int32) []*types.KeyValue {

	key := getOrderKeyStatus(order, status)
	kv = append(kv, &types.KeyValue{key, nil})

	key = getOrderKeyCoin(order, status)
	kv = append(kv, &types.KeyValue{key, nil})

	key = getOrderKeyAddrStatus(order, status)
	kv = append(kv, &types.KeyValue{key, nil})

	key = getOrderKeyAddrCoin(order, status)
	kv = append(kv, &types.KeyValue{key, nil})

	return kv
}

func getBuyOrderKeyValue(kv []*types.KeyValue, order *types.RelayOrder, status int32) []*types.KeyValue {
	OrderId := []byte(order.Id)

	key := getBuyOrderKeyAddr(order, status)
	kv = append(kv, &types.KeyValue{key, OrderId})

	return kv
}

func deleteBuyOrderKeyValue(kv []*types.KeyValue, order *types.RelayOrder, status int32) []*types.KeyValue {

	key := getBuyOrderKeyAddr(order, status)
	kv = append(kv, &types.KeyValue{key, nil})

	return kv
}
