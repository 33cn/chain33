package executor

import (
	"time"

	"strconv"

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/relay/types"
	"gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
)

const (
	lockingTime   = 12 * time.Hour //as currently one BTC tx may need wait quite long time
	lockBtcHeight = 12 * 6
	lockBtyAmount = 100 * 1e8
)

type relayLog struct {
	ty.RelayOrder
}

func newRelayLog(order *ty.RelayOrder) *relayLog {
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

	if r.CoinTxHash != "" {
		key = []byte(calcCoinHash(r.CoinTxHash))
		kvSet = append(kvSet, &types.KeyValue{key, value})
	}

	return kvSet
}

func (r *relayLog) receiptLog(relayLogType int32) *types.ReceiptLog {
	log := &types.ReceiptLog{}
	log.Ty = relayLogType
	receipt := &ty.ReceiptRelayLog{
		OrderId:       r.Id,
		CurStatus:     r.Status.String(),
		PreStatus:     r.PreStatus.String(),
		CreaterAddr:   r.CreaterAddr,
		TxAmount:      strconv.FormatFloat(float64(r.Amount)/float64(types.Coin), 'f', 4, 64),
		CoinOperation: ty.RelayOrderOperation[r.CoinOperation],
		Coin:          r.Coin,
		CoinAmount:    strconv.FormatFloat(float64(r.CoinAmount)/float64(types.Coin), 'f', 4, 64),
		CoinAddr:      r.CoinAddr,
		CoinTxHash:    r.CoinTxHash,
		CoinWaits:     r.CoinWaits,
		CreateTime:    r.CreateTime,
		AcceptAddr:    r.AcceptAddr,
		AcceptTime:    r.AcceptTime,
		ConfirmTime:   r.ConfirmTime,
		FinishTime:    r.FinishTime,
		CoinHeight:    r.CoinHeight,
	}

	log.Log = types.Encode(receipt)

	return log
}

type relayDB struct {
	coinsAccount *account.DB
	db           dbm.KV
	txHash       []byte
	fromAddr     string
	blockTime    int64
	height       int64
	execAddr     string
	btc          *btcStore
}

func newRelayDB(r *relay, tx *types.Transaction) *relayDB {
	hash := tx.Hash()
	fromAddr := tx.From()
	btc := newBtcStore(r.GetLocalDB())
	return &relayDB{r.GetCoinsAccount(), r.GetStateDB(), hash,
		fromAddr, r.GetBlockTime(), r.GetHeight(), dapp.ExecAddress(r.GetName()), btc}
}

func (action *relayDB) getOrderByID(orderId []byte) (*ty.RelayOrder, error) {
	value, err := action.db.Get(orderId)
	if err != nil {
		return nil, err
	}

	var order ty.RelayOrder
	if err = types.Decode(value, &order); err != nil {
		return nil, err
	}
	return &order, nil
}

func (action *relayDB) getOrderByCoinHash(hash []byte) (*ty.RelayOrder, error) {
	value, err := action.db.Get(hash)
	if err != nil {
		return nil, err
	}

	var order ty.RelayOrder
	if err = types.Decode(value, &order); err != nil {
		return nil, err
	}
	return &order, nil
}

