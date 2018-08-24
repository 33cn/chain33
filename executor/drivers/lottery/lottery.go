package lottery

import (
	"fmt"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/types"
)

var llog = log.New("module", "execs.lottery")

func Init() {
	drivers.Register(newLottery().GetName(), newLottery, 0)
}

type Lottery struct {
	drivers.DriverBase
}

func newLottery() drivers.Driver {
	l := &Lottery{}
	l.SetChild(l)
	return l
}

func (l *Lottery) GetName() string {
	return types.ExecName(types.LotteryX)
}

func (lott *Lottery) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	var action types.LotteryAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return nil, err
	}
	llog.Debug("Exec Lottery tx=", "tx=", action)
	actiondb := NewLotteryAction(lott, tx)
	if action.Ty == types.LotteryActionCreate && action.GetCreate() != nil {
		llog.Debug("LotteryActionCreate")
		return actiondb.LotteryCreate(action.GetCreate())
	} else if action.Ty == types.LotteryActionBuy && action.GetBuy() != nil {
		llog.Debug("LotteryActionBuy")
		return actiondb.LotteryBuy(action.GetBuy())
	} else if action.Ty == types.LotteryActionShow && action.GetShow() != nil {
		llog.Debug("LotteryActionShow")
		return actiondb.LotteryShow(action.GetShow())
	} else if action.Ty == types.LotteryActionDraw && action.GetDraw() != nil {
		llog.Debug("LotteryActionDraw")
		return actiondb.LotteryDraw(action.GetDraw())
	} else if action.Ty == types.LotteryActionClose && action.GetClose() != nil {
		llog.Debug("LotteryActionClose")
		return actiondb.LotteryClose(action.GetClose())
	}
	//return error
	return nil, types.ErrActionNotSupport
}

func (lott *Lottery) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := lott.DriverBase.ExecLocal(tx, receipt, index)
	if err != nil {
		return nil, err
	}
	if receipt.GetTy() != types.ExecOk {
		return set, nil
	}
	for i := 0; i < len(receipt.Logs); i++ {
		item := receipt.Logs[i]

		if item.Ty == types.TyLogLotteryCreate || item.Ty == types.TyLogLotteryBuy ||
			item.Ty == types.TyLogLotteryShow || item.Ty == types.TyLogLotteryDraw ||
			item.Ty == types.TyLogLotteryClose {
			var lotterylog types.ReceiptLottery
			err := types.Decode(item.Log, &lotterylog)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			kv := lott.saveLottery(&lotterylog)
			set.KV = append(set.KV, kv...)
			if item.Ty == types.TyLogLotteryShow {
				kv := lott.saveLotteryShow(&lotterylog)
				set.KV = append(set.KV, kv...)
			}
		}
	}
	return set, nil
}

func (lott *Lottery) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := lott.DriverBase.ExecDelLocal(tx, receipt, index)
	if err != nil {
		return nil, err
	}
	if receipt.GetTy() != types.ExecOk {
		return set, nil
	}
	for i := 0; i < len(receipt.Logs); i++ {
		item := receipt.Logs[i]

		if item.Ty == types.TyLogLotteryCreate || item.Ty == types.TyLogLotteryBuy ||
			item.Ty == types.TyLogLotteryShow || item.Ty == types.TyLogLotteryDraw ||
			item.Ty == types.TyLogLotteryClose {
			var lotterylog types.ReceiptLottery
			err := types.Decode(item.Log, &lotterylog)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			kv := lott.deleteLottery(&lotterylog)
			set.KV = append(set.KV, kv...)
			if item.Ty == types.TyLogLotteryShow {
				kv := lott.deleteLotteryShow(&lotterylog)
				set.KV = append(set.KV, kv...)
			}
		}
	}
	return set, nil
}

