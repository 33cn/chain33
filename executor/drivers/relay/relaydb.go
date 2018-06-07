package relay

import (
	"time"

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/types"
)

const lockingTime = 6 * time.Hour
const waitBlockHeight = 6

type relayLog struct {
	types.RelayOrder
}

func newRelayLog(order *types.RelayOrder) *relayLog {
	return &relayLog{*order}
}

func (r *relayLog) save(db dbm.KV) []*types.KeyValue {
	set := r.getKVSet()
	for i := 0; i < len(set); i++ {
		db.Set(set[i].GetKey(), set[i].Value)
	}

	return set
}

func (r *relayLog) getKVSet() (kvSet []*types.KeyValue) {
	value := types.Encode(&r.RelayOrder)
	key := []byte(r.Id)
	kvSet = append(kvSet, &types.KeyValue{key, value})
	return kvSet
}

func (r *relayLog) receiptLog(relayLogType int32) *types.ReceiptLog {
	log := &types.ReceiptLog{}
	log.Ty = relayLogType
	base := &types.RelayOrder{
		Id:            r.Id,
		Status:        r.Status,
		PreStatus:     r.PreStatus,
		CreaterAddr:   r.CreaterAddr,
		Amount:        r.Amount,
		CoinOperation: r.CoinOperation,
		Coin:          r.Coin,
		CoinAmount:    r.CoinAmount,
		CoinAddr:      r.CoinAddr,
		CoinTxHash:    r.CoinTxHash,
		CreateTime:    r.CreateTime,
		AcceptAddr:    r.AcceptAddr,
		AcceptTime:    r.AcceptTime,
		ConfirmTime:   r.ConfirmTime,
		FinishTime:    r.FinishTime,
		Height:        r.Height,
	}

	receipt := &types.ReceiptRelayLog{base}
	log.Log = types.Encode(receipt)

	return log
}

type relayDB struct {
	coinsAccount *account.DB
	db           dbm.KV
	txHash       string
	fromAddr     string
	blockTime    int64
	height       int64
	execAddr     string
}

func newRelayDB(r *relay, tx *types.Transaction) *relayDB {
	hash := common.Bytes2Hex(tx.Hash())
	fromAddr := account.PubKeyToAddress(tx.GetSignature().GetPubkey()).String()
	return &relayDB{r.GetCoinsAccount(), r.GetStateDB(), hash,
		fromAddr, r.GetBlockTime(), r.GetHeight(), r.GetAddr()}
}

func (action *relayDB) getOrderByID(orderId []byte) (*types.RelayOrder, error) {
	value, err := action.db.Get(orderId)
	if err != nil {
		return nil, err
	}

	var order types.RelayOrder
	if err = types.Decode(value, &order); err != nil {
		return nil, err
	}
	return &order, nil
}

func (action *relayDB) relayCreate(order *types.RelayCreate) (*types.Receipt, error) {
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
	relayLog := newRelayLog(uOrder)
	sellOrderKV := relayLog.save(action.db)
	logs = append(logs, relayLog.receiptLog(types.TyLogRelayCreate))
	kv = append(kv, sellOrderKV...)

	return &types.Receipt{types.ExecOk, kv, logs}, nil
}

func checkRevokeOrder(order *types.RelayOrder, blockTime int64) error {
	nowTime := time.Now()

	if order.Status == types.RelayOrderStatus_locking {
		acceptTime := time.Unix(order.AcceptTime, 0)
		if nowTime.Sub(acceptTime) < lockingTime {
			relaylog.Error("relay revoke from locking", "current duration", nowTime.Sub(acceptTime), "less", lockingTime)
			return types.ErrTime
		}
	}

	if order.Status == types.RelayOrderStatus_confirming {
		confirmTime := time.Unix(order.ConfirmTime, 0)
		if nowTime.Sub(confirmTime) < 4*lockingTime {
			relaylog.Error("relay revoke from confirming ", "current duration", nowTime.Sub(confirmTime), "less", 4*lockingTime)
			return types.ErrTime
		}
	}

	return nil

}