func (action *relayDB) create(order *ty.RelayCreate) (*types.Receipt, error) {
	var receipt *types.Receipt
	var err error
	var coinAddr string
	var coinWaits uint32
	if order.Operation == ty.RelayOrderBuy {
		receipt, err = action.coinsAccount.ExecFrozen(action.fromAddr, action.execAddr, int64(order.BtyAmount))
		if err != nil {
			relaylog.Error("account.ExecFrozen relay ", "addrFrom", action.fromAddr, "execAddr", action.execAddr, "amount", order.BtyAmount)
			return nil, err
		}

		coinAddr = order.Addr
		coinWaits = order.CoinWaits
	} else {
		receipt, err = action.coinsAccount.ExecFrozen(action.fromAddr, action.execAddr, int64(lockBtyAmount))
		if err != nil {
			relaylog.Error("account.ExecFrozen relay ", "addrFrom", action.fromAddr, "execAddr", action.execAddr, "amount", lockBtyAmount)
			return nil, err
		}
	}

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	uOrder := &ty.RelayOrder{
		Id:            calcRelayOrderID(common.ToHex(action.txHash)),
		Status:        ty.RelayOrderStatus_pending,
		PreStatus:     ty.RelayOrderStatus_init,
		Amount:        order.BtyAmount,
		CreaterAddr:   action.fromAddr,
		CoinOperation: order.Operation,
		Coin:          order.Coin,
		CoinAmount:    order.Amount,
		CoinAddr:      coinAddr,
		CoinWaits:     coinWaits,
		CreateTime:    action.blockTime,
		Height:        action.height,
	}

	height, err := action.btc.getLastBtcHeadHeight()
	if err != nil {
		return nil, err
	}
	uOrder.CoinHeight = uint64(height)

	logs = append(logs, receipt.Logs...)
	kv = append(kv, receipt.KV...)

	relayLog := newRelayLog(uOrder)
	sellOrderKV := relayLog.save(action.db)
	logs = append(logs, relayLog.receiptLog(ty.TyLogRelayCreate))
	kv = append(kv, sellOrderKV...)

	return &types.Receipt{types.ExecOk, kv, logs}, nil
}

func (action *relayDB) checkRevokeOrder(order *ty.RelayOrder) error {
	nowTime := time.Unix(action.blockTime, 0)
	var nowBtcHeight, subHeight int64

	nowBtcHeight, err := action.btc.getLastBtcHeadHeight()
	if err != nil {
		return err
	}

	if nowBtcHeight > 0 && order.CoinHeight > 0 && nowBtcHeight > int64(order.CoinHeight) {
		subHeight = nowBtcHeight - int64(order.CoinHeight)
	}

	if order.Status == ty.RelayOrderStatus_locking {
		acceptTime := time.Unix(order.AcceptTime, 0)
		if nowTime.Sub(acceptTime) < lockingTime && subHeight < lockBtcHeight {
			relaylog.Error("relay revoke locking", "duration", nowTime.Sub(acceptTime), "lockingTime", lockingTime, "subHeight", subHeight)
			return ty.ErrRelayBtcTxTimeErr
		}
	}

	if order.Status == ty.RelayOrderStatus_confirming {
		confirmTime := time.Unix(order.ConfirmTime, 0)
		if nowTime.Sub(confirmTime) < 4*lockingTime && subHeight < 4*lockBtcHeight {
			relaylog.Error("relay revoke confirming ", "duration", nowTime.Sub(confirmTime), "confirmTime", 4*lockingTime, "subHeight", subHeight)
			return ty.ErrRelayBtcTxTimeErr
		}
	}

	return nil

}

func (action *relayDB) revokeCreate(revoke *ty.RelayRevoke) (*types.Receipt, error) {
	orderId := []byte(revoke.OrderId)
	order, err := action.getOrderByID(orderId)
	if err != nil {
		return nil, ty.ErrRelayOrderNotExist
	}

	err = action.checkRevokeOrder(order)
	if err != nil {
		return nil, err
	}

	if order.Status == ty.RelayOrderStatus_init {
		return nil, ty.ErrRelayOrderStatusErr
	}

	if order.Status == ty.RelayOrderStatus_pending && revoke.Action == ty.RelayUnlock {
		return nil, ty.ErrRelayOrderParamErr
	}

	if order.Status == ty.RelayOrderStatus_finished {
		return nil, ty.ErrRelayOrderSoldout
	}
	if order.Status == ty.RelayOrderStatus_canceled {
		return nil, ty.ErrRelayOrderRevoked
	}

	if action.fromAddr != order.CreaterAddr {
		return nil, ty.ErrRelayReturnAddr
	}

	var receipt *types.Receipt
	var receiptTransfer *types.Receipt
	if order.CoinOperation == ty.RelayOrderBuy {
		receipt, err = action.coinsAccount.ExecActive(order.CreaterAddr, action.execAddr, int64(order.Amount))
		if err != nil {
			relaylog.Error("revoke create", "addrFrom", order.CreaterAddr, "execAddr", action.execAddr, "amount", order.Amount)
			return nil, err
		}
	} else if order.Status != ty.RelayOrderStatus_pending {
		receipt, err = action.coinsAccount.ExecActive(order.AcceptAddr, action.execAddr, int64(order.Amount))
		if err != nil {
			relaylog.Error("revoke create", "addrFrom", order.AcceptAddr, "execAddr", action.execAddr, "amount", order.Amount)
			return nil, err
		}

		receiptTransfer, err = action.coinsAccount.ExecTransferFrozen(order.CreaterAddr, order.AcceptAddr, action.execAddr, int64(lockBtyAmount))
		if err != nil {
			relaylog.Error("revokeAccept", "from", order.AcceptAddr, "to", order.CreaterAddr, "execAddr", action.execAddr, "amount", lockBtyAmount)
			return nil, err
		}
	}

	order.PreStatus = order.Status
	if revoke.Action == ty.RelayUnlock {
		order.Status = ty.RelayOrderStatus_pending
	} else {
		order.Status = ty.RelayOrderStatus_canceled
	}

	relayLog := newRelayLog(order)
	orderKV := relayLog.save(action.db)

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	if receipt != nil {
		logs = append(logs, receipt.Logs...)
		kv = append(kv, receipt.KV...)
	}
	if receiptTransfer != nil {
		logs = append(logs, receiptTransfer.Logs...)
		kv = append(kv, receiptTransfer.KV...)
	}
	logs = append(logs, relayLog.receiptLog(ty.TyLogRelayRevokeCreate))
	kv = append(kv, orderKV...)
	return &types.Receipt{types.ExecOk, kv, logs}, nil
}