func (lott *Lottery) Query(funcName string, params []byte) (types.Message, error) {
	if funcName == "GetLotteryNormalInfo" {
		var info types.ReqLotteryInfo
		err := types.Decode(params, &info)
		if err != nil {
			return nil, err
		}
		lottery, err := findLottery(lott.GetStateDB(), info.GetLotteryId())
		if err != nil {
			return nil, err
		}
		return &types.ReplyLotteryNormalInfo{lottery.CreateTime,
			lottery.PurchasePeriod,
			lottery.ShowPeriod,
			lottery.MaxPurchaseNum,
			lottery.CreateAddr}, nil
	} else if funcName == "GetLotteryCurrentInfo" {
		var info types.ReqLotteryInfo
		err := types.Decode(params, &info)
		if err != nil {
			return nil, err
		}
		lottery, err := findLottery(lott.GetStateDB(), info.GetLotteryId())
		if err != nil {
			return nil, err
		}
		return &types.ReplyLotteryCurrentInfo{lottery.Status,
			lottery.Fund,
			lottery.LastTransToPurState,
			lottery.LastTransToShowState,
			lottery.TotalPurchasedTxNum,
			lottery.TotalShowedNum,
			lottery.Round}, nil
	} else if funcName == "GetLotteryHistoryLuckyNumber" {
		var info types.ReqLotteryInfo
		err := types.Decode(params, &info)
		if err != nil {
			return nil, err
		}
		lottery, err := findLottery(lott.GetStateDB(), info.GetLotteryId())
		if err != nil {
			return nil, err
		}
		reply := &types.ReplyLotteryHistoryLuckyNumber{}
		reply.LuckyNumber = append(reply.LuckyNumber, lottery.LuckyNumber...)
		return reply, nil
	}

	return nil, types.ErrActionNotSupport
}

func calcLotteryShowKey(lotteryId string, addr string, round int64, number int64, amount int64) []byte {
	key := fmt.Sprintf("lottery-show:%s:%s:%d:%d", addr, lotteryId, round, amount)
	return []byte(key)
}

func (lott *Lottery) saveLotteryShow(lotterylog *types.ReceiptLottery) (kvs []*types.KeyValue) {
	kv := &types.KeyValue{}
	kv.Key = calcLotteryShowKey(lotterylog.LotteryId, lotterylog.Addr, lotterylog.Round, lotterylog.Number, lotterylog.Amount)
	kv.Value = []byte(lotterylog.LotteryId)
	kvs = append(kvs, kv)
	return kvs
}

func (lott *Lottery) deleteLotteryShow(lotterylog *types.ReceiptLottery) (kvs []*types.KeyValue) {
	kv := &types.KeyValue{}
	kv.Key = calcLotteryShowKey(lotterylog.LotteryId, lotterylog.Addr, lotterylog.Round, lotterylog.Number, lotterylog.Amount)
	kv.Value = nil
	kvs = append(kvs, kv)
	return kvs
}

func (lott *Lottery) saveLottery(lotterylog *types.ReceiptLottery) (kvs []*types.KeyValue) {
	if lotterylog.PrevStatus > 0 {
		kv := dellottery(lotterylog.LotteryId, lotterylog.PrevStatus)
		kvs = append(kvs, kv)
	}
	kvs = append(kvs, addlottery(lotterylog.LotteryId, lotterylog.Status))
	return kvs
}

func (lott *Lottery) deleteLottery(lotterylog *types.ReceiptLottery) (kvs []*types.KeyValue) {
	if lotterylog.PrevStatus > 0 {
		kv := addlottery(lotterylog.LotteryId, lotterylog.PrevStatus)
		kvs = append(kvs, kv)
	}
	kvs = append(kvs, dellottery(lotterylog.LotteryId, lotterylog.Status))
	return kvs
}

func calcLotteryKey(lotteryId string, status int32) []byte {
	key := fmt.Sprintf("lottery:%d:%s", status, lotteryId)
	return []byte(key)
}

func addlottery(lotteryId string, status int32) *types.KeyValue {
	kv := &types.KeyValue{}
	kv.Key = calcLotteryKey(lotteryId, status)
	kv.Value = []byte(lotteryId)
	return kv
}

func dellottery(lotteryId string, status int32) *types.KeyValue {
	kv := &types.KeyValue{}
	kv.Key = calcLotteryKey(lotteryId, status)
	kv.Value = nil
	return kv
}