func (action *relayDB) revokeCreate(revoke *types.RelayRevoke) (*types.Receipt, error) {
	orderId := []byte(revoke.OrderId)
	order, err := action.getOrderByID(orderId)
	if err != nil {
		return nil, types.ErrRelayOrderNotExist
	}

	err = checkRevokeOrder(order, action.blockTime)
	if err != nil {
		return nil, err
	}

	if order.Status == types.RelayOrderStatus_finished {
		return nil, types.ErrRelayOrderSoldout
	} else if order.Status == types.RelayOrderStatus_canceled {
		relaylog.Error("revokeCreate error", "action", revoke.Action)
		return nil, types.ErrRelayOrderRevoked
	}

	if action.fromAddr != order.CreaterAddr {
		return nil, types.ErrRelayReturnAddr
	}

	var receipt *types.Receipt
	if order.CoinOperation == types.RelayCreateBuy {
		receipt, err = action.coinsAccount.ExecActive(order.CreaterAddr, action.execAddr, int64(order.Amount))
		if err != nil {
			relaylog.Error("revoke create", "addrFrom", order.CreaterAddr, "execAddr", action.execAddr, "amount", order.Amount)
			return nil, err
		}
	}

	if order.CoinOperation == types.RelayCreateSell && order.Status != types.RelayOrderStatus_pending {
		receipt, err = action.coinsAccount.ExecActive(order.AcceptAddr, action.execAddr, int64(order.Amount))
		if err != nil {
			relaylog.Error("revoke create", "addrFrom", order.AcceptAddr, "execAddr", action.execAddr, "amount", order.Amount)
			return nil, err
		}
	}

	order.PreStatus = order.Status
	if revoke.Action == types.RelayRevokeUnlockOrder {
		order.Status = types.RelayOrderStatus_pending
	} else {
		order.Status = types.RelayOrderStatus_canceled
	}

	relayLog := newRelayLog(order)
	orderKV := relayLog.save(action.db)

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	if receipt != nil {
		logs = append(logs, receipt.Logs...)
		kv = append(kv, receipt.KV...)
	}
	logs = append(logs, relayLog.receiptLog(types.TyLogRelayRevokeCreate))
	kv = append(kv, orderKV...)
	return &types.Receipt{types.ExecOk, kv, logs}, nil
}