func (action *relayDB) accept(accept *ty.RelayAccept) (*types.Receipt, error) {
	orderId := []byte(accept.OrderId)
	order, err := action.getOrderByID(orderId)
	if err != nil {
		return nil, ty.ErrRelayOrderNotExist
	}

	if order.Status == ty.RelayOrderStatus_canceled {
		return nil, ty.ErrRelayOrderRevoked
	}
	if order.Status != ty.RelayOrderStatus_pending {
		return nil, ty.ErrRelayOrderSoldout
	}

	var receipt *types.Receipt

	if order.CoinOperation == ty.RelayOrderBuy {
		receipt, err = action.coinsAccount.ExecFrozen(action.fromAddr, action.execAddr, int64(lockBtyAmount))
		if err != nil {
			relaylog.Error("relay accept frozen fail ", "addrFrom", action.fromAddr, "execAddr", action.execAddr, "amount", lockBtyAmount)
			return nil, err
		}

	} else {
		if accept.CoinAddr == "" {
			relaylog.Error("accept, for sell operation, coinAddr needed")
			return nil, ty.ErrRelayOrderParamErr
		}

		order.CoinAddr = accept.CoinAddr
		order.CoinWaits = accept.CoinWaits

		receipt, err = action.coinsAccount.ExecFrozen(action.fromAddr, action.execAddr, int64(order.Amount))
		if err != nil {
			relaylog.Error("relay accept frozen fail", "addrFrom", action.fromAddr, "execAddr", action.execAddr, "amount", order.Amount)
			return nil, err
		}
	}

	order.PreStatus = order.Status
	order.Status = ty.RelayOrderStatus_locking
	order.AcceptAddr = action.fromAddr
	order.AcceptTime = action.blockTime

	height, err := action.btc.getLastBtcHeadHeight()
	if err != nil {
		return nil, err
	}
	order.CoinHeight = uint64(height)

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue

	logs = append(logs, receipt.Logs...)
	kv = append(kv, receipt.KV...)

	relayLog := newRelayLog(order)
	sellOrderKV := relayLog.save(action.db)

	logs = append(logs, relayLog.receiptLog(ty.TyLogRelayAccept))
	kv = append(kv, sellOrderKV...)

	return &types.Receipt{types.ExecOk, kv, logs}, nil

}

func (action *relayDB) relayRevoke(revoke *ty.RelayRevoke) (*types.Receipt, error) {
	if revoke.Target == ty.RelayRevokeCreate {
		return action.revokeCreate(revoke)
	}

	return action.revokeAccept(revoke)
}

