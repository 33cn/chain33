package executor

import (
	"sort"

	log "github.com/inconshreveable/log15"
	pty "gitlab.33.cn/chain33/chain33/plugin/dapp/lottery/types"
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
)

var llog = log.New("module", "execs.lottery")
var driverName = pty.LotteryX

func init() {
	ety := types.LoadExecutorType(driverName)
	ety.InitFuncList(types.ListMethod(&Lottery{}))
}

type subConfig struct {
	ParaRemoteGrpcClient string `json:"paraRemoteGrpcClient"`
}

var cfg subConfig

func Init(name string, sub []byte) {
	driverName := GetName()
	if name != driverName {
		panic("system dapp can't be rename")
	}
	if sub != nil {
		types.MustDecode(sub, &cfg)
	}
	drivers.Register(driverName, newLottery, types.GetDappFork(driverName, "Enable"))
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
	l.SetExecutorType(types.LoadExecutorType(driverName))
	return l
}

func (l *Lottery) GetDriverName() string {
	return pty.LotteryX
}

func (lott *Lottery) findLotteryBuyRecords(key []byte) (*pty.LotteryBuyRecords, error) {

	count := lott.GetLocalDB().PrefixCount(key)
	llog.Error("findLotteryBuyRecords", "count", count)

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

func (lott *Lottery) findLotteryBuyRecord(key []byte) (*pty.LotteryBuyRecord, error) {
	value, err := lott.GetLocalDB().Get(key)
	if err != nil && err != types.ErrNotFound {
		llog.Error("findLotteryBuyRecord", "err", err)
		return nil, err
	}
	if err == types.ErrNotFound {
		return nil, nil
	}
	var record pty.LotteryBuyRecord

	err = types.Decode(value, &record)
	if err != nil {
		llog.Error("findLotteryBuyRecord", "err", err)
		return nil, err
	}
	return &record, nil
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

func (lott *Lottery) saveLotteryBuy(lotterylog *pty.ReceiptLottery) (kvs []*types.KeyValue) {
	key := calcLotteryBuyKey(lotterylog.LotteryId, lotterylog.Addr, lotterylog.Round, lotterylog.Index)
	kv := &types.KeyValue{}
	record := &pty.LotteryBuyRecord{lotterylog.Number, lotterylog.Amount, lotterylog.Round, 0, lotterylog.Way, lotterylog.Index, lotterylog.Time, lotterylog.TxHash}
	kv = &types.KeyValue{key, types.Encode(record)}

	kvs = append(kvs, kv)
	return kvs
}

func (lott *Lottery) deleteLotteryBuy(lotterylog *pty.ReceiptLottery) (kvs []*types.KeyValue) {
	key := calcLotteryBuyKey(lotterylog.LotteryId, lotterylog.Addr, lotterylog.Round, lotterylog.Index)

	kv := &types.KeyValue{key, nil}
	kvs = append(kvs, kv)
	return kvs
}

func (lott *Lottery) updateLotteryBuy(lotterylog *pty.ReceiptLottery, isAdd bool) (kvs []*types.KeyValue) {
	if lotterylog.UpdateInfo != nil {
		llog.Debug("updateLotteryBuy")
		buyInfo := lotterylog.UpdateInfo.BuyInfo
		//sort for map
		addrkeys := make([]string, len(buyInfo))
		i := 0

		for addr := range buyInfo {
			addrkeys[i] = addr
			i++
		}
		sort.Strings(addrkeys)
		//update old record
		for _, addr := range addrkeys {
			for _, updateRec := range buyInfo[addr].Records {
				//find addr, index
				key := calcLotteryBuyKey(lotterylog.LotteryId, addr, lotterylog.Round, updateRec.Index)
				record, err := lott.findLotteryBuyRecord(key)
				if err != nil || record == nil {
					return kvs
				}
				kv := &types.KeyValue{}

				if isAdd {
					llog.Debug("updateLotteryBuy update key")
					record.Type = updateRec.Type
				} else {
					record.Type = 0
				}

				kv = &types.KeyValue{key, types.Encode(record)}
				kvs = append(kvs, kv)
			}
		}
		return kvs
	}
	return kvs
}

func (lott *Lottery) saveLotteryDraw(lotterylog *pty.ReceiptLottery) (kvs []*types.KeyValue) {
	key := calcLotteryDrawKey(lotterylog.LotteryId, lotterylog.Round)
	kv := &types.KeyValue{}
	record := &pty.LotteryDrawRecord{lotterylog.LuckyNumber, lotterylog.Round, lotterylog.Time, lotterylog.TxHash}
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

func (lott *Lottery) GetPayloadValue() types.Message {
	return &pty.LotteryAction{}
}