func (action *relayDB) accept(accept *types.RelayAccept) (*types.Receipt, error) {
	orderId := []byte(accept.OrderId)
	order, err := action.getOrderByID(orderId)
	if err != nil {
		return nil, types.ErrRelayOrderNotExist
	}

	if order.Status == types.RelayOrderStatus_canceled {
		return nil, types.ErrRelayOrderRevoked
	}

	if order.Status != types.RelayOrderStatus_pending {
		return nil, types.ErrRelayOrderSoldout
	}

	var receipt *types.Receipt
	if order.CoinOperation == types.RelayCreateSell {
		if accept.CoinAddr == "" {
			relaylog.Error("accept, for sell operation, coinAddr needed")
			return nil, types.ErrRelayOrderParamErr
		}

		order.CoinAddr = accept.CoinAddr

		receipt, err = action.coinsAccount.ExecFrozen(action.fromAddr, action.execAddr, int64(order.Amount))
		if err != nil {
			relaylog.Error("relay accept frozen fail", "addrFrom", action.fromAddr, "execAddr", action.execAddr, "amount", order.Amount)
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

	relayLog := newRelayLog(order)
	sellOrderKV := relayLog.save(action.db)

	logs = append(logs, relayLog.receiptLog(types.TyLogRelayAccept))
	kv = append(kv, sellOrderKV...)

	return &types.Receipt{types.ExecOk, kv, logs}, nil

}

func (action *relayDB) relayRevoke(revoke *types.RelayRevoke) (*types.Receipt, error) {
	if revoke.Target == types.RelayOrderCreate {
		return action.revokeCreate(revoke)
	} else if revoke.Target == types.RelayOrderAccept {
		return action.revokeAccept(revoke)
	}

	return nil, types.ErrRelayOrderParamErr

}

func (action *relayDB) revokeAccept(revoke *types.RelayRevoke) (*types.Receipt, error) {
	orderIdByte := []byte(revoke.OrderId)
	order, err := action.getOrderByID(orderIdByte)
	if err != nil {
		return nil, types.ErrRelayOrderNotExist
	}

	if order.Status == types.RelayOrderStatus_pending || order.Status == types.RelayOrderStatus_canceled {
		return nil, types.ErrRelayOrderRevoked
	} else if order.Status == types.RelayOrderStatus_finished {
		return nil, types.ErrRelayOrderFinished
	}

	err = checkRevokeOrder(order, action.blockTime)
	if err != nil {
		return nil, err
	}

	if action.fromAddr != order.AcceptAddr {
		return nil, types.ErrRelayReturnAddr
	}

	var receipt *types.Receipt
	if order.CoinOperation == types.RelayCreateSell {
		receipt, err = action.coinsAccount.ExecActive(order.AcceptAddr, action.execAddr, int64(order.Amount))
		if err != nil {
			relaylog.Error("revokeAccept", "addrFrom", order.AcceptAddr, "execAddr", action.execAddr, "amount", order.Amount)
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

	relayLog := newRelayLog(order)
	sellOrderKV := relayLog.save(action.db)
	logs = append(logs, relayLog.receiptLog(types.TyLogRelayRevokeAccept))
	kv = append(kv, sellOrderKV...)

	return &types.Receipt{types.ExecOk, kv, logs}, nil
}

func (action *relayDB) confirmTx(confirm *types.RelayConfirmTx) (*types.Receipt, error) {
	orderId := []byte(confirm.OrderId)
	order, err := action.getOrderByID(orderId)
	if err != nil {
		return nil, types.ErrRelayOrderNotExist
	}

	if order.Status == types.RelayOrderStatus_pending {
		return nil, types.ErrRelayOrderOnSell
	} else if order.Status == types.RelayOrderStatus_finished {
		return nil, types.ErrRelayOrderSoldout
	} else if order.Status == types.RelayOrderStatus_canceled {
		return nil, types.ErrRelayOrderRevoked
	}

	var confirmAddr string
	if order.CoinOperation == types.RelayCreateBuy {
		confirmAddr = order.AcceptAddr
	} else {
		confirmAddr = order.CreaterAddr
	}
	if action.fromAddr != confirmAddr {
		relaylog.Error("confirmTx", "fromAddr", action.fromAddr, "not confirmAddr", confirmAddr, "oper", order.CoinOperation)
		return nil, types.ErrRelayReturnAddr
	}

	order.PreStatus = order.Status
	order.Status = types.RelayOrderStatus_confirming
	order.ConfirmTime = action.blockTime
	order.CoinTxHash = confirm.TxHash

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue

	relayLog := newRelayLog(order)
	sellOrderKV := relayLog.save(action.db)
	logs = append(logs, relayLog.receiptLog(types.TyLogRelayConfirmTx))
	kv = append(kv, sellOrderKV...)

	receipt := &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil

}

func (action *relayDB) verifyTx(verify *types.RelayVerify, r *relay) (*types.Receipt, error) {
	orderId := []byte(verify.OrderId)
	order, err := action.getOrderByID(orderId)
	if err != nil {
		return nil, types.ErrRelayOrderNotExist
	}

	if order.Status == types.RelayOrderStatus_finished {
		return nil, types.ErrRelayOrderSoldout
	} else if order.Status == types.RelayOrderStatus_canceled {
		return nil, types.ErrRelayOrderRevoked
	} else if order.Status == types.RelayOrderStatus_pending || order.Status == types.RelayOrderStatus_locking {
		return nil, types.ErrRelayOrderOnSell
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
	order.FinishTxHash = action.txHash

	relayLog := newRelayLog(order)
	orderKV := relayLog.save(action.db)

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	logs = append(logs, receipt.Logs...)
	logs = append(logs, relayLog.receiptLog(types.TyLogRelayFinishTx))
	kv = append(kv, receipt.KV...)
	kv = append(kv, orderKV...)
	return &types.Receipt{types.ExecOk, kv, logs}, nil

}

func (action *relayDB) verifyBtcTx(verify *types.RelayVerifyCli, r *relay) (*types.Receipt, error) {
	orderId := []byte(verify.OrderId)
	order, err := action.getOrderByID(orderId)
	if err != nil {
		return nil, types.ErrRelayOrderNotExist
	}

	if order.Status == types.RelayOrderStatus_finished {
		return nil, types.ErrRelayOrderSoldout
	} else if order.Status == types.RelayOrderStatus_canceled {
		return nil, types.ErrRelayOrderRevoked
	} else if order.Status == types.RelayOrderStatus_pending || order.Status == types.RelayOrderStatus_locking {
		return nil, types.ErrRelayOrderOnSell
	}

	var receipt *types.Receipt

	err = r.btcStore.verifyBtcTx(verify)
	if err != nil {
		return nil, err
	}

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

	relayLog := newRelayLog(order)
	orderKV := relayLog.save(action.db)

	if err != nil {
		return nil, err
	}

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	logs = append(logs, receipt.Logs...)
	logs = append(logs, relayLog.receiptLog(types.TyLogRelayFinishTx))
	kv = append(kv, receipt.KV...)
	kv = append(kv, orderKV...)
	return &types.Receipt{types.ExecOk, kv, logs}, nil

}

func (action *relayDB) saveBtcHeader(headers *types.BtcHeaders) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kv []*types.KeyValue

	// TODO make some simple check
	for _, head := range headers.BtcHeader {
		log := &types.ReceiptLog{}
		log.Ty = types.TyLogRelayRcvBTCHead
		receipt := &types.ReceiptRelayRcvBTCHeaders{head}
		log.Log = types.Encode(receipt)
		logs = append(logs, log)
	}

	return &types.Receipt{types.ExecOk, kv, logs}, nil
}
