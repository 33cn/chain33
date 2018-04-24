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

	/* orderid, rawTx, index sibling, blockhash */
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
	if status == Relay_Deal || status == Relay_Revoked {
		kv = deleteSellOrderKeyValue(kv, order, Relay_OnSell)
	}

	return kv
}

func getDeleteSellOrderKv(order *types.RelayOrder) []*types.KeyValue {
	status := order.Status
	var kv []*types.KeyValue
	kv = deleteSellOrderKeyValue(kv, order, status)
	if status == Relay_Deal || status == Relay_Revoked {
		kv = getSellOrderKeyValue(kv, order, Relay_OnSell)
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

	key := getSellOrderKeyToken(order.Exchgcoin, status, order.Orderid, order.Sellamount, order.Exchgamount, order.Height)
	kv = append(kv, &types.KeyValue{key, orderid})

	key = getSellOrderKeyAddrStatus(order.Exchgcoin, order.Selladdr, status, order.Orderid, order.Sellamount, order.Exchgamount)
	kv = append(kv, &types.KeyValue{key, orderid})

	key = getSellOrderKeyAddrToken(order.Exchgcoin, order.Selladdr, status, order.Orderid, order.Sellamount, order.Exchgamount)
	kv = append(kv, &types.KeyValue{key, orderid})

	return kv

}

func deleteSellOrderKeyValue(kv []*types.KeyValue, order *types.RelayOrder, status int32) []*types.KeyValue {

	key := getSellOrderKeyToken(order.Exchgcoin, status, order.Orderid, order.Sellamount, order.Exchgamount, order.Height)
	kv = append(kv, &types.KeyValue{key, nil})

	key = getSellOrderKeyAddrStatus(order.Exchgcoin, order.Selladdr, status, order.Orderid, order.Sellamount, order.Exchgamount)
	kv = append(kv, &types.KeyValue{key, nil})

	key = getSellOrderKeyAddrToken(order.Exchgcoin, order.Selladdr, status, order.Orderid, order.Sellamount, order.Exchgamount)
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
