package relay

import (
	"time"

	"strconv"

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/types"
)

const (
	lockingTime     = 6 * time.Hour
	lockBtcHeight   = 6 * 6
	waitBlockHeight = 6
)

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

	if r.CoinTxHash != "" {
		key = []byte(calcCoinHash(r.CoinTxHash))
		kvSet = append(kvSet, &types.KeyValue{key, value})
	}

	return kvSet
}

func (r *relayLog) receiptLog(relayLogType int32) *types.ReceiptLog {
	log := &types.ReceiptLog{}
	log.Ty = relayLogType
	receipt := &types.ReceiptRelayLog{
		OrderId:       r.Id,
		CurStatus:     r.Status.String(),
		PreStatus:     r.PreStatus.String(),
		CreaterAddr:   r.CreaterAddr,
		TxAmount:      strconv.FormatFloat(float64(r.Amount)/float64(types.Coin), 'f', 4, 64),
		CoinOperation: types.RelayOrderOperation[r.CoinOperation],
		Coin:          r.Coin,
		CoinAmount:    strconv.FormatFloat(float64(r.CoinAmount)/float64(types.Coin), 'f', 4, 64),
		CoinAddr:      r.CoinAddr,
		CoinTxHash:    r.CoinTxHash,
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
}

func newRelayDB(r *relay, tx *types.Transaction) *relayDB {
	hash := tx.Hash()
	fromAddr := tx.From()
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

func (action *relayDB) getOrderByCoinHash(hash []byte) (*types.RelayOrder, error) {
	value, err := action.db.Get(hash)
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
	if order.Operation == types.RelayOrderBuy {
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
		Id:            calcRelayOrderID(common.ToHex(action.txHash)),
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

func checkRevokeOrder(order *types.RelayOrder, btc *btcStore) error {
	nowTime := time.Now()
	var nowBtcHeight, subHeight int64

	nowBtcHeight, err := btc.getLastBtcHeadHeight()
	if err != nil {
		return err
	}

	if nowBtcHeight > 0 && order.CoinHeight > 0 && nowBtcHeight > int64(order.CoinHeight) {
		subHeight = nowBtcHeight - int64(order.CoinHeight)
	}

	if order.Status == types.RelayOrderStatus_locking {
		acceptTime := time.Unix(order.AcceptTime, 0)
		if nowTime.Sub(acceptTime) < lockingTime && subHeight < lockBtcHeight {
			relaylog.Error("relay revoke locking", "duration", nowTime.Sub(acceptTime), "lockingTime", lockingTime, "subHeight", subHeight)
			return types.ErrRelayBtcTxTimeErr
		}
	}

	if order.Status == types.RelayOrderStatus_confirming {
		confirmTime := time.Unix(order.ConfirmTime, 0)
		if nowTime.Sub(confirmTime) < 4*lockingTime && subHeight < 4*lockBtcHeight {
			relaylog.Error("relay revoke confirming ", "duration", nowTime.Sub(confirmTime), "confirmTime", 4*lockingTime, "subHeight", subHeight)
			return types.ErrRelayBtcTxTimeErr
		}
	}

	return nil

}

func (action *relayDB) revokeCreate(btc *btcStore, revoke *types.RelayRevoke) (*types.Receipt, error) {
	orderId := []byte(revoke.OrderId)
	order, err := action.getOrderByID(orderId)
	if err != nil {
		return nil, types.ErrRelayOrderNotExist
	}

	err = checkRevokeOrder(order, btc)
	if err != nil {
		return nil, err
	}

	if order.Status == types.RelayOrderStatus_finished {
		return nil, types.ErrRelayOrderSoldout
	}
	if order.Status == types.RelayOrderStatus_canceled {
		return nil, types.ErrRelayOrderRevoked
	}

	if action.fromAddr != order.CreaterAddr {
		return nil, types.ErrRelayReturnAddr
	}

	var receipt *types.Receipt
	if order.CoinOperation == types.RelayOrderBuy {
		receipt, err = action.coinsAccount.ExecActive(order.CreaterAddr, action.execAddr, int64(order.Amount))
		if err != nil {
			relaylog.Error("revoke create", "addrFrom", order.CreaterAddr, "execAddr", action.execAddr, "amount", order.Amount)
			return nil, err
		}
	}

	if order.CoinOperation == types.RelayOrderSell && order.Status != types.RelayOrderStatus_pending {
		receipt, err = action.coinsAccount.ExecActive(order.AcceptAddr, action.execAddr, int64(order.Amount))
		if err != nil {
			relaylog.Error("revoke create", "addrFrom", order.AcceptAddr, "execAddr", action.execAddr, "amount", order.Amount)
			return nil, err
		}
	}

	order.PreStatus = order.Status
	if revoke.Action == types.RelayUnlock {
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

func (action *relayDB) accept(btc *btcStore, accept *types.RelayAccept) (*types.Receipt, error) {
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
	if order.CoinOperation == types.RelayOrderSell {
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

	height, err := btc.getLastBtcHeadHeight()
	if err != nil {
		return nil, err
	}
	order.CoinHeight = uint64(height)

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

func (action *relayDB) relayRevoke(btc *btcStore, revoke *types.RelayRevoke) (*types.Receipt, error) {
	if revoke.Target == types.RelayRevokeCreate {
		return action.revokeCreate(btc, revoke)
	}

	return action.revokeAccept(btc, revoke)
}

func (action *relayDB) revokeAccept(btc *btcStore, revoke *types.RelayRevoke) (*types.Receipt, error) {
	orderIdByte := []byte(revoke.OrderId)
	order, err := action.getOrderByID(orderIdByte)
	if err != nil {
		return nil, types.ErrRelayOrderNotExist
	}

	if order.Status == types.RelayOrderStatus_pending || order.Status == types.RelayOrderStatus_canceled {
		return nil, types.ErrRelayOrderRevoked
	}
	if order.Status == types.RelayOrderStatus_finished {
		return nil, types.ErrRelayOrderFinished
	}

	err = checkRevokeOrder(order, btc)
	if err != nil {
		return nil, err
	}

	if action.fromAddr != order.AcceptAddr {
		return nil, types.ErrRelayReturnAddr
	}

	var receipt *types.Receipt
	if order.CoinOperation == types.RelayOrderSell {
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

func (action *relayDB) confirmTx(btc *btcStore, confirm *types.RelayConfirmTx) (*types.Receipt, error) {
	orderId := []byte(confirm.OrderId)
	order, err := action.getOrderByID(orderId)
	if err != nil {
		return nil, types.ErrRelayOrderNotExist
	}

	if order.Status == types.RelayOrderStatus_pending {
		return nil, types.ErrRelayOrderOnSell
	}
	if order.Status == types.RelayOrderStatus_finished {
		return nil, types.ErrRelayOrderSoldout
	}
	if order.Status == types.RelayOrderStatus_canceled {
		return nil, types.ErrRelayOrderRevoked
	}

	//report Error if coinTxHash has been used and not same orderId, if same orderId, means to modify the txHash
	coinTxOrder, err := action.getOrderByCoinHash([]byte(calcCoinHash(confirm.TxHash)))
	if coinTxOrder != nil {
		if coinTxOrder.Id != confirm.OrderId {
			relaylog.Error("confirmTx", "coinTxHash", confirm.TxHash, "has been used in other order", coinTxOrder.Id)
			return nil, types.ErrRelayCoinTxHashUsed
		}
	}

	var confirmAddr string
	if order.CoinOperation == types.RelayOrderBuy {
		confirmAddr = order.AcceptAddr
	} else {
		confirmAddr = order.CreaterAddr
	}
	if action.fromAddr != confirmAddr {
		return nil, types.ErrRelayReturnAddr
	}

	order.PreStatus = order.Status
	order.Status = types.RelayOrderStatus_confirming
	order.ConfirmTime = action.blockTime
	order.CoinTxHash = confirm.TxHash
	height, err := btc.getLastBtcHeadHeight()
	if err != nil {
		relaylog.Error("confirmTx Get Last BTC", "orderid", confirm.OrderId)
		return nil, err
	}
	order.CoinHeight = uint64(height)

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue

	relayLog := newRelayLog(order)
	sellOrderKV := relayLog.save(action.db)
	logs = append(logs, relayLog.receiptLog(types.TyLogRelayConfirmTx))
	kv = append(kv, sellOrderKV...)

	receipt := &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil

}

func (action *relayDB) verifyTx(btc *btcStore, verify *types.RelayVerify) (*types.Receipt, error) {
	orderId := []byte(verify.OrderId)
	order, err := action.getOrderByID(orderId)
	if err != nil {
		return nil, types.ErrRelayOrderNotExist
	}

	if order.Status == types.RelayOrderStatus_finished {
		return nil, types.ErrRelayOrderSoldout
	}
	if order.Status == types.RelayOrderStatus_canceled {
		return nil, types.ErrRelayOrderRevoked
	}
	if order.Status == types.RelayOrderStatus_pending || order.Status == types.RelayOrderStatus_locking {
		return nil, types.ErrRelayOrderOnSell
	}

	err = btc.verifyBtcTx(verify, order)
	if err != nil {
		return nil, err
	}

	var receipt *types.Receipt
	if order.CoinOperation == types.RelayOrderBuy {
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
	order.FinishTxHash = common.ToHex(action.txHash)

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

func (action *relayDB) verifyCmdTx(btc *btcStore, verify *types.RelayVerifyCli) (*types.Receipt, error) {
	orderId := []byte(verify.OrderId)
	order, err := action.getOrderByID(orderId)
	if err != nil {
		return nil, types.ErrRelayOrderNotExist
	}

	if order.Status == types.RelayOrderStatus_finished {
		return nil, types.ErrRelayOrderSoldout
	}
	if order.Status == types.RelayOrderStatus_canceled {
		return nil, types.ErrRelayOrderRevoked
	}
	if order.Status == types.RelayOrderStatus_pending || order.Status == types.RelayOrderStatus_locking {
		return nil, types.ErrRelayOrderOnSell
	}

	var receipt *types.Receipt

	err = btc.verifyCmdBtcTx(verify)
	if err != nil {
		return nil, err
	}

	if order.CoinOperation == types.RelayOrderBuy {
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

func saveBtcLastHead(db dbm.KV, head *types.RelayLastRcvBtcHeader) (set []*types.KeyValue) {
	if head == nil || head.Header == nil {
		return nil
	}

	value := types.Encode(head.Header)
	key := []byte(btcLastHead)
	set = append(set, &types.KeyValue{key, value})

	value = types.Encode(&types.Int64{int64(head.BaseHeight)})
	key = []byte(btcLastBaseHeight)
	set = append(set, &types.KeyValue{key, value})

	for i := 0; i < len(set); i++ {
		db.Set(set[i].GetKey(), set[i].Value)
	}
	return set
}

func getBtcLastHead(db dbm.KV) (*types.RelayLastRcvBtcHeader, error) {
	var lastHead types.RelayLastRcvBtcHeader

	value, err := db.Get([]byte(btcLastHead))
	if err != nil {
		return nil, err
	}
	var head types.BtcHeader
	if err = types.Decode(value, &head); err != nil {
		return nil, err
	}
	lastHead.Header = &head

	value, err = db.Get([]byte(btcLastBaseHeight))
	if err != nil {
		return nil, err
	}
	height, err := decodeHeight(value)
	if err != nil {
		return nil, err
	}
	lastHead.BaseHeight = uint64(height)

	return &lastHead, nil
}

func (action *relayDB) saveBtcHeader(headers *types.BtcHeaders) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	var preHead = &types.RelayLastRcvBtcHeader{}
	var receipt = &types.ReceiptRelayRcvBTCHeaders{}

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
	log.Ty = types.TyLogRelayRcvBTCHead

	for _, head := range headers.BtcHeader {
		err := verifyBlockHeader(head, preHead)
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
