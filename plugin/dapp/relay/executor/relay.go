package executor

import (
	log "github.com/inconshreveable/log15"

	rTy "gitlab.33.cn/chain33/chain33/plugin/dapp/relay/types"
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
)

var relaylog = log.New("module", "execs.relay")

func Init(name string) {
	drivers.Register(GetName(), newRelay, types.ForkV18Relay) //TODO: ForkV18Relay
}

func GetName() string {
	return newRelay().GetName()
}

type relay struct {
	drivers.DriverBase
}

func newRelay() drivers.Driver {
	r := &relay{}
	r.SetChild(r)

	return r
}

func (r *relay) GetDriverName() string {
	return "relay"
}

func (r *relay) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	var action rTy.RelayAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return nil, err
	}

	actiondb := newRelayDB(r, tx)
	switch action.GetTy() {
	case rTy.RelayActionCreate:
		if action.GetCreate() == nil {
			return nil, types.ErrInputPara
		}
		return actiondb.relayCreate(action.GetCreate())

	case rTy.RelayActionAccept:
		if action.GetAccept() == nil {
			return nil, types.ErrInputPara
		}
		return actiondb.accept(action.GetAccept())

	case rTy.RelayActionRevoke:
		if action.GetRevoke() == nil {
			return nil, types.ErrInputPara
		}
		return actiondb.relayRevoke(action.GetRevoke())

	case rTy.RelayActionConfirmTx:
		if action.GetConfirmTx() == nil {
			return nil, types.ErrInputPara
		}
		return actiondb.confirmTx(action.GetConfirmTx())

	case rTy.RelayActionVerifyTx:
		if action.GetVerify() == nil {
			return nil, types.ErrInputPara
		}
		return actiondb.verifyTx(action.GetVerify())

	// OrderId, rawTx, index sibling, blockhash
	case rTy.RelayActionVerifyCmdTx:
		if action.GetVerifyCli() == nil {
			return nil, types.ErrInputPara
		}
		return actiondb.verifyCmdTx(action.GetVerifyCli())

	case rTy.RelayActionRcvBTCHeaders:
		if action.GetBtcHeaders() == nil {
			return nil, types.ErrInputPara
		}
		return actiondb.saveBtcHeader(action.GetBtcHeaders(), r.GetLocalDB())

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
		case rTy.TyLogRelayCreate,
			rTy.TyLogRelayRevokeCreate,
			rTy.TyLogRelayAccept,
			rTy.TyLogRelayRevokeAccept,
			rTy.TyLogRelayConfirmTx,
			rTy.TyLogRelayFinishTx:
			var receipt rTy.ReceiptRelayLog
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				return nil, err
			}
			kv := r.getOrderKv([]byte(receipt.OrderId), item.Ty)
			set.KV = append(set.KV, kv...)
		case rTy.TyLogRelayRcvBTCHead:
			var receipt = &rTy.ReceiptRelayRcvBTCHeaders{}
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
		case rTy.TyLogRelayCreate,
			rTy.TyLogRelayRevokeCreate,
			rTy.TyLogRelayAccept,
			rTy.TyLogRelayRevokeAccept,
			rTy.TyLogRelayConfirmTx,
			rTy.TyLogRelayFinishTx:
			var receipt rTy.ReceiptRelayLog
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				return nil, err
			}
			kv := r.getDeleteOrderKv([]byte(receipt.OrderId), item.Ty)
			set.KV = append(set.KV, kv...)
		case rTy.TyLogRelayRcvBTCHead:
			var receipt = &rTy.ReceiptRelayRcvBTCHeaders{}
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
		var addrCoins rTy.ReqRelayAddrCoins

		err := types.Decode(params, &addrCoins)
		if err != nil {
			return nil, err
		}
		return r.GetSellOrderByStatus(&addrCoins)
	case "GetSellRelayOrder":
		var addrCoins rTy.ReqRelayAddrCoins
		err := types.Decode(params, &addrCoins)
		if err != nil {
			return nil, err
		}
		return r.GetSellRelayOrder(&addrCoins)
	case "GetBuyRelayOrder":
		var addrCoins rTy.ReqRelayAddrCoins
		err := types.Decode(params, &addrCoins)
		if err != nil {
			return nil, err
		}
		return r.GetBuyRelayOrder(&addrCoins)

	case "GetBTCHeaderList":
		var req rTy.ReqRelayBtcHeaderHeightList
		err := types.Decode(params, &req)
		if err != nil {
			return nil, err
		}
		db := newBtcStore(r.GetLocalDB())
		return db.getHeadHeightList(&req)

	case "GetBTCHeaderCurHeight":
		var req rTy.ReqRelayQryBTCHeadHeight
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

func (r *relay) GetSellOrderByStatus(addrCoins *rTy.ReqRelayAddrCoins) (types.Message, error) {
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

func (r *relay) GetSellRelayOrder(addrCoins *rTy.ReqRelayAddrCoins) (types.Message, error) {
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

func (r *relay) GetBuyRelayOrder(addrCoins *rTy.ReqRelayAddrCoins) (types.Message, error) {
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

	var reply rTy.ReplyRelayOrders
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

func insertOrderDescending(toBeInserted *rTy.RelayOrder, orders []*rTy.RelayOrder) []*rTy.RelayOrder {
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
			rear := append([]*rTy.RelayOrder{}, orders[index:]...)
			orders = append(orders[0:index], toBeInserted)
			orders = append(orders, rear...)
		}
	}
	return orders
}

func (r *relay) getSellOrderFromDb(OrderId []byte) (*rTy.RelayOrder, error) {
	value, err := r.GetStateDB().Get(OrderId)
	if err != nil {
		return nil, err
	}
	var order rTy.RelayOrder
	types.Decode(value, &order)
	return &order, nil
}

func (r *relay) getBTCHeaderFromDb(hash []byte) (*rTy.BtcHeader, error) {
	value, err := r.GetStateDB().Get(hash)
	if err != nil {
		return nil, err
	}
	var header rTy.BtcHeader
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

func getCreateOrderKeyValue(kv []*types.KeyValue, order *rTy.RelayOrder, status int32) []*types.KeyValue {
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

func deleteCreateOrderKeyValue(kv []*types.KeyValue, order *rTy.RelayOrder, status int32) []*types.KeyValue {

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
