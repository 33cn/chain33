package relay

import (
	log "github.com/inconshreveable/log15"

	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/types"
)

var relaylog = log.New("module", "execs.relay")

func Init() {
	drivers.Register(newRelay().GetName(), newRelay, types.ForkV18Relay) //TODO: ForkV18Relay
}

type relay struct {
	drivers.DriverBase
}

func newRelay() drivers.Driver {
	r := &relay{}
	r.SetChild(r)

	return r
}

func (r *relay) GetName() string {
	return types.ExecName("relay")
}

func (r *relay) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	var action types.RelayAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return nil, err
	}

	actiondb := newRelayDB(r, tx)
	switch action.GetTy() {
	case types.RelayActionCreate:
		return actiondb.relayCreate(action.GetCreate())

	case types.RelayActionAccept:
		return actiondb.accept(action.GetAccept())

	case types.RelayActionRevoke:
		return actiondb.relayRevoke(action.GetRevoke())

	case types.RelayActionConfirmTx:
		return actiondb.confirmTx(action.GetConfirmTx())

	case types.RelayActionVerifyTx:
		return actiondb.verifyTx(action.GetVerify())

	// OrderId, rawTx, index sibling, blockhash
	case types.RelayActionVerifyCmdTx:
		return actiondb.verifyCmdTx(action.GetVerifyCli())

	case types.RelayActionRcvBTCHeaders:
		return actiondb.saveBtcHeader(action.GetBtcHeaders())

	default:
		return nil, types.ErrActionNotSupport
	}
}

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
		switch item.Ty {
		case types.TyLogRelayCreate,
			types.TyLogRelayRevokeCreate,
			types.TyLogRelayAccept,
			types.TyLogRelayRevokeAccept,
			types.TyLogRelayConfirmTx,
			types.TyLogRelayFinishTx:
			var receipt types.ReceiptRelayLog
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				return nil, err
			}
			kv := r.getOrderKv([]byte(receipt.OrderId), item.Ty)
			set.KV = append(set.KV, kv...)
		case types.TyLogRelayRcvBTCHead:
			var receipt = &types.ReceiptRelayRcvBTCHeaders{}
			err := types.Decode(item.Log, receipt)
			if err != nil {
				return nil, err
			}

			btc := newBtcStore(r.GetLocalDB())
			for _, head := range receipt.Headers {
				kv, err := btc.saveBlockHead(head)
				if err != nil {
					return nil, err
				}
				set.KV = append(set.KV, kv...)
			}

			kv, err := btc.saveBlockLastHead(receipt)
			if err != nil {
				return nil, err
			}
			set.KV = append(set.KV, kv...)

		default:

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
		switch item.Ty {
		case types.TyLogRelayCreate,
			types.TyLogRelayRevokeCreate,
			types.TyLogRelayAccept,
			types.TyLogRelayRevokeAccept,
			types.TyLogRelayConfirmTx,
			types.TyLogRelayFinishTx:
			var receipt types.ReceiptRelayLog
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				return nil, err
			}
			kv := r.getDeleteOrderKv([]byte(receipt.OrderId), item.Ty)
			set.KV = append(set.KV, kv...)
		case types.TyLogRelayRcvBTCHead:
			var receipt = &types.ReceiptRelayRcvBTCHeaders{}
			err := types.Decode(item.Log, receipt)
			if err != nil {
				return nil, err
			}

			btc := newBtcStore(r.GetLocalDB())
			for _, head := range receipt.Headers {
				kv, err := btc.delBlockHead(head)
				if err != nil {
					return nil, err
				}
				set.KV = append(set.KV, kv...)
			}

			kv, err := btc.delBlockLastHead(receipt)
			if err != nil {
				return nil, err
			}
			set.KV = append(set.KV, kv...)
		default:

		}
	}

	return set, nil

}

func (r *relay) Query(funcName string, params []byte) (types.Message, error) {
	switch funcName {
	case "GetRelayOrderByStatus":
		var addrCoins types.ReqRelayAddrCoins

		err := types.Decode(params, &addrCoins)
		if err != nil {
			return nil, err
		}
		return r.GetSellOrderByStatus(&addrCoins)
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
		db := newBtcStore(r.GetLocalDB())
		return db.getHeadHeightList(&req)

	case "GetBTCHeaderCurHeight":
		var req types.ReqRelayQryBTCHeadHeight
		err := types.Decode(params, &req)
		if err != nil {
			return nil, err
		}
		db := newBtcStore(r.GetLocalDB())
		return db.getBtcCurHeight(&req)
	default:
	}
	relaylog.Error("relay Query", "Query type not supprt with func name", funcName)
	return nil, types.ErrQueryNotSupport
}