func (action *relayDB) revokeAccept(revoke *ty.RelayRevoke) (*types.Receipt, error) {
	orderIdByte := []byte(revoke.OrderId)
	order, err := action.getOrderByID(orderIdByte)
	if err != nil {
		return nil, ty.ErrRelayOrderNotExist
	}

	if order.Status == ty.RelayOrderStatus_pending || order.Status == ty.RelayOrderStatus_canceled {
		return nil, ty.ErrRelayOrderRevoked
	}
	if order.Status == ty.RelayOrderStatus_finished {
		return nil, ty.ErrRelayOrderFinished
	}

	err = action.checkRevokeOrder(order)
	if err != nil {
		return nil, err
	}

	if action.fromAddr != order.AcceptAddr {
		return nil, ty.ErrRelayReturnAddr
	}

	var receipt *types.Receipt
	if order.CoinOperation == ty.RelayOrderSell {
		receipt, err = action.coinsAccount.ExecActive(order.AcceptAddr, action.execAddr, int64(order.Amount))
		if err != nil {
			relaylog.Error("revokeAccept", "addrFrom", order.AcceptAddr, "execAddr", action.execAddr, "amount", order.Amount)
			return nil, err
		}
		order.CoinAddr = ""
	}

	var receiptTransfer *types.Receipt
	if order.CoinOperation == ty.RelayOrderBuy {
		receiptTransfer, err = action.coinsAccount.ExecTransferFrozen(order.AcceptAddr, order.CreaterAddr, action.execAddr, int64(lockBtyAmount))
		if err != nil {
			relaylog.Error("revokeAccept", "from", order.AcceptAddr, "to", order.CreaterAddr, "execAddr", action.execAddr, "amount", lockBtyAmount)
			return nil, err
		}
	}

	order.PreStatus = order.Status
	order.Status = ty.RelayOrderStatus_pending
	order.AcceptAddr = ""
	order.AcceptTime = 0

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	if receipt != nil {
		logs = append(logs, receipt.Logs...)
		kv = append(kv, receipt.KV...)
	}
	if receiptTransfer != nil {
		logs = append(logs, receiptTransfer.Logs...)
		kv = append(kv, receiptTransfer.KV...)
	}
	relayLog := newRelayLog(order)
	sellOrderKV := relayLog.save(action.db)
	logs = append(logs, relayLog.receiptLog(ty.TyLogRelayRevokeAccept))
	kv = append(kv, sellOrderKV...)

	return &types.Receipt{types.ExecOk, kv, logs}, nil
}

func (action *relayDB) confirmTx(confirm *ty.RelayConfirmTx) (*types.Receipt, error) {
	orderId := []byte(confirm.OrderId)
	order, err := action.getOrderByID(orderId)
	if err != nil {
		return nil, ty.ErrRelayOrderNotExist
	}

	if order.Status == ty.RelayOrderStatus_pending {
		return nil, ty.ErrRelayOrderOnSell
	}
	if order.Status == ty.RelayOrderStatus_finished {
		return nil, ty.ErrRelayOrderSoldout
	}
	if order.Status == ty.RelayOrderStatus_canceled {
		return nil, ty.ErrRelayOrderRevoked
	}

	//report Error if coinTxHash has been used and not same orderId, if same orderId, means to modify the txHash
	coinTxOrder, err := action.getOrderByCoinHash([]byte(calcCoinHash(confirm.TxHash)))
	if coinTxOrder != nil {
		if coinTxOrder.Id != confirm.OrderId {
			relaylog.Error("confirmTx", "coinTxHash", confirm.TxHash, "has been used in other order", coinTxOrder.Id)
			return nil, ty.ErrRelayCoinTxHashUsed
		}
	}

	var confirmAddr string
	if order.CoinOperation == ty.RelayOrderBuy {
		confirmAddr = order.AcceptAddr
	} else {
		confirmAddr = order.CreaterAddr
	}
	if action.fromAddr != confirmAddr {
		return nil, ty.ErrRelayReturnAddr
	}

	order.PreStatus = order.Status
	order.Status = ty.RelayOrderStatus_confirming
	order.ConfirmTime = action.blockTime
	order.CoinTxHash = confirm.TxHash
	height, err := action.btc.getLastBtcHeadHeight()
	if err != nil {
		relaylog.Error("confirmTx Get Last BTC", "orderid", confirm.OrderId)
		return nil, err
	}
	order.CoinHeight = uint64(height)

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue

	relayLog := newRelayLog(order)
	sellOrderKV := relayLog.save(action.db)
	logs = append(logs, relayLog.receiptLog(ty.TyLogRelayConfirmTx))
	kv = append(kv, sellOrderKV...)

	receipt := &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil

}

