package executor

import (
	"reflect"

	log "github.com/inconshreveable/log15"
	pty "gitlab.33.cn/chain33/chain33/plugin/dapp/lottery/types"
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
)

var llog = log.New("module", "execs.lottery")

//初始化过程比较重量级，有很多reflact, 所以弄成全局的
var executorFunList = make(map[string]reflect.Method)
var executorType = pty.NewType()

func init() {
	actionFunList := executorType.GetFuncMap()
	executorFunList = types.ListMethod(&Lottery{})
	for k, v := range actionFunList {
		executorFunList[k] = v
	}
}

func Init(name string) {
	driverName := GetName()
	if name != driverName {
		panic("system dapp can't be rename")
	}
	drivers.Register(driverName, newLottery, 0)
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
	l.SetExecutorType(executorType)
	return l
}

func (l *Lottery) GetDriverName() string {
	return types.LotteryX
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

func (lott *Lottery) GetFuncMap() map[string]reflect.Method {
	return executorFunList
}

func (lott *Lottery) GetPayloadValue() types.Message {
	return &pty.LotteryAction{}
}
