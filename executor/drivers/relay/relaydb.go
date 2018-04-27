package relay

import (
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/types"
)

type relayDB struct {
	types.RelayOrder
}

func newRelayDB(relayorder *types.RelayOrder) (relaydb *relayDB) {
	relaydb = &relayDB{*relayorder}
	return
}

func (r *relayDB) save(db dbm.KV) []*types.KeyValue {
	set := r.getKVSet()
	for i := 0; i < len(set); i++ {
		db.Set(set[i].GetKey(), set[i].Value)
	}

	return set
}

func (r *relayDB) getKVSet() (kvset []*types.KeyValue) {
	value := types.Encode(&r.RelayOrder)
	key := []byte(r.Orderid)
	kvset = append(kvset, &types.KeyValue{key, value})
	return kvset
}

func (r *relayDB) getSellLogs(relayType int32) *types.ReceiptLog {
	log := &types.ReceiptLog{}
	log.Ty = relayType
	base := &types.ReceiptRelayBase{
		Orderid:        r.Orderid,
		Status:         r.Status,
		Owner:          r.Selladdr,
		Sellamount:     r.Sellamount,
		Exchgcoin:      r.Exchgcoin,
		Exchgamount:    r.Exchgamount,
		Exchgaddr:      r.Exchgaddr,
		Waitcoinblocks: r.Waitcoinblocks,
		Createtime:     r.Createtime,
	}
	if types.TyLogRelaySell == relayType {
		receipt := &types.ReceiptRelaySell{base}
		log.Log = types.Encode(receipt)

	} else if types.TyLogRelayRevoke == relayType {
		receipt := &types.ReceiptRelayRevoke{base}
		log.Log = types.Encode(receipt)
	}

	return log
}

func (r *relayDB) getBuyLogs(txhash string) *types.ReceiptLog {
	log := &types.ReceiptLog{}
	log.Ty = types.TyLogRelayBuy
	receiptBuy := &types.ReceiptRelayBuy{
		Orderid:     r.Orderid,
		Buyeraddr:   r.Buyeraddr,
		Buyamount:   r.Sellamount,
		Exchgcoin:   r.Exchgcoin,
		Exchgamount: r.Exchgamount,
		Exchgaddr:   r.Exchgaddr,
		Buytxhash:   txhash,
	}
	log.Log = types.Encode(receiptBuy)

	return log
}

func (r *relayDB) getVerifyLogs(blocktime int64) *types.ReceiptLog {
	log := &types.ReceiptLog{}
	log.Ty = types.TyLogRelayVerify
	base := &types.ReceiptRelayBase{
		Orderid:         r.Orderid,
		Status:          r.Status,
		Owner:           r.Selladdr,
		Sellamount:      r.Sellamount,
		Exchgcoin:       r.Exchgcoin,
		Exchgamount:     r.Exchgamount,
		Exchgaddr:       r.Exchgaddr,
		Waitcoinblocks:  r.Waitcoinblocks,
		Createtime:      r.Createtime,
		Buyeraddr:       r.Buyeraddr,
		Buyertime:       r.Buyertime,
		Buyercoinheight: r.Buyercoinheight,
		Finishtime:      blocktime,
	}

	receipt := &types.ReceiptRelaySell{base}
	log.Log = types.Encode(receipt)

	return log
}

type relayAction struct {
	coinsAccount *account.DB
	db           dbm.KV
	txhash       string
	fromaddr     string
	blocktime    int64
	height       int64
	execaddr     string
}

func newRelayAction(r *relay, tx *types.Transaction) *relayAction {
	hash := common.Bytes2Hex(tx.Hash())
	fromaddr := account.PubKeyToAddress(tx.GetSignature().GetPubkey()).String()
	return &relayAction{r.GetCoinsAccount(), r.GetStateDB(), hash, fromaddr,
		r.GetBlockTime(), r.GetHeight(), r.GetAddr()}
}

func getRelayOrderFromID(orderid []byte, db dbm.KV) (*types.RelayOrder, error) {
	value, err := db.Get(orderid)
	if err != nil {
		relaylog.Error("getRelayOrderFromID", "Failed to get value from db with orderid", string(orderid))
		return nil, err
	}

	var order types.RelayOrder
	if err = types.Decode(value, &order); err != nil {
		relaylog.Error("getRelayOrderFromID", "Failed to decode order", string(orderid))
		return nil, err
	}
	return &order, nil
}

func (action *relayAction) relaySell(sell *types.RelaySell) (*types.Receipt, error) {

	//冻结子账户资金
	receipt, err := action.coinsAccount.ExecFrozen(action.fromaddr, action.execaddr, sell.Sellamount)
	if err != nil {
		relaylog.Error("account.ExecFrozen relay ", "addrFrom", action.fromaddr, "execaddr", action.execaddr, "amount", sell.Sellamount)
		return nil, err
	}

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	order := &types.RelayOrder{
		Orderid:        calcRelaySellID(action.txhash),
		Status:         types.Relay_OnSell,
		Sellamount:     sell.Sellamount,
		Selladdr:       action.fromaddr,
		Exchgcoin:      sell.Exchgcoin,
		Exchgamount:    sell.Exchgamount,
		Exchgaddr:      sell.Exchgaddr,
		Waitcoinblocks: sell.Waitcoinblocks,
		Createtime:     action.blocktime,
		Height:         action.height,
	}

	relaydb := newRelayDB(order)
	sellOrderKV := relaydb.save(action.db)
	logs = append(logs, receipt.Logs...)
	logs = append(logs, relaydb.getSellLogs(types.TyLogRelaySell))
	kv = append(kv, receipt.KV...)
	kv = append(kv, sellOrderKV...)

	receipt = &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil
}

