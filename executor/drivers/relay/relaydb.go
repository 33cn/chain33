package relay

import (
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/types"
	"strconv"
)

// relay status
const (
	Relay_OnSell = iota
	Relay_Deal
	Relay_Finished
	Relay_Revoked
)

var RelayOrderStatus = map[int32]string{
	Relay_OnSell:   "OnSell",
	Relay_Deal:     "Deal",
	Relay_Finished: "Finished",
	Relay_Revoked:  "Revoked",
}

type relayDB struct {
	types.RelayOrder
}

func newRelayDB(relayorder *types.RelayOrder) (relaydb *relayDB) {
	relaydb = &relayDB{*relayorder}
	return
}

func (r *relayDB) save(db dbm.KVDB) []*types.KeyValue {
	set := r.getKVSet()
	for i := 0; i < len(set); i++ {
		db.Set(set[i].GetKey(), set[i].Value)
	}

	return set
}

func (r *relayDB) getKVSet() (kvset []*types.KeyValue) {
	value := types.Encode(&r.RelayOrder)
	key := []byte(r.orderid)
	kvset = append(kvset, &types.KeyValue{key, value})
	return kvset
}

func (r *relayDB) getSellLogs(relayType int32) *types.ReceiptLog {
	log := &types.ReceiptLog{}
	log.Ty = relayType
	base := &types.ReceiptRelayBase{
		orderid:        r.orderid,
		status:         RelayOrderStatus[r.status],
		owner:          r.selladdr,
		sellamount:     r.sellamount,
		exchgcoin:      r.exchgcoin,
		exchgamount:    r.exchgamount,
		exchgaddr:      r.exchgaddr,
		waitcoinblocks: r.waitcoinblocks,
		createtime:     r.createtime,
	}
	if types.TyLogRelaySell == relayType {
		receipt := &types.ReceiptRelaySell{*base}
		log.Log = types.Encode(receipt)

	} else if types.TyLogRelayRevoke == relayType {
		receipt := &types.ReceiptRelayRevoke{*base}
		log.Log = types.Encode(receipt)
	}

	return log
}

func (r *relayDB) getBuyLogs(txhash string) *types.ReceiptLog {
	log := &types.ReceiptLog{}
	log.Ty = types.TyLogRelayBuy
	receiptBuy := &types.ReceiptRelayBuy{
		orderid:      r.orderid,
		buyeraddr:    r.buyeraddr,
		buyamount:    r.sellamount,
		exchgcoin:    r.exchgcoin,
		exchgamount:  r.exchgamount,
		exchgaccount: r.exchgaccount,
		buytxhash:    txhash,
	}
	log.Log = types.Encode(receiptBuy)

	return log
}

func (r *relayDB) getVerifyLogs(blocktime int64) *types.ReceiptLog {
	log := &types.ReceiptLog{}
	log.Ty = types.TyLogRelayVerify
	base := &types.ReceiptRelayBase{
		orderid:           r.orderid,
		status:            RelayOrderStatus[r.status],
		owner:             r.selladdr,
		sellamount:        r.sellamount,
		exchgcoin:         r.exchgcoin,
		exchgamount:       r.exchgamount,
		exchgaddr:         r.exchgaddr,
		waitcoinblocks:    r.waitcoinblocks,
		createtime:        r.createtime,
		buyeraddr:         r.buyeraddr,
		buyertime:         r.buyertime,
		buyercoinblocknum: r.buyercoinblocknum,
		finishtime:        blocktime,
	}

	receipt := &types.ReceiptRelaySell{*base}
	log.Log = types.Encode(receipt)

	return log
}

type relayAction struct {
	coinsAccount *account.AccountDB
	db           dbm.KVDB
	txhash       string
	fromaddr     string
	blocktime    int64
	height       int64
	execaddr     string
}

func newRelayAction(r *relay, tx *types.Transaction) *relayAction {
	hash := common.Bytes2Hex(tx.Hash())
	fromaddr := account.PubKeyToAddress(tx.GetSignature().GetPubkey()).String()
	return &relayAction{r.GetCoinsAccount(), r.GetDB(), hash, fromaddr,
		r.GetBlockTime(), r.GetHeight(), r.GetAddr()}
}

