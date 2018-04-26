package relay

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

import (
	log "github.com/inconshreveable/log15"

	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/types"
)

var relaylog = log.New("module", "execs.relay")

func init() {
	r := newRelay()
	drivers.Register(r.GetName(), r, types.ForkV7AddRelay) //TODO: ForkV7AddRelay
}

type relay struct {
	drivers.DriverBase
}

func newRelay() *relay {
	r := &relay{}
	r.SetChild(r)
	return r
}

func (r *relay) GetName() string {
	return "relay"
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

	// orderid, txhash
	//case types.RelayActionVerifyTx:
	//	return actiondb.relayVerifyBTCTx(action.GetRverifybtc())

	// orderid, rawTx, index sibling, blockhash
	case types.RelayActionVerifyBTCTx:
		return actiondb.relayVerifyBTCTx(action.GetRverifybtc())

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
		if item.Ty == types.TyLogRelaySell || item.Ty == types.TyLogRelayRevoke {
			var receipt types.ReceiptRelaySell
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err)
			}
			kv := r.saveSell([]byte(receipt.Base.Orderid), item.Ty)
			set.KV = append(set.KV, kv...)
		} else if item.Ty == types.TyLogRelayBuy {
			var receipt types.ReceiptRelayBuy
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err)
			}

			kv := r.saveBuy([]byte(receipt.Orderid), item.Ty)
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
		if item.Ty == types.TyLogRelaySell || item.Ty == types.TyLogRelayRevoke {
			var receipt types.ReceiptRelaySell
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err)
			}
			kv := r.deletSell([]byte(receipt.Base.Orderid), item.Ty)
			set.KV = append(set.KV, kv...)
		} else if item.Ty == types.TyLogRelayBuy {
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
	//查询某个特定用户的一个或者多个coin兑换的卖单,包括所有状态的卖单
	//TODO:后续可以考虑支持查询不同状态的卖单
	case "GetRelayOrder":
		var addrCoins types.ReqRelayAddrCoins
		err := types.Decode(params, &addrCoins)
		if err != nil {
			return nil, err
		}
		return r.GetRelayOrder(&addrCoins)
	case "GetRelayBuyOrder":
		var addrCoins types.ReqRelayAddrCoins
		err := types.Decode(params, &addrCoins)
		if err != nil {
			return nil, err
		}
		return r.GetRelayBuyOrder(&addrCoins)
		//查寻所有的可以进行交易的卖单
	case "GetRelayOrderByCoinStatus":
		var addrCoins types.ReqRelayAddrCoins
		err := types.Decode(params, &addrCoins)
		if err != nil {
			return nil, err
		}
		return r.GetOrderByCoinStatus(&addrCoins)
	default:
	}
	relaylog.Error("relay Query", "Query type not supprt with func name", funcName)
	return nil, types.ErrQueryNotSupport
}