func (action *relayAction) relayRevokeSell(revoke *types.RelayRevokeSell) (*types.Receipt, error) {
	orderid := []byte(revoke.Orderid)
	order, err := getRelayOrderFromID(orderid, action.db)
	if err != nil {
		return nil, types.ErrTRelayOrderNotExist
	}

	if order.Status == types.Relay_Deal || order.Status == types.Relay_Finished {
		return nil, types.ErrTRelayOrderSoldout
	} else if order.Status == types.Relay_Revoked {
		return nil, types.ErrTRelayOrderRevoked
	}

	if action.fromaddr != order.Selladdr {
		return nil, types.ErrTRelayReturnAddr
	}
	//然后实现购买token的转移,因为这部分token在之前的卖单生成时已经进行冻结
	receipt, err := action.coinsAccount.ExecActive(order.Selladdr, action.execaddr, order.Sellamount)
	if err != nil {
		relaylog.Error("account.ExecActive ", "addrFrom", order.Selladdr, "execaddr", action.execaddr, "amount", order.Sellamount)
		return nil, err
	}

	order.Status = types.Relay_Revoked
	relaydb := newRelayDB(order)
	orderKV := relaydb.save(action.db)

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	logs = append(logs, receipt.Logs...)
	logs = append(logs, relaydb.getSellLogs(types.TyLogRelayRevoke))
	kv = append(kv, receipt.KV...)
	kv = append(kv, orderKV...)
	return &types.Receipt{types.ExecOk, kv, logs}, nil
}

func (action *relayAction) relayBuy(buy *types.RelayBuy) (*types.Receipt, error) {
	orderid := []byte(buy.Orderid)
	order, err := getRelayOrderFromID(orderid, action.db)
	if err != nil {
		return nil, types.ErrTRelayOrderNotExist
	}

	if order.Status == types.Relay_OnSell || order.Status == types.Relay_Finished {
		return nil, types.ErrTRelayOrderSoldout

	}

	if order.Status == types.Relay_Revoked {
		return nil, types.ErrTRelayOrderRevoked
	}

	order.Status = types.Relay_Deal
	order.Buyeraddr = action.fromaddr
	order.Buyertime = action.blocktime
	//order.buyercoinblocknum =

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue

	relaydb := newRelayDB(order)
	sellOrderKV := relaydb.save(action.db)
	logs = append(logs, relaydb.getSellLogs(types.TyLogRelaySell))
	logs = append(logs, relaydb.getBuyLogs(action.txhash))
	kv = append(kv, sellOrderKV...)

	receipt := &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil

}

func (action *relayAction) relayRevokeBuy(revoke *types.RelayRevokeBuy) (*types.Receipt, error) {
	orderidByte := []byte(revoke.Orderid)
	order, err := getRelayOrderFromID(orderidByte, action.db)
	if err != nil {
		return nil, types.ErrTRelayOrderNotExist
	}

	if order.Status == types.Relay_OnSell {
		return nil, types.ErrTRelayOrderOnSell
	} else if order.Status == types.Relay_Revoked {
		return nil, types.ErrTRelayOrderRevoked
	} else if order.Status == types.Relay_Finished {
		return nil, types.ErrTRelayOrderFinished
	}

	if action.fromaddr != order.Buyeraddr {
		return nil, types.ErrTRelayReturnAddr
	}

	order.Status = types.Relay_OnSell
	order.Buyeraddr = ""
	order.Buyertime = 0
	//order.buyercoinblocknum = Nil

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue

	relaydb := newRelayDB(order)
	sellOrderKV := relaydb.save(action.db)
	logs = append(logs, relaydb.getBuyLogs(action.txhash))
	kv = append(kv, sellOrderKV...)

	receipt := &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil
}

func (action *relayAction) relayVerifyBTCTx(verifydata *types.RelayVerifyBTC) (*types.Receipt, error) {
	orderid := []byte(verifydata.Orderid)
	order, err := getRelayOrderFromID(orderid, action.db)
	if err != nil {
		return nil, types.ErrTRelayOrderNotExist
	}

	if order.Status == types.Relay_Finished {
		return nil, types.ErrTRelayOrderSoldout
	} else if order.Status == types.Relay_Revoked {
		return nil, types.ErrTRelayOrderRevoked
	} else if order.Status == types.Relay_OnSell {
		return nil, types.ErrTRelayOrderOnSell
	}

	if action.fromaddr != order.Buyeraddr {
		return nil, types.ErrTRelayReturnAddr
	}

	rst, err := BTCVerifyTx(verifydata)
	if err != nil || rst != true {

		return nil, types.ErrTRelayVerify
	}

	receipt, err := action.coinsAccount.ExecTransferFrozen(order.Selladdr, order.Buyeraddr, action.execaddr, order.Sellamount)
	if err != nil {
		//relaylog.Error()
		return nil, err
	}

	order.Status = types.Relay_Finished
	relaydb := newRelayDB(order)
	orderKV := relaydb.save(action.db)

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	logs = append(logs, receipt.Logs...)
	logs = append(logs, relaydb.getSellLogs(types.TyLogRelaySell))
	logs = append(logs, relaydb.getVerifyLogs(action.blocktime))
	kv = append(kv, receipt.KV...)
	kv = append(kv, orderKV...)
	return &types.Receipt{types.ExecOk, kv, logs}, nil

}