func getRelayOrderFromID(orderid []byte, db dbm.KVDB) (*types.RelayOrder, error) {
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
	receipt, err := action.coinsAccount.ExecFrozen(action.fromaddr, action.execaddr, sell.Amount)
	if err != nil {
		relaylog.Error("account.ExecFrozen relay ", "addrFrom", action.fromaddr, "execaddr", action.execaddr, "amount", sell.Amount)
		return nil, err
	}

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	order := &types.RelayOrder{
		orderid:        calcRelaySellID(action.txhash),
		staus:          Relay_OnSell,
		sellamount:     sell.Amount,
		selladdr:       action.fromaddr,
		exchgcoin:      sell.Exchgcoin,
		exchgamount:    sell.Exchgamount,
		exchgaddr:      sell.Exchgaddr,
		waitcoinblocks: sell.Waitcoinblocks,
		createtime:     action.blocktime,
		height:         action.height,
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
	orderid := []byte(revoke.orderid)
	order, err := getRelayOrderFromID(orderid, action.db)
	if err != nil {
		return nil, types.ErrTRelayOrderNotExist
	}

	if order.Status == Relay_Deal || order.Status == Relay_Finished {
		return nil, types.ErrTRelayOrderSoldout
	} else if order.Status == Relay_Revoked {
		return nil, types.ErrTRelayOrderRevoked
	}

	if action.fromaddr != order.selladdr {
		return nil, types.ErrTRelayReturnAddr
	}
	//然后实现购买token的转移,因为这部分token在之前的卖单生成时已经进行冻结
	receipt, err := action.coinsAccount.ExecActive(order.selladdr, action.execaddr, order.sellamount)
	if err != nil {
		relaylog.Error("account.ExecActive ", "addrFrom", order.selladdr, "execaddr", action.execaddr, "amount", order.sellamount)
		return nil, err
	}

	order.Status = Relay_Revoked
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
	orderid := []byte(buy.orderid)
	order, err := getRelayOrderFromID(orderid, action.db)
	if err != nil {
		return nil, types.ErrTRelayOrderNotExist
	}

	if order.status == types.RelayOnsell || order.status == types.RelayFinished {
		return nil, types.ErrTRelayOrderSoldout

	}

	if order.status == types.RelayRevoked {
		return nil, types.ErrTRelayOrderRevoked
	}

	order.status = Relay_Deal
	order.buyeraddr = action.fromaddr
	order.buyertime = action.blocktime
	//order.buyercoinblocknum =

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue

	relaydb := newRelayDB(order)
	sellOrderKV := relaydb.save(action.db)
	logs = append(logs, relaydb.getBuyLogs(action.txhash))
	kv = append(kv, sellOrderKV...)

	receipt = &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil

}

func (action *relayAction) relayRevokeBuy(revoke *types.RelayRevokeBuy) (*types.Receipt, error) {
	orderidByte := []byte(revoke.orderid)
	order, err := getRelayOrderFromID(orderidByte, action.db)
	if err != nil {
		return nil, types.ErrTRelayOrderNotExist
	}

	if order.Status == Relay_OnSell {
		return nil, types.ErrTRelayOrderOnsell
	} else if order.Status == Relay_Revoked {
		return nil, types.ErrTRelayOrderRevoked
	} else if order.Status == Relay_Finished {
		return nil, types.ErrTRelayOrderFinished
	}

	if action.fromaddr != order.buyeraddr {
		return nil, types.ErrTRelayReturnAddr
	}

	order.status = Relay_OnSell
	order.buyeraddr = nil
	order.buyertime = nil
	//order.buyercoinblocknum = Nil

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue

	relaydb := newRelayDB(order)
	sellOrderKV := relaydb.save(action.db)
	logs = append(logs, relaydb.getBuyLogs(action.txhash))
	kv = append(kv, sellOrderKV...)

	receipt = &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil
}

func (action *relayAction) relayVerifyCoinTx(verifydata *types.RelayVerify) (*types.Receipt, error) {
	orderid := []byte(verifydata.orderid)
	order, err := getRelayOrderFromID(orderid, action.db)
	if err != nil {
		return nil, types.ErrTRelayOrderNotExist
	}

	if order.Status == Relay_Finished {
		return nil, types.ErrTRelayOrderSoldout
	} else if order.Status == Relay_Revoked {
		return nil, types.ErrTRelayOrderRevoked
	} else if order.Status == Relay_OnSell {
		return nil, types.ErrTRelayOrderOnSell
	}

	if action.fromaddr != order.buyeraddr {
		return nil, types.ErrTRelayReturnAddr
	}

	rst, err = BTCVerifyTx(verifydata)
	if err != nil || rst != true {

		return err, types.ErrTRelayVerify
	}

	receipt, err := action.coinsAccount.ExecTransferFrozen(order.selladdr, order.buyeraddr, action.execaddr, order.sellamount)
	if err != nil {
		relaylog.Error()
		return nil, err
	}

	order.Status = Relay_Finished
	relaydb := newRelayDB(order)
	orderKV := relaydb.save(action.db)

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	logs = append(logs, receipt.Logs...)
	logs = append(logs, relaydb.getVerifyLogs(action.blocktime))
	kv = append(kv, receipt.KV...)
	kv = append(kv, orderKV...)
	return &types.Receipt{types.ExecOk, kv, logs}, nil

}