func (r *relay) GetRelayOrder(addrCoins *types.ReqRelayAddrCoins) (types.Message, error) {
	orderidGot := make(map[string]bool)
	var orderids [][]byte
	if 0 == len(addrCoins.Coins) {
		values, err := r.GetLocalDB().List(getRelayOrderPrefixAddr(addrCoins.Addr), nil, 0, 0)
		if err != nil {
			return nil, err
		}
		if len(values) != 0 {
			relaylog.Debug("relay Query", "get number of orderid", len(values))
			orderids = append(orderids, values...)
		}
	} else {
		for _, coin := range addrCoins.Coins {
			values, err := r.GetLocalDB().List(getRelayOrderPrefixAddrToken(coin, addrCoins.Addr), nil, 0, 0)
			relaylog.Debug("relay Query", "Begin to list addr with coin", coin, "got values", len(values))
			if err != nil {
				return nil, err
			}
			if len(values) != 0 {
				orderids = append(orderids, values...)
			}
		}
	}

	var reply types.ReplyRelayOrders
	for _, orderid := range orderids {
		//因为通过db list功能获取的sellid由于条件设置宽松会出现重复sellid的情况，在此进行过滤
		if !orderidGot[string(orderid)] {
			if order, err := r.getSellOrderFromDb(orderid); err == nil {
				relaylog.Debug("trade Query", "getSellOrderFromID", string(orderid))
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

func (r *relay) GetRelayBuyOrder(addrCoins *types.ReqRelayAddrCoins) (types.Message, error) {
	var orderIdGot = make(map[string]bool)
	var orderIds [][]byte

	values, err := r.GetLocalDB().List(getBuyOrderPrefixAddr(addrCoins.Addr), nil, 0, 0)
	if err != nil {
		return nil, err
	}
	if len(values) != 0 {
		relaylog.Debug("relay buy order Query", "get number of order", len(values))
		orderIds = append(orderIds, values...)
	}

	var reply types.ReplyRelayOrders
	for _, orderid := range orderIds {
		order, err := r.getSellOrderFromDb(orderid)
		if err != nil {
			return nil, err
		}

		if !orderIdGot[order.Orderid] {
			if len(addrCoins.Coins) != 0 {
				for _, coin := range addrCoins.Coins {
					if order.Exchgcoin == coin {
						reply.Relayorders = append(reply.Relayorders, order)
					}
				}

			} else {
				reply.Relayorders = append(reply.Relayorders, order)
			}

			orderIdGot[order.Orderid] = true
		}
	}

	return &reply, nil

}

func (r *relay) GetOrderByCoinStatus(addrCoins *types.ReqRelayAddrCoins) (types.Message, error) {
	orderIdGot := make(map[string]bool)
	var orderids [][]byte
	if 0 == len(addrCoins.Coins) {
		values, err := r.GetLocalDB().List(getRelayOrderPrefixTokenStatus("BTC", addrCoins.Status), nil, 0, 0)
		if err != nil {
			return nil, err
		}

		if 0 != len(values) {
			relaylog.Debug("relay coin status Query", "get number of orderid", len(values))
			orderids = append(orderids, values...)
		}
	} else {
		for _, coin := range addrCoins.Coins {
			values, err := r.GetLocalDB().List(getRelayOrderPrefixTokenStatus(coin, addrCoins.Status), nil, 0, 0)
			if err != nil {
				return nil, err
			}

			if 0 != len(values) {
				relaylog.Debug("relay coin status Query", "get number of orderid", len(values))
				orderids = append(orderids, values...)
			}
		}
	}

	var reply types.ReplyRelayOrders
	for _, orderid := range orderids {
		if !orderIdGot[string(orderid)] {
			if order, err := r.getSellOrderFromDb(orderid); err == nil {
				relaylog.Debug("trade coin status Query", "getSellOrderFromID", string(orderid))
				reply.Relayorders = append(reply.Relayorders, order)
			}
			orderIdGot[string(orderid)] = true
		}
	}

	return &reply, nil
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

func getSellOrderKv(order *types.RelayOrder) []*types.KeyValue {
	status := order.Status
	var kv []*types.KeyValue
	kv = getSellOrderKeyValue(kv, order, status)
	if status == types.Relay_Deal || status == types.Relay_Revoked {
		kv = deleteSellOrderKeyValue(kv, order, types.Relay_OnSell)
	}

	return kv
}

func getDeleteSellOrderKv(order *types.RelayOrder) []*types.KeyValue {
	status := order.Status
	var kv []*types.KeyValue
	kv = deleteSellOrderKeyValue(kv, order, status)
	if status == types.Relay_Deal || status == types.Relay_Revoked {
		kv = getSellOrderKeyValue(kv, order, types.Relay_OnSell)
	}

	return kv
}

func getBuyOrderKv(order *types.RelayOrder) []*types.KeyValue {
	status := order.Status
	var kv []*types.KeyValue
	return getBuyOrderKeyValue(kv, order, status)
}

func getDeleteBuyOrderKv(order *types.RelayOrder) []*types.KeyValue {
	status := order.Status
	var kv []*types.KeyValue
	return deleteBuyOrderKeyValue(kv, order, status)

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

	key := getRelayOrderKeyToken(order.Exchgcoin, status, order.Orderid, order.Sellamount, order.Exchgamount, order.Height)
	kv = append(kv, &types.KeyValue{key, orderid})

	key = getRelayOrderKeyAddrStatus(order.Exchgcoin, order.Selladdr, status, order.Orderid, order.Sellamount, order.Exchgamount)
	kv = append(kv, &types.KeyValue{key, orderid})

	key = getRelayOrderKeyAddrToken(order.Exchgcoin, order.Selladdr, status, order.Orderid, order.Sellamount, order.Exchgamount)
	kv = append(kv, &types.KeyValue{key, orderid})

	return kv

}

func deleteSellOrderKeyValue(kv []*types.KeyValue, order *types.RelayOrder, status int32) []*types.KeyValue {

	key := getRelayOrderKeyToken(order.Exchgcoin, status, order.Orderid, order.Sellamount, order.Exchgamount, order.Height)
	kv = append(kv, &types.KeyValue{key, nil})

	key = getRelayOrderKeyAddrStatus(order.Exchgcoin, order.Selladdr, status, order.Orderid, order.Sellamount, order.Exchgamount)
	kv = append(kv, &types.KeyValue{key, nil})

	key = getRelayOrderKeyAddrToken(order.Exchgcoin, order.Selladdr, status, order.Orderid, order.Sellamount, order.Exchgamount)
	kv = append(kv, &types.KeyValue{key, nil})

	return kv
}

func getBuyOrderKeyValue(kv []*types.KeyValue, order *types.RelayOrder, status int32) []*types.KeyValue {
	orderid := []byte(order.Orderid)

	key := getBuyOrderKeyAddr(order.Buyeraddr, order.Exchgcoin, order.Orderid, order.Sellamount, order.Exchgamount)
	kv = append(kv, &types.KeyValue{key, orderid})

	return kv
}

func deleteBuyOrderKeyValue(kv []*types.KeyValue, order *types.RelayOrder, status int32) []*types.KeyValue {

	key := getBuyOrderKeyAddr(order.Buyeraddr, order.Exchgcoin, order.Orderid, order.Sellamount, order.Exchgamount)
	kv = append(kv, &types.KeyValue{key, nil})

	return kv
}
