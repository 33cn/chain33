package executor

import (
	"fmt"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common"
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
)

var llog = log.New("module", "execs.lottery")

func Init(name string) {
	drivers.Register(GetName(), newLottery, 0)
}

func GetName() string {
	return newLottery().GetName()
}

type Lottery struct {
	drivers.DriverBase
}

func newLottery() drivers.Driver {
	l := &Lottery{}
	l.SetChild(l)
	return l
}

func (l *Lottery) GetDriverName() string {
	return types.LotteryX
}

func (lott *Lottery) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	var action types.LotteryAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return nil, err
	}
	llog.Debug("Exec Lottery tx=", "tx=", action)
	actiondb := NewLotteryAction(lott, tx)
	defer actiondb.conn.Close()

	if action.Ty == types.LotteryActionCreate && action.GetCreate() != nil {
		llog.Debug("LotteryActionCreate")
		return actiondb.LotteryCreate(action.GetCreate())
	} else if action.Ty == types.LotteryActionBuy && action.GetBuy() != nil {
		llog.Debug("LotteryActionBuy")
		return actiondb.LotteryBuy(action.GetBuy())
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
			item.Ty == types.TyLogLotteryDraw || item.Ty == types.TyLogLotteryClose {
			var lotterylog types.ReceiptLottery
			err := types.Decode(item.Log, &lotterylog)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			kv := lott.saveLottery(&lotterylog)
			set.KV = append(set.KV, kv...)

			if item.Ty == types.TyLogLotteryBuy {
				kv := lott.saveLotteryBuy(&lotterylog, common.ToHex(tx.Hash()))
				set.KV = append(set.KV, kv...)
			} else if item.Ty == types.TyLogLotteryDraw {
				kv := lott.saveLotteryDraw(&lotterylog)
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
			item.Ty == types.TyLogLotteryDraw || item.Ty == types.TyLogLotteryClose {
			var lotterylog types.ReceiptLottery
			err := types.Decode(item.Log, &lotterylog)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			kv := lott.deleteLottery(&lotterylog)
			set.KV = append(set.KV, kv...)

			if item.Ty == types.TyLogLotteryBuy {
				kv := lott.deleteLotteryBuy(&lotterylog, common.ToHex(tx.Hash()))
				set.KV = append(set.KV, kv...)
			} else if item.Ty == types.TyLogLotteryDraw {
				kv := lott.deleteLotteryDraw(&lotterylog)
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
		return &types.ReplyLotteryNormalInfo{lottery.CreateHeight,
			lottery.PurBlockNum,
			lottery.DrawBlockNum,
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
			lottery.LastTransToDrawState,
			lottery.TotalPurchasedTxNum,
			lottery.Round,
			lottery.LuckyNumber,
			lottery.LastTransToPurStateOnMain,
			lottery.LastTransToDrawStateOnMain}, nil
	} else if funcName == "GetLotteryHistoryLuckyNumber" {
		var req types.ReqLotteryLuckyHistory
		err := types.Decode(params, &req)
		if err != nil {
			return nil, err
		}
		return ListLotteryLuckyHistory(lott.GetLocalDB(), lott.GetStateDB(), &req)
	} else if funcName == "GetLotteryRoundLuckyNumber" {
		var req types.ReqLotteryLuckyInfo
		err := types.Decode(params, &req)
		if err != nil {
			return nil, err
		}

		key := calcLotteryDrawKey(req.LotteryId, req.Round)
		record, err := lott.findLotteryDrawRecord(key)
		if err != nil {
			return nil, err
		}
		return record, nil
	} else if funcName == "GetLotteryHistoryBuyInfo" {
		var req types.ReqLotteryBuyHistory
		err := types.Decode(params, &req)
		if err != nil {
			return nil, err
		}
		return ListLotteryBuyRecords(lott.GetLocalDB(), lott.GetStateDB(), &req)
	} else if funcName == "GetLotteryBuyRoundInfo" {
		var req types.ReqLotteryBuyInfo
		err := types.Decode(params, &req)
		if err != nil {
			return nil, err
		}

		key := calcLotteryBuyRoundPrefix(req.LotteryId, req.Addr, req.Round)
		record, err := lott.findLotteryBuyRecord(key)
		if err != nil {
			return nil, err
		}
		return record, nil
	}
	return nil, types.ErrActionNotSupport
}

func calcLotteryBuyPrefix(lotteryId string, addr string) []byte {
	key := fmt.Sprintf("lottery-buy:%s:%s", lotteryId, addr)
	return []byte(key)
}

func calcLotteryBuyRoundPrefix(lotteryId string, addr string, round int64) []byte {
	key := fmt.Sprintf("lottery-buy:%s:%s:%10d", lotteryId, addr, round)
	return []byte(key)
}

func calcLotteryBuyKey(lotteryId string, addr string, round int64, txId string) []byte {
	key := fmt.Sprintf("lottery-buy:%s:%s:%10d:%s", lotteryId, addr, round, txId)
	return []byte(key)
}

func calcLotteryDrawPrefix(lotteryId string) []byte {
	key := fmt.Sprintf("lottery-draw:%s", lotteryId)
	return []byte(key)
}

func calcLotteryDrawKey(lotteryId string, round int64) []byte {
	key := fmt.Sprintf("lottery-draw:%s:%10d", lotteryId, round)
	return []byte(key)
}

func (lott *Lottery) findLotteryBuyRecord(key []byte) (*types.LotteryBuyRecords, error) {

	count := lott.GetLocalDB().PrefixCount(key)
	llog.Error("findLotteryBuyRecord", "count", count)

	values, err := lott.GetLocalDB().List(key, nil, int32(count), 0)
	if err != nil {
		return nil, err
	}
	var records types.LotteryBuyRecords

	for _, value := range values {
		var record types.LotteryBuyRecord
		err := types.Decode(value, &record)
		if err != nil {
			continue
		}
		records.Records = append(records.Records, &record)
	}

	return &records, nil
}

func (lott *Lottery) findLotteryDrawRecord(key []byte) (*types.LotteryDrawRecord, error) {
	value, err := lott.GetLocalDB().Get(key)
	if err != nil && err != types.ErrNotFound {
		llog.Error("findLotteryDrawRecord", "err", err)
		return nil, err
	}
	if err == types.ErrNotFound {
		return nil, nil
	}
	var record types.LotteryDrawRecord

	err = types.Decode(value, &record)
	if err != nil {
		llog.Error("findLotteryDrawRecord", "err", err)
		return nil, err
	}
	return &record, nil
}

func (lott *Lottery) saveLotteryBuy(lotterylog *types.ReceiptLottery, txId string) (kvs []*types.KeyValue) {
	key := calcLotteryBuyKey(lotterylog.LotteryId, lotterylog.Addr, lotterylog.Round, txId)
	kv := &types.KeyValue{}
	record := &types.LotteryBuyRecord{lotterylog.Number, lotterylog.Amount, lotterylog.Round}
	kv = &types.KeyValue{key, types.Encode(record)}

	kvs = append(kvs, kv)
	return kvs
}

func (lott *Lottery) deleteLotteryBuy(lotterylog *types.ReceiptLottery, txId string) (kvs []*types.KeyValue) {
	key := calcLotteryBuyKey(lotterylog.LotteryId, lotterylog.Addr, lotterylog.Round, txId)

	kv := &types.KeyValue{key, nil}
	kvs = append(kvs, kv)
	return kvs
}

func (lott *Lottery) saveLotteryDraw(lotterylog *types.ReceiptLottery) (kvs []*types.KeyValue) {
	key := calcLotteryDrawKey(lotterylog.LotteryId, lotterylog.Round)
	kv := &types.KeyValue{}
	record := &types.LotteryDrawRecord{lotterylog.LuckyNumber, lotterylog.Round}
	kv = &types.KeyValue{key, types.Encode(record)}
	kvs = append(kvs, kv)
	return kvs
}

func (lott *Lottery) deleteLotteryDraw(lotterylog *types.ReceiptLottery) (kvs []*types.KeyValue) {
	key := calcLotteryDrawKey(lotterylog.LotteryId, lotterylog.Round)
	kv := &types.KeyValue{key, nil}
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