func (action *relayDB) verifyTx(verify *ty.RelayVerify) (*types.Receipt, error) {
	orderId := []byte(verify.OrderId)
	order, err := action.getOrderByID(orderId)
	if err != nil {
		return nil, ty.ErrRelayOrderNotExist
	}

	if order.Status == ty.RelayOrderStatus_finished {
		return nil, ty.ErrRelayOrderSoldout
	}
	if order.Status == ty.RelayOrderStatus_canceled {
		return nil, ty.ErrRelayOrderRevoked
	}
	if order.Status == ty.RelayOrderStatus_pending || order.Status == ty.RelayOrderStatus_locking {
		return nil, ty.ErrRelayOrderOnSell
	}

	err = action.btc.verifyBtcTx(verify, order)
	if err != nil {
		return nil, err
	}

	var receipt *types.Receipt
	if order.CoinOperation == ty.RelayOrderBuy {
		receipt, err = action.coinsAccount.ExecTransferFrozen(order.CreaterAddr, order.AcceptAddr, action.execAddr, int64(order.Amount))
		if err != nil {
			relaylog.Error("verify buy transfer fail", "error", err.Error())
			return nil, err
		}

	} else {
		receipt, err = action.coinsAccount.ExecTransferFrozen(order.AcceptAddr, order.CreaterAddr, action.execAddr, int64(order.Amount))
		if err != nil {
			relaylog.Error("verify sell transfer fail", "error", err.Error())
			return nil, err
		}
	}

	var receiptTransfer *types.Receipt
	if order.CoinOperation == ty.RelayOrderBuy {
		receiptTransfer, err = action.coinsAccount.ExecActive(order.AcceptAddr, action.execAddr, int64(lockBtyAmount))
		if err != nil {
			relaylog.Error("verify exec active", "from", order.AcceptAddr, "amount", lockBtyAmount)
			return nil, err
		}

	} else {
		receiptTransfer, err = action.coinsAccount.ExecActive(order.CreaterAddr, action.execAddr, int64(lockBtyAmount))
		if err != nil {
			relaylog.Error("verify exec active", "from", order.CreaterAddr, "amount", lockBtyAmount)
			return nil, err
		}
	}

	order.PreStatus = order.Status
	order.Status = ty.RelayOrderStatus_finished
	order.FinishTime = action.blockTime
	order.FinishTxHash = common.ToHex(action.txHash)

	relayLog := newRelayLog(order)
	orderKV := relayLog.save(action.db)

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	logs = append(logs, receipt.Logs...)
	logs = append(logs, receiptTransfer.Logs...)
	logs = append(logs, relayLog.receiptLog(ty.TyLogRelayFinishTx))
	kv = append(kv, receipt.KV...)
	kv = append(kv, receiptTransfer.KV...)
	kv = append(kv, orderKV...)
	return &types.Receipt{types.ExecOk, kv, logs}, nil

}

