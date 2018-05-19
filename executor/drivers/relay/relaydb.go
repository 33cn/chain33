package relay

import (
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/types"
)

//当前方案没有考虑验证失败后允许buyer重试机制， 待整体功能实现后，再讨论要不要增加buyer重试机制
//如果购买订单和提交交易过后4小时，仍未交易成功，则seller可以收回订单
const LOCKING_PERIOD = 4 * 60
const BUY_CANCLED = "BUY_CANCLED"

//TODO
// 1,仔细检查状态切换和 命令接受时的状态允许
// 2, confirm 的状态检查

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

func (r *relayDB) getBuyLogs(relayLogType int32, txhash string) *types.ReceiptLog {
	log := &types.ReceiptLog{}
	log.Ty = relayLogType
	receiptBuy := &types.ReceiptRelayBuy{
		Orderid:     r.Orderid,
		Status:      r.Status,
		Buyeraddr:   r.Buyeraddr,
		Buyamount:   r.Sellamount,
		Exchgcoin:   r.Exchgcoin,
		Exchgamount: r.Exchgamount,
		Exchgaddr:   r.Exchgaddr,
		Exchgtxhash: r.Exchgtxhash,
		Buytxhash:   txhash,
	}
	log.Log = types.Encode(receiptBuy)

	return log
}

