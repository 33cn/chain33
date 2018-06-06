package relay

import (
	"time"

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/types"
)

//当前方案没有考虑验证失败后允许buyer重试机制， 待整体功能实现后，再讨论要不要增加buyer重试机制
//如果购买订单和提交交易过后4小时，仍未交易成功，则seller可以收回订单
const LOCKING_PERIOD = 6 * time.Hour
const ACCEPT_CANCLED = "ACCEPT_CANCLED"
const WAIT_BLOCK_HEIGHT = 6

//TODO
// 1,仔细检查状态切换和 命令接受时的状态允许
// 2, confirm 的状态检查

type relayDB struct {
	types.RelayOrder
}

func newRelayDB(order *types.RelayOrder) *relayDB {
	return &relayDB{*order}
}

func (r *relayDB) save(db dbm.KV) []*types.KeyValue {
	set := r.getKVSet()
	for i := 0; i < len(set); i++ {
		db.Set(set[i].GetKey(), set[i].Value)
	}

	return set
}

func (r *relayDB) getKVSet() (kvSet []*types.KeyValue) {
	value := types.Encode(&r.RelayOrder)
	key := []byte(r.Id)
	kvSet = append(kvSet, &types.KeyValue{key, value})
	return kvSet
}

func (r *relayDB) getAcceptLogs(ty int32) *types.ReceiptLog {
	log := &types.ReceiptLog{}
	log.Ty = ty
	accept := &types.ReceiptRelayAccept{
		OrderId:     r.Id,
		Status:      r.Status,
		AcceptAddr:  r.AcceptAddr,
		Amount:      r.Amount,
		Exchgcoin:   r.Coin,
		Exchgamount: r.CoinAmount,
		Exchgaddr:   r.CoinAddr,
		Exchgtxhash: r.CoinTxHash,
		AcceptTime:  r.AcceptTime,
		ConfirmTime: r.ConfirmTime,
	}
	log.Log = types.Encode(accept)

	return log
}

func (r *relayDB) getCreateLogs(relayLogType int32) *types.ReceiptLog {
	log := &types.ReceiptLog{}
	log.Ty = relayLogType
	base := &types.ReceiptRelayBase{
		OrderId:     r.Id,
		Status:      r.Status,
		CreateAddr:  r.CreaterAddr,
		Amount:      r.Amount,
		Coin:        r.Coin,
		CoinAmount:  r.CoinAmount,
		CoinAddr:    r.CoinAddr,
		AcceptAddr:  r.AcceptAddr,
		AcceptTime:  r.AcceptTime,
		ConfirmTime: r.ConfirmTime,
		FinishTime:  r.FinishTime,
	}

	receipt := &types.ReceiptRelayCreate{base}
	log.Log = types.Encode(receipt)

	return log
}

type relayAction struct {
	coinsAccount *account.DB
	db           dbm.KV
	txHash       string
	fromAddr     string
	blockTime    int64
	height       int64
	execAddr     string
}

func newRelayAction(r *relay, tx *types.Transaction) *relayAction {
	hash := common.Bytes2Hex(tx.Hash())
	fromAddr := account.PubKeyToAddress(tx.GetSignature().GetPubkey()).String()
	return &relayAction{r.GetCoinsAccount(), r.GetStateDB(), hash, fromAddr,
		r.GetBlockTime(), r.GetHeight(), r.GetAddr()}
}

func getRelayOrderFromID(orderId []byte, db dbm.KV) (*types.RelayOrder, error) {
	value, err := db.Get(orderId)
	if err != nil {
		relaylog.Error("getRelayOrderFromID", "Failed to get value from db with orderId", string(orderId))
		return nil, err
	}

	var order types.RelayOrder
	if err = types.Decode(value, &order); err != nil {
		relaylog.Error("getRelayOrderFromID", "Failed to decode order", string(orderId))
		return nil, err
	}
	return &order, nil
}