func (action *relayDB) verifyCmdTx(verify *ty.RelayVerifyCli) (*types.Receipt, error) {
	orderId := []byte(verify.OrderId)
	order, err := action.getOrderByID(orderId)
	if err != nil {
		return nil, ty.ErrRelayOrderNotExist
	}

	if order.Status == ty.RelayOrderStatus_finished {
		return nil, ty.ErrRelayOrderSoldout
	}
	if order.Status == ty.RelayOrderStatus_canceled {
		return nil, ty.ErrRelayOrderRevoked
	}
	if order.Status == ty.RelayOrderStatus_pending || order.Status == ty.RelayOrderStatus_locking {
		return nil, ty.ErrRelayOrderOnSell
	}

	var receipt *types.Receipt

	err = action.btc.verifyCmdBtcTx(verify)
	if err != nil {
		return nil, err
	}

	if order.CoinOperation == ty.RelayOrderBuy {
		receipt, err = action.coinsAccount.ExecTransferFrozen(order.CreaterAddr, order.AcceptAddr, action.execAddr, int64(order.Amount))

	} else {
		receipt, err = action.coinsAccount.ExecTransferFrozen(order.AcceptAddr, order.CreaterAddr, action.execAddr, int64(order.Amount))
	}

	if err != nil {
		relaylog.Error("relay verify tx transfer fail", "error", err.Error())
		return nil, err
	}

	var receiptTransfer *types.Receipt
	if order.CoinOperation == ty.RelayOrderBuy {
		receiptTransfer, err = action.coinsAccount.ExecActive(order.AcceptAddr, action.execAddr, int64(lockBtyAmount))
		if err != nil {
			relaylog.Error("verify exec active", "from", order.AcceptAddr, "amount", lockBtyAmount)
			return nil, err
		}

	} else {
		receiptTransfer, err = action.coinsAccount.ExecActive(order.CreaterAddr, action.execAddr, int64(lockBtyAmount))
		if err != nil {
			relaylog.Error("verify exec active", "from", order.CreaterAddr, "amount", lockBtyAmount)
			return nil, err
		}
	}

	order.PreStatus = order.Status
	order.Status = ty.RelayOrderStatus_finished
	order.FinishTime = action.blockTime

	relayLog := newRelayLog(order)
	orderKV := relayLog.save(action.db)

	if err != nil {
		return nil, err
	}

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	logs = append(logs, receipt.Logs...)
	logs = append(logs, receiptTransfer.Logs...)
	logs = append(logs, relayLog.receiptLog(ty.TyLogRelayFinishTx))
	kv = append(kv, receipt.KV...)
	kv = append(kv, receiptTransfer.KV...)
	kv = append(kv, orderKV...)
	return &types.Receipt{types.ExecOk, kv, logs}, nil

}

func saveBtcLastHead(db dbm.KV, head *ty.RelayLastRcvBtcHeader) (set []*types.KeyValue) {
	if head == nil || head.Header == nil {
		return nil
	}

	value := types.Encode(head)
	key := []byte(btcLastHead)
	set = append(set, &types.KeyValue{key, value})

	for i := 0; i < len(set); i++ {
		db.Set(set[i].GetKey(), set[i].Value)
	}
	return set
}

func getBtcLastHead(db dbm.KV) (*ty.RelayLastRcvBtcHeader, error) {
	value, err := db.Get([]byte(btcLastHead))
	if err != nil {
		return nil, err
	}
	var head ty.RelayLastRcvBtcHeader
	if err = types.Decode(value, &head); err != nil {
		return nil, err
	}

	return &head, nil
}

func (action *relayDB) saveBtcHeader(headers *ty.BtcHeaders, localDb dbm.KVDB) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	var preHead = &ty.RelayLastRcvBtcHeader{}
	var receipt = &ty.ReceiptRelayRcvBTCHeaders{}

	if action.fromAddr != subconfig.GStr("genesis") {
		return nil, types.ErrFromAddr
	}

	lastHead, err := getBtcLastHead(action.db)
	if err != nil && err != types.ErrNotFound {
		return nil, err
	}

	if lastHead != nil {
		preHead.Header = lastHead.Header
		preHead.BaseHeight = lastHead.BaseHeight

		receipt.LastHeight = lastHead.Header.Height
		receipt.LastBaseHeight = lastHead.BaseHeight
	}

	log := &types.ReceiptLog{}
	log.Ty = ty.TyLogRelayRcvBTCHead

	for _, head := range headers.BtcHeader {
		err := verifyBlockHeader(head, preHead, localDb)
		if err != nil {
			return nil, err
		}

		preHead.Header = head
		if head.IsReset {
			preHead.BaseHeight = head.Height
		}
		receipt.Headers = append(receipt.Headers, head)
	}

	receipt.NewHeight = preHead.Header.Height
	receipt.NewBaseHeight = preHead.BaseHeight

	log.Log = types.Encode(receipt)
	logs = append(logs, log)
	kv = saveBtcLastHead(action.db, preHead)
	return &types.Receipt{types.ExecOk, kv, logs}, nil
}
