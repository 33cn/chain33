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

	actiondb := newRelayDB(r, tx)
	switch action.GetTy() {
	case types.RelayActionCreate:
		return actiondb.relayCreate(action.GetCreate())

	case types.RelayActionAccept:
		return actiondb.accept(action.GetAccept())

	case types.RelayActionRevoke:
		return actiondb.relayRevoke(action.GetRevoke())

	//OrderId, txHash
	case types.RelayActionConfirmTx:
		return actiondb.confirmTx(action.GetConfirmTx())

	// OrderId, rawTx, index sibling, blockhash
	case types.RelayActionVerifyTx:
		return actiondb.verifyTx(action.GetVerify(), r)

	// OrderId, rawTx, index sibling, blockhash
	case types.RelayActionVerifyBTCTx:
		return actiondb.verifyBtcTx(action.GetVerifyCli(), r)

	case types.RelayActionRcvBTCHeaders:
		return actiondb.saveBtcHeader(action.GetBtcHeaders())

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
		if item.Ty == types.TyLogRelayCreate ||
			item.Ty == types.TyLogRelayRevokeCreate ||
			item.Ty == types.TyLogRelayAccept ||
			item.Ty == types.TyLogRelayRevokeAccept ||
			item.Ty == types.TyLogRelayConfirmTx ||
			item.Ty == types.TyLogRelayFinishTx {
			var receipt types.ReceiptRelayLog
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err)
			}
			kv := r.getOrderKv([]byte(receipt.Base.Id), item.Ty)
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
		if item.Ty == types.TyLogRelayCreate ||
			item.Ty == types.TyLogRelayRevokeCreate ||
			item.Ty == types.TyLogRelayAccept ||
			item.Ty == types.TyLogRelayRevokeAccept ||
			item.Ty == types.TyLogRelayConfirmTx ||
			item.Ty == types.TyLogRelayFinishTx {
			var receipt types.ReceiptRelayLog
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err)
			}
			kv := r.getDeleteOrderKv([]byte(receipt.Base.Id), item.Ty)
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
		val := getAcceptPrefixAddr(addrCoins.Addr)
		prefixs = append(prefixs, val)
	} else {
		for _, coin := range addrCoins.Coins {
			val := getAcceptPrefixAddrCoin(addrCoins.Addr, coin)
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

func (r *relay) getOrderKv(OrderId []byte, ty int32) []*types.KeyValue {
	order, _ := r.getSellOrderFromDb(OrderId)

	var kv []*types.KeyValue
	kv = getCreateOrderKeyValue(kv, order, int32(order.Status))
	kv = deleteCreateOrderKeyValue(kv, order, int32(order.PreStatus))

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

	key := getOrderKeyStatus(order, status)
	kv = append(kv, &types.KeyValue{key, OrderId})

	key = getOrderKeyCoin(order, status)
	kv = append(kv, &types.KeyValue{key, OrderId})

	key = getOrderKeyAddrStatus(order, status)
	kv = append(kv, &types.KeyValue{key, OrderId})

	key = getOrderKeyAddrCoin(order, status)
	kv = append(kv, &types.KeyValue{key, OrderId})

	key = getAcceptKeyAddr(order, status)
	if key != nil {
		kv = append(kv, &types.KeyValue{key, OrderId})
	}

	return kv

}

func deleteCreateOrderKeyValue(kv []*types.KeyValue, order *types.RelayOrder, status int32) []*types.KeyValue {

	key := getOrderKeyStatus(order, status)
	kv = append(kv, &types.KeyValue{key, nil})

	key = getOrderKeyCoin(order, status)
	kv = append(kv, &types.KeyValue{key, nil})

	key = getOrderKeyAddrStatus(order, status)
	kv = append(kv, &types.KeyValue{key, nil})

	key = getOrderKeyAddrCoin(order, status)
	kv = append(kv, &types.KeyValue{key, nil})

	key = getAcceptKeyAddr(order, status)
	if key != nil {
		kv = append(kv, &types.KeyValue{key, nil})
	}

	return kv
}