func (action *relayAction) relayCreate(order *types.RelayCreate) (*types.Receipt, error) {
	var receipt *types.Receipt
	var err error
	var coinAddr string
	if order.Operation == types.RelayCreateBuy {
		receipt, err = action.coinsAccount.ExecFrozen(action.fromAddr, action.execAddr, int64(order.BtyAmount))
		if err != nil {
			relaylog.Error("account.ExecFrozen relay ", "addrFrom", action.fromAddr, "execAddr", action.execAddr, "amount", order.BtyAmount)
			return nil, err
		}

		coinAddr = order.Addr
	}

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	uOrder := &types.RelayOrder{
		Id:            calcRelayOrderID(action.txHash),
		Status:        types.RelayOrderStatus_pending,
		PreStatus:     types.RelayOrderStatus_init,
		Amount:        order.BtyAmount,
		CreaterAddr:   action.fromAddr,
		CoinOperation: order.Operation,
		Coin:          order.Coin,
		CoinAmount:    order.Amount,
		CoinAddr:      coinAddr,
		CreateTime:    action.blockTime,
		Height:        action.height,
	}

	if receipt != nil {
		logs = append(logs, receipt.Logs...)
		kv = append(kv, receipt.KV...)
	}
	relayDb := newRelayDB(uOrder)
	sellOrderKV := relayDb.save(action.db)
	logs = append(logs, relayDb.getCreateLogs(types.TyLogRelayCreate))
	kv = append(kv, sellOrderKV...)

	return &types.Receipt{types.ExecOk, kv, logs}, nil
}

func checkRevokeOrder(order *types.RelayOrder, blockTime int64) error {
	nowTime := time.Now()

	if order.Status == types.RelayOrderStatus_locking {
		acceptTime := time.Unix(order.AcceptTime, 0)
		if nowTime.Sub(acceptTime) < LOCKING_PERIOD {
			relaylog.Error("relay revoke sell ", "action.blockTime-order.Buytime", nowTime.Sub(acceptTime), "less", LOCKING_PERIOD)
			return types.ErrTime
		}
	}

	if order.Status == types.RelayOrderStatus_confirming {
		confirmTime := time.Unix(order.ConfirmTime, 0)
		if nowTime.Sub(confirmTime) < 4*LOCKING_PERIOD {
			relaylog.Error("relay revoke sell ", "action.blockTime-order.Confirmtime", nowTime.Sub(confirmTime), "less", 4*LOCKING_PERIOD)
			return types.ErrTime
		}
	}

	return nil

}

func (action *relayAction) relayRevokeCreate(revoke *types.RelayRevokeCreate) (*types.Receipt, error) {
	orderId := []byte(revoke.OrderId)
	order, err := getRelayOrderFromID(orderId, action.db)
	if err != nil {
		return nil, types.ErrTRelayOrderNotExist
	}

	err = checkRevokeOrder(order, action.blockTime)
	if err != nil {
		return nil, err
	}

	if order.Status == types.RelayOrderStatus_finished {
		return nil, types.ErrTRelayOrderSoldout
	} else if order.Status == types.RelayOrderStatus_canceled {
		return nil, types.ErrTRelayOrderRevoked
	}

	if action.fromAddr != order.CreaterAddr {
		return nil, types.ErrTRelayReturnAddr
	}

	var receipt *types.Receipt
	if order.CoinOperation == types.RelayCreateBuy {
		receipt, err = action.coinsAccount.ExecActive(order.CreaterAddr, action.execAddr, int64(order.Amount))
		if err != nil {
			relaylog.Error("account.ExecActive ", "addrFrom", order.CreaterAddr, "execAddr", action.execAddr, "amount", order.Amount)
			return nil, err
		}
	}

	if order.CoinOperation == types.RelayCreateSell && order.Status != types.RelayOrderStatus_pending {
		receipt, err = action.coinsAccount.ExecActive(order.AcceptAddr, action.execAddr, int64(order.Amount))
		if err != nil {
			relaylog.Error("account.ExecActive ", "addrFrom", order.AcceptAddr, "execAddr", action.execAddr, "amount", order.Amount)
			return nil, err
		}
	}

	order.PreStatus = order.Status
	if revoke.Action == types.RelayUnlockOrder {
		order.Status = types.RelayOrderStatus_pending
	} else {
		order.Status = types.RelayOrderStatus_canceled
	}

	relayDb := newRelayDB(order)
	orderKV := relayDb.save(action.db)

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	if receipt != nil {
		logs = append(logs, receipt.Logs...)
		kv = append(kv, receipt.KV...)
	}
	logs = append(logs, relayDb.getCreateLogs(types.TyLogRelayRevokeCreate))
	if order.PreStatus != types.RelayOrderStatus_init {
		logs = append(logs, relayDb.getAcceptLogs(types.TyLogRelayRevokeCreate))
	}

	kv = append(kv, orderKV...)
	return &types.Receipt{types.ExecOk, kv, logs}, nil
}