func (r *relay) GetSellOrderByStatus(addrCoins *types.ReqRelayAddrCoins) (types.Message, error) {
	var prefixs [][]byte
	if 0 == len(addrCoins.Coins) {
		val := calcOrderPrefixStatus((int32)(addrCoins.Status))
		prefixs = append(prefixs, val)
	} else {
		for _, coin := range addrCoins.Coins {
			val := calcOrderPrefixCoinStatus(coin, (int32)(addrCoins.Status))
			prefixs = append(prefixs, val)
		}
	}

	return r.GetSellOrder(prefixs)

}

func (r *relay) GetSellRelayOrder(addrCoins *types.ReqRelayAddrCoins) (types.Message, error) {
	var prefixs [][]byte
	if 0 == len(addrCoins.Coins) {
		val := calcOrderPrefixAddr(addrCoins.Addr)
		prefixs = append(prefixs, val)
	} else {
		for _, coin := range addrCoins.Coins {
			val := calcOrderPrefixAddrCoin(addrCoins.Addr, coin)
			prefixs = append(prefixs, val)
		}
	}

	return r.GetSellOrder(prefixs)

}

func (r *relay) GetBuyRelayOrder(addrCoins *types.ReqRelayAddrCoins) (types.Message, error) {
	var prefixs [][]byte
	if 0 == len(addrCoins.Coins) {
		val := calcAcceptPrefixAddr(addrCoins.Addr)
		prefixs = append(prefixs, val)
	} else {
		for _, coin := range addrCoins.Coins {
			val := calcAcceptPrefixAddrCoin(addrCoins.Addr, coin)
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
			OrderIds = append(OrderIds, values...)
		}
	}

	return r.getRelayOrderReply(OrderIds)

}

func (r *relay) getRelayOrderReply(OrderIds [][]byte) (types.Message, error) {
	OrderIdGot := make(map[string]bool)

	var reply types.ReplyRelayOrders
	for _, OrderId := range OrderIds {
		if !OrderIdGot[string(OrderId)] {
			if order, err := r.getSellOrderFromDb(OrderId); err == nil {
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

func (r *relay) getOrderKv(OrderId []byte, ty int32) []*types.KeyValue {
	order, _ := r.getSellOrderFromDb(OrderId)

	var kv []*types.KeyValue
	kv = deleteCreateOrderKeyValue(kv, order, int32(order.PreStatus))
	kv = getCreateOrderKeyValue(kv, order, int32(order.Status))

	return kv
}

func (r *relay) getDeleteOrderKv(OrderId []byte, ty int32) []*types.KeyValue {
	order, _ := r.getSellOrderFromDb(OrderId)
	var kv []*types.KeyValue
	kv = deleteCreateOrderKeyValue(kv, order, int32(order.Status))
	kv = getCreateOrderKeyValue(kv, order, int32(order.PreStatus))

	return kv
}

func getCreateOrderKeyValue(kv []*types.KeyValue, order *types.RelayOrder, status int32) []*types.KeyValue {
	OrderId := []byte(order.Id)

	key := calcOrderKeyStatus(order, status)
	kv = append(kv, &types.KeyValue{key, OrderId})

	key = calcOrderKeyCoin(order, status)
	kv = append(kv, &types.KeyValue{key, OrderId})

	key = calcOrderKeyAddrStatus(order, status)
	kv = append(kv, &types.KeyValue{key, OrderId})

	key = calcOrderKeyAddrCoin(order, status)
	kv = append(kv, &types.KeyValue{key, OrderId})

	key = calcAcceptKeyAddr(order, status)
	if key != nil {
		kv = append(kv, &types.KeyValue{key, OrderId})
	}

	return kv

}

func deleteCreateOrderKeyValue(kv []*types.KeyValue, order *types.RelayOrder, status int32) []*types.KeyValue {

	key := calcOrderKeyStatus(order, status)
	kv = append(kv, &types.KeyValue{key, nil})

	key = calcOrderKeyCoin(order, status)
	kv = append(kv, &types.KeyValue{key, nil})

	key = calcOrderKeyAddrStatus(order, status)
	kv = append(kv, &types.KeyValue{key, nil})

	key = calcOrderKeyAddrCoin(order, status)
	kv = append(kv, &types.KeyValue{key, nil})

	key = calcAcceptKeyAddr(order, status)
	if key != nil {
		kv = append(kv, &types.KeyValue{key, nil})
	}

	return kv
}