func (r *relayDB) getSellLogs(relayLogType int32) *types.ReceiptLog {
	log := &types.ReceiptLog{}
	log.Ty = relayLogType
	base := &types.ReceiptRelayBase{
		Orderid:        r.Orderid,
		Status:         r.Status,
		Selladdr:       r.Selladdr,
		Sellamount:     r.Sellamount,
		Exchgcoin:      r.Exchgcoin,
		Exchgamount:    r.Exchgamount,
		Exchgaddr:      r.Exchgaddr,
		Waitcoinblocks: r.Waitcoinblocks,
		Createtime:     r.Createtime,
		Buyeraddr:      r.Buyeraddr,
		Buyertime:      r.Buytime,
		Finishtime:     r.Finishtime,
		Finishresult:   r.Finishresult,
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
	receipt, err := action.coinsAccount.ExecFrozen(action.fromaddr, action.execaddr, int64(sell.Sellamount))
	if err != nil {
		relaylog.Error("account.ExecFrozen relay ", "addrFrom", action.fromaddr, "execaddr", action.execaddr, "amount", sell.Sellamount)
		return nil, err
	}

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	order := &types.RelayOrder{
		Orderid:        calcRelayOrderID(action.txhash),
		Status:         types.RelayOrderStatus_pending,
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

	if order.Status == types.RelayOrderStatus_locking {
		if action.blocktime-order.Buytime < LOCKING_PERIOD {
			relaylog.Error("relay revoke sell ", "action.blocktime-order.Buytime", action.blocktime-order.Buytime, "less", LOCKING_PERIOD)
			return nil, types.ErrTime
		}
	}

	if order.Status == types.RelayOrderStatus_confirming {
		if action.blocktime-order.Confirmtime < LOCKING_PERIOD {
			relaylog.Error("relay revoke sell ", "action.blocktime-order.Confirmtime", action.blocktime-order.Confirmtime, "less", LOCKING_PERIOD)
			return nil, types.ErrTime
		}
	}

	if order.Status == types.RelayOrderStatus_finished {
		return nil, types.ErrTRelayOrderSoldout
	} else if order.Status == types.RelayOrderStatus_canceled {
		return nil, types.ErrTRelayOrderRevoked
	}

	if action.fromaddr != order.Selladdr {
		return nil, types.ErrTRelayReturnAddr
	}

	//然后实现购买token的转移,因为这部分token在之前的卖单生成时已经进行冻结
	receipt, err := action.coinsAccount.ExecActive(order.Selladdr, action.execaddr, int64(order.Sellamount))
	if err != nil {
		relaylog.Error("account.ExecActive ", "addrFrom", order.Selladdr, "execaddr", action.execaddr, "amount", order.Sellamount)
		return nil, err
	}

	order.Status = types.RelayOrderStatus_canceled
	relaydb := newRelayDB(order)
	orderKV := relaydb.save(action.db)

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	logs = append(logs, receipt.Logs...)
	logs = append(logs, relaydb.getSellLogs(types.TyLogRelayRevokeSell))
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

	if order.Status == types.RelayOrderStatus_locking || order.Status == types.RelayOrderStatus_confirming || order.Status == types.RelayOrderStatus_finished {
		return nil, types.ErrTRelayOrderSoldout

	}

	if order.Status == types.RelayOrderStatus_canceled {
		return nil, types.ErrTRelayOrderRevoked
	}

	order.Status = types.RelayOrderStatus_locking
	order.Buyeraddr = action.fromaddr
	order.Buytime = action.blocktime
	//order.Lockingtime = action.blocktime
	//order.buyercoinblocknum =

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue

	relaydb := newRelayDB(order)
	sellOrderKV := relaydb.save(action.db)
	logs = append(logs, relaydb.getSellLogs(types.TyLogRelaySell))
	logs = append(logs, relaydb.getBuyLogs(types.TyLogRelayBuy, action.txhash))
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

	if order.Status == types.RelayOrderStatus_pending || order.Status == types.RelayOrderStatus_canceled {
		return nil, types.ErrTRelayOrderRevoked
	} else if order.Status == types.RelayOrderStatus_confirming {
		return nil, types.ErrTRelayOrderConfirming
	} else if order.Status == types.RelayOrderStatus_finished {
		return nil, types.ErrTRelayOrderFinished
	}

	if action.fromaddr != order.Buyeraddr {
		return nil, types.ErrTRelayReturnAddr
	}

	order.Status = types.RelayOrderStatus_pending
	order.Buyeraddr = BUY_CANCLED
	order.Buytime = 0
	//order.buyercoinblocknum = Nil

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue

	relaydb := newRelayDB(order)
	sellOrderKV := relaydb.save(action.db)
	logs = append(logs, relaydb.getSellLogs(types.TyLogRelaySell))
	logs = append(logs, relaydb.getBuyLogs(types.TyLogRelayRevokeBuy, action.txhash))
	kv = append(kv, sellOrderKV...)

	receipt := &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil
}

func (action *relayAction) relayConfirmTx(confirm *types.RelayConfirmTx) (*types.Receipt, error) {
	orderid := []byte(confirm.Orderid)
	order, err := getRelayOrderFromID(orderid, action.db)
	if err != nil {
		return nil, types.ErrTRelayOrderNotExist
	}

	if order.Status == types.RelayOrderStatus_pending {
		return nil, types.ErrTRelayOrderOnSell

	} else if order.Status == types.RelayOrderStatus_finished {
		return nil, types.ErrTRelayOrderSoldout
	} else if order.Status == types.RelayOrderStatus_confirming {
		return nil, types.ErrTRelayOrderConfirming
	} else if order.Status == types.RelayOrderStatus_canceled {
		return nil, types.ErrTRelayOrderRevoked
	}

	if action.fromaddr != order.Buyeraddr {
		return nil, types.ErrTRelayReturnAddr
	}

	order.Status = types.RelayOrderStatus_confirming
	order.Confirmtime = action.blocktime
	order.Exchgtxhash = confirm.Txhash
	//若超过3次仍未验证通过，则seller随时可以撤回交易
	//order.Lockingtimes = order.Lockingtimes+1
	//order.buyercoinblocknum =

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue

	relaydb := newRelayDB(order)
	sellOrderKV := relaydb.save(action.db)
	logs = append(logs, relaydb.getSellLogs(types.TyLogRelaySell))
	logs = append(logs, relaydb.getBuyLogs(types.TyLogRelayBuy, action.txhash))
	kv = append(kv, sellOrderKV...)

	receipt := &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil

}

func (action *relayAction) relayVerifyTx(verifydata *types.RelayVerify, r *relay) (*types.Receipt, error) {
	orderid := []byte(verifydata.Orderid)
	order, err := getRelayOrderFromID(orderid, action.db)
	if err != nil {
		return nil, types.ErrTRelayOrderNotExist
	}

	if order.Status == types.RelayOrderStatus_finished {
		return nil, types.ErrTRelayOrderSoldout
	} else if order.Status == types.RelayOrderStatus_canceled {
		return nil, types.ErrTRelayOrderRevoked
	} else if order.Status == types.RelayOrderStatus_pending {
		return nil, types.ErrTRelayOrderOnSell
	}

	var receipt *types.Receipt

	rst, err := r.btcstore.verifyTx(verifydata, order)
	if err != nil || rst != true {

		err = types.ErrTRelayVerify
	}

	if rst {
		receipt, err = action.coinsAccount.ExecTransferFrozen(order.Selladdr, order.Buyeraddr, action.execaddr, int64(order.Sellamount))
	}

	order.Status = types.RelayOrderStatus_finished
	order.Finishtime = action.blocktime
	order.Finishresult = err.Error()
	relaydb := newRelayDB(order)
	orderKV := relaydb.save(action.db)

	if err != nil {
		return nil, err
	}

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	logs = append(logs, receipt.Logs...)
	logs = append(logs, relaydb.getSellLogs(types.TyLogRelaySell))
	logs = append(logs, relaydb.getBuyLogs(types.TyLogRelayBuy, action.txhash))
	kv = append(kv, receipt.KV...)
	kv = append(kv, orderKV...)
	return &types.Receipt{types.ExecOk, kv, logs}, nil

}

func (action *relayAction) relayVerifyBTCTx(verifydata *types.RelayVerifyBTC, r *relay) (*types.Receipt, error) {
	orderid := []byte(verifydata.Orderid)
	order, err := getRelayOrderFromID(orderid, action.db)
	if err != nil {
		return nil, types.ErrTRelayOrderNotExist
	}

	if order.Status == types.RelayOrderStatus_finished {
		return nil, types.ErrTRelayOrderSoldout
	} else if order.Status == types.RelayOrderStatus_canceled {
		return nil, types.ErrTRelayOrderRevoked
	} else if order.Status == types.RelayOrderStatus_pending {
		return nil, types.ErrTRelayOrderOnSell
	}

	var receipt *types.Receipt

	rst, err := r.btcstore.verifyBTCTx(verifydata)
	if err != nil || rst != true {

		err = types.ErrTRelayVerify
	}

	if rst {
		receipt, err = action.coinsAccount.ExecTransferFrozen(order.Selladdr, order.Buyeraddr, action.execaddr, int64(order.Sellamount))
	}

	order.Status = types.RelayOrderStatus_finished
	order.Finishtime = action.blocktime
	if err != nil {
		order.Finishresult = err.Error()
	} else {
		order.Finishresult = "tx success"
	}
	relaydb := newRelayDB(order)
	orderKV := relaydb.save(action.db)

	if err != nil {
		return nil, err
	}

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	logs = append(logs, receipt.Logs...)
	logs = append(logs, relaydb.getSellLogs(types.TyLogRelaySell))
	logs = append(logs, relaydb.getBuyLogs(types.TyLogRelayBuy, action.txhash))
	kv = append(kv, receipt.KV...)
	kv = append(kv, orderKV...)
	return &types.Receipt{types.ExecOk, kv, logs}, nil

}

func getBTCHeaderLog(relayLogType int32, head *types.BtcHeader) *types.ReceiptLog {
	log := &types.ReceiptLog{}
	log.Ty = relayLogType
	receipt := &types.ReceiptRelayRcvBTCHeaders{head}

	log.Log = types.Encode(receipt)

	return log

}

func (action *relayAction) relaySaveBTCHeader(headers *types.BtcHeaders) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kv []*types.KeyValue

	//TODO make some simple check
	for _, head := range headers.BtcHeader {
		relaylog.Info("relaySaveBTCHeader", "heigth", head.Height)
		logs = append(logs, getBTCHeaderLog(types.TyLogRelayRcvBTCHead, head))
	}

	return &types.Receipt{types.ExecOk, kv, logs}, nil
}