func (action *relayAction) relayAccept(accept *types.RelayAccept) (*types.Receipt, error) {
	orderId := []byte(accept.OrderId)
	order, err := getRelayOrderFromID(orderId, action.db)
	if err != nil {
		return nil, types.ErrTRelayOrderNotExist
	}

	if order.Status == types.RelayOrderStatus_canceled {
		return nil, types.ErrTRelayOrderRevoked
	}

	if order.Status != types.RelayOrderStatus_pending {
		return nil, types.ErrTRelayOrderSoldout
	}

	var receipt *types.Receipt
	if order.CoinOperation == types.RelayCreateSell {
		if accept.CoinAddr == "" {
			relaylog.Error("relayAccept, for sell operation, coinAddr needed")
			return nil, types.ErrTRelayOrderParamErr
		}

		order.CoinAddr = accept.CoinAddr

		receipt, err = action.coinsAccount.ExecFrozen(action.fromAddr, action.execAddr, int64(order.Amount))
		if err != nil {
			relaylog.Error("account.ExecFrozen relay ", "addrFrom", action.fromAddr, "execAddr", action.execAddr, "amount", order.Amount)
			return nil, err
		}
	}

	order.PreStatus = order.Status
	order.Status = types.RelayOrderStatus_locking
	order.AcceptAddr = action.fromAddr
	order.AcceptTime = action.blockTime

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue

	if receipt != nil {
		logs = append(logs, receipt.Logs...)
		kv = append(kv, receipt.KV...)
	}

	relayDb := newRelayDB(order)
	sellOrderKV := relayDb.save(action.db)

	logs = append(logs, relayDb.getCreateLogs(types.TyLogRelayAccept))
	logs = append(logs, relayDb.getAcceptLogs(types.TyLogRelayAccept))
	kv = append(kv, sellOrderKV...)

	return &types.Receipt{types.ExecOk, kv, logs}, nil

}

func (action *relayAction) relayRevokeAccept(revoke *types.RelayRevokeAccept) (*types.Receipt, error) {
	orderIdByte := []byte(revoke.OrderId)
	order, err := getRelayOrderFromID(orderIdByte, action.db)
	if err != nil {
		return nil, types.ErrTRelayOrderNotExist
	}

	if order.Status == types.RelayOrderStatus_pending || order.Status == types.RelayOrderStatus_canceled {
		return nil, types.ErrTRelayOrderRevoked
	} else if order.Status == types.RelayOrderStatus_finished {
		return nil, types.ErrTRelayOrderFinished
	}

	err = checkRevokeOrder(order, action.blockTime)
	if err != nil {
		return nil, err
	}

	if action.fromAddr != order.AcceptAddr {
		return nil, types.ErrTRelayReturnAddr
	}

	var receipt *types.Receipt
	if order.CoinOperation == types.RelayCreateSell {
		receipt, err = action.coinsAccount.ExecActive(order.AcceptAddr, action.execAddr, int64(order.Amount))
		if err != nil {
			relaylog.Error("account.ExecActive ", "addrFrom", order.AcceptAddr, "execAddr", action.execAddr, "amount", order.Amount)
			return nil, err
		}
		order.CoinAddr = ""
	}

	order.PreStatus = order.Status
	order.Status = types.RelayOrderStatus_pending
	order.AcceptAddr = ""
	order.AcceptTime = 0

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	if receipt != nil {
		logs = append(logs, receipt.Logs...)
		kv = append(kv, receipt.KV...)
	}

	relayDb := newRelayDB(order)
	sellOrderKV := relayDb.save(action.db)
	logs = append(logs, relayDb.getCreateLogs(types.TyLogRelayRevokeAccept))
	logs = append(logs, relayDb.getAcceptLogs(types.TyLogRelayRevokeAccept))
	kv = append(kv, sellOrderKV...)

	return &types.Receipt{types.ExecOk, kv, logs}, nil
}

func (action *relayAction) relayConfirmTx(confirm *types.RelayConfirmTx) (*types.Receipt, error) {
	orderId := []byte(confirm.OrderId)
	order, err := getRelayOrderFromID(orderId, action.db)
	if err != nil {
		return nil, types.ErrTRelayOrderNotExist
	}

	if order.Status == types.RelayOrderStatus_pending {
		return nil, types.ErrTRelayOrderOnSell
	} else if order.Status == types.RelayOrderStatus_finished {
		return nil, types.ErrTRelayOrderSoldout
	} else if order.Status == types.RelayOrderStatus_canceled {
		return nil, types.ErrTRelayOrderRevoked
	}

	var confirmAddr string
	if order.CoinOperation == types.RelayCreateBuy {
		confirmAddr = order.AcceptAddr
	} else {
		confirmAddr = order.CreaterAddr
	}
	if action.fromAddr != confirmAddr {
		relaylog.Error("relayConfirmTx", "fromAddr", action.fromAddr, "not confirmAddr", confirmAddr, "oper", order.CoinOperation)
		return nil, types.ErrTRelayReturnAddr
	}

	order.PreStatus = order.Status
	order.Status = types.RelayOrderStatus_confirming
	order.ConfirmTime = action.blockTime
	order.CoinTxHash = confirm.TxHash

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue

	relayDb := newRelayDB(order)
	sellOrderKV := relayDb.save(action.db)
	logs = append(logs, relayDb.getCreateLogs(types.TyLogRelayConfirmTx))
	logs = append(logs, relayDb.getAcceptLogs(types.TyLogRelayConfirmTx))
	kv = append(kv, sellOrderKV...)

	receipt := &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil

}

