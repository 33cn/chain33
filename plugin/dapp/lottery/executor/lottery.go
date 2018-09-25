package executor

import (
	"fmt"

	log "github.com/inconshreveable/log15"
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
	pty "gitlab.33.cn/chain33/chain33/plugin/dapp/lottery/types"
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

func (lott *Lottery) findLotteryBuyRecord(key []byte) (*pty.LotteryBuyRecords, error) {

	count := lott.GetLocalDB().PrefixCount(key)
	llog.Error("findLotteryBuyRecord", "count", count)

	values, err := lott.GetLocalDB().List(key, nil, int32(count), 0)
	if err != nil {
		return nil, err
	}
	var records pty.LotteryBuyRecords

	for _, value := range values {
		var record pty.LotteryBuyRecord
		err := types.Decode(value, &record)
		if err != nil {
			continue
		}
		records.Records = append(records.Records, &record)
	}

	return &records, nil
}

func (lott *Lottery) findLotteryDrawRecord(key []byte) (*pty.LotteryDrawRecord, error) {
	value, err := lott.GetLocalDB().Get(key)
	if err != nil && err != types.ErrNotFound {
		llog.Error("findLotteryDrawRecord", "err", err)
		return nil, err
	}
	if err == types.ErrNotFound {
		return nil, nil
	}
	var record pty.LotteryDrawRecord

	err = types.Decode(value, &record)
	if err != nil {
		llog.Error("findLotteryDrawRecord", "err", err)
		return nil, err
	}
	return &record, nil
}

func (lott *Lottery) saveLotteryBuy(lotterylog *pty.ReceiptLottery, txId string) (kvs []*types.KeyValue) {
	key := calcLotteryBuyKey(lotterylog.LotteryId, lotterylog.Addr, lotterylog.Round, txId)
	kv := &types.KeyValue{}
	record := &pty.LotteryBuyRecord{lotterylog.Number, lotterylog.Amount, lotterylog.Round}
	kv = &types.KeyValue{key, types.Encode(record)}

	kvs = append(kvs, kv)
	return kvs
}

func (lott *Lottery) deleteLotteryBuy(lotterylog *pty.ReceiptLottery, txId string) (kvs []*types.KeyValue) {
	key := calcLotteryBuyKey(lotterylog.LotteryId, lotterylog.Addr, lotterylog.Round, txId)

	kv := &types.KeyValue{key, nil}
	kvs = append(kvs, kv)
	return kvs
}

func (lott *Lottery) saveLotteryDraw(lotterylog *pty.ReceiptLottery) (kvs []*types.KeyValue) {
	key := calcLotteryDrawKey(lotterylog.LotteryId, lotterylog.Round)
	kv := &types.KeyValue{}
	record := &pty.LotteryDrawRecord{lotterylog.LuckyNumber, lotterylog.Round}
	kv = &types.KeyValue{key, types.Encode(record)}
	kvs = append(kvs, kv)
	return kvs
}

func (lott *Lottery) deleteLotteryDraw(lotterylog *pty.ReceiptLottery) (kvs []*types.KeyValue) {
	key := calcLotteryDrawKey(lotterylog.LotteryId, lotterylog.Round)
	kv := &types.KeyValue{key, nil}
	kvs = append(kvs, kv)
	return kvs
}

func (lott *Lottery) saveLottery(lotterylog *pty.ReceiptLottery) (kvs []*types.KeyValue) {
	if lotterylog.PrevStatus > 0 {
		kv := dellottery(lotterylog.LotteryId, lotterylog.PrevStatus)
		kvs = append(kvs, kv)
	}
	kvs = append(kvs, addlottery(lotterylog.LotteryId, lotterylog.Status))
	return kvs
}

func (lott *Lottery) deleteLottery(lotterylog *pty.ReceiptLottery) (kvs []*types.KeyValue) {
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