func (action *relayAction) relayVerifyTx(verify *types.RelayVerify, r *relay) (*types.Receipt, error) {
	orderId := []byte(verify.OrderId)
	order, err := getRelayOrderFromID(orderId, action.db)
	if err != nil {
		return nil, types.ErrTRelayOrderNotExist
	}

	if order.Status == types.RelayOrderStatus_finished {
		return nil, types.ErrTRelayOrderSoldout
	} else if order.Status == types.RelayOrderStatus_canceled {
		return nil, types.ErrTRelayOrderRevoked
	} else if order.Status == types.RelayOrderStatus_pending || order.Status == types.RelayOrderStatus_locking {
		return nil, types.ErrTRelayOrderOnSell
	}

	err = r.btcStore.verifyTx(verify, order)
	if err != nil {
		return nil, err
	}

	var receipt *types.Receipt
	if order.CoinOperation == types.RelayCreateBuy {
		receipt, err = action.coinsAccount.ExecTransferFrozen(order.CreaterAddr, order.AcceptAddr, action.execAddr, int64(order.Amount))

	} else {
		receipt, err = action.coinsAccount.ExecTransferFrozen(order.AcceptAddr, order.CreaterAddr, action.execAddr, int64(order.Amount))
	}

	if err != nil {
		relaylog.Error("relay verify tx transfer fail", "error", err.Error())
		return nil, err
	}

	order.PreStatus = order.Status
	order.Status = types.RelayOrderStatus_finished
	order.FinishTime = action.blockTime

	relayDb := newRelayDB(order)
	orderKV := relayDb.save(action.db)

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	logs = append(logs, receipt.Logs...)
	logs = append(logs, relayDb.getCreateLogs(types.TyLogRelayFinishTx))
	logs = append(logs, relayDb.getAcceptLogs(types.TyLogRelayFinishTx))
	kv = append(kv, receipt.KV...)
	kv = append(kv, orderKV...)
	return &types.Receipt{types.ExecOk, kv, logs}, nil

}

func (action *relayAction) relayVerifyBTCTx(verify *types.RelayVerifyBTC, r *relay) (*types.Receipt, error) {
	orderId := []byte(verify.OrderId)
	order, err := getRelayOrderFromID(orderId, action.db)
	if err != nil {
		return nil, types.ErrTRelayOrderNotExist
	}

	if order.Status == types.RelayOrderStatus_finished {
		return nil, types.ErrTRelayOrderSoldout
	} else if order.Status == types.RelayOrderStatus_canceled {
		return nil, types.ErrTRelayOrderRevoked
	} else if order.Status == types.RelayOrderStatus_pending || order.Status == types.RelayOrderStatus_locking {
		return nil, types.ErrTRelayOrderOnSell
	}

	var receipt *types.Receipt

	err = r.btcStore.verifyBTCTx(verify)
	if err != nil {
		return nil, err
	}

	receipt, err = action.coinsAccount.ExecTransferFrozen(order.CreaterAddr, order.AcceptAddr, action.execAddr, int64(order.Amount))
	if err != nil {
		relaylog.Error("relay verify tx transfer fail", "error", err.Error())
		return nil, err
	}

	order.PreStatus = order.Status
	order.Status = types.RelayOrderStatus_finished
	order.FinishTime = action.blockTime

	relayDb := newRelayDB(order)
	orderKV := relayDb.save(action.db)

	if err != nil {
		return nil, err
	}

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	logs = append(logs, receipt.Logs...)
	logs = append(logs, relayDb.getCreateLogs(types.TyLogRelayCreate))
	logs = append(logs, relayDb.getAcceptLogs(types.TyLogRelayAccept))
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
		logs = append(logs, getBTCHeaderLog(types.TyLogRelayRcvBTCHead, head))
	}

	return &types.Receipt{types.ExecOk, kv, logs}, nil
}
