package lottery

import (
	//"bytes"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"time"

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/common"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/types"
)

const (
	exciting = 100000 / 2
	lucky    = 1000 / 2
	happy    = 100 / 2
	notbad   = 10 / 2
)

const (
	minPurchasePeriod  = 360 //10m
	minShowPeriod      = 180 //3m
	maxAmountPerTicket = 10000
)

const defaultAddrPurTimes = 10
const luckyNumMol = 100000
const bty = 100000000 //1e8
const randMolNum = 5

type LotteryDB struct {
	types.Lottery
}

func NewLotteryDB(lotteryId string, purchasePeriod int64, showPeriod int64,
	maxPurchaseNum int64, blocktime int64, addr string) *LotteryDB {
	lott := &LotteryDB{}
	lott.LotteryId = lotteryId
	lott.PurchasePeriod = purchasePeriod
	lott.ShowPeriod = showPeriod
	lott.MaxPurchaseNum = maxPurchaseNum
	lott.CreateTime = blocktime
	lott.Fund = 0
	lott.Status = types.LotteryCreated //invalid status
	//lott.LastTransToPurState = blocktime
	//lott.Records = make(map[string]*types.PurchaseRecords) make is useless if no value set
	lott.TotalPurchasedTxNum = 0
	lott.TotalShowedNum = 0
	lott.CreateAddr = addr
	lott.Round = 1
	//lott.LuckyNumber = make([]int64, 1000)
	return lott
}

func (lott *LotteryDB) GetKVSet() (kvset []*types.KeyValue) {
	value := types.Encode(&lott.Lottery)
	kvset = append(kvset, &types.KeyValue{Key(lott.LotteryId), value})
	return kvset
}

func (lott *LotteryDB) Save(db dbm.KV) {
	set := lott.GetKVSet()
	for i := 0; i < len(set); i++ {
		db.Set(set[i].GetKey(), set[i].Value)
	}
}

func Key(id string) (key []byte) {
	key = append(key, []byte("mavl-"+types.ExecName(types.LotteryX)+"-")...)
	key = append(key, []byte(id)...)
	return key
}

type Action struct {
	coinsAccount *account.DB
	db           dbm.KV
	txhash       []byte
	fromaddr     string
	blocktime    int64
	height       int64
	execaddr     string
	difficulty   uint64
	api          client.QueueProtocolAPI
}

func NewLotteryAction(l *Lottery, tx *types.Transaction) *Action {
	hash := tx.Hash()
	fromaddr := tx.From()
	return &Action{l.GetCoinsAccount(), l.GetStateDB(), hash, fromaddr, l.GetBlockTime(), l.GetHeight(), l.GetAddr(), l.GetDifficulty(), l.GetApi()}
}

func (action *Action) GetReceiptLog(lottery *types.Lottery, preStatus int32, logTy int32,
	round int64, number int64, amount int64) *types.ReceiptLog {
	log := &types.ReceiptLog{}
	l := &types.ReceiptLottery{}

	log.Ty = logTy

	l.LotteryId = lottery.LotteryId
	l.Status = lottery.Status
	l.PrevStatus = preStatus
	if logTy == types.TyLogLotteryShow {
		l.Round = round
		l.Number = number
		l.Amount = amount
	}

	log.Log = types.Encode(l)
	return log
}

func (action *Action) LotteryCreate(create *types.LotteryCreate) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	var receipt *types.Receipt

	lotteryId := common.ToHex(action.txhash)

	if !isRightCreator(action.fromaddr) {
		return nil, types.ErrNoPrivilege
	}

	if create.GetPurchasePeriod() < minPurchasePeriod {
		return nil, types.ErrLotteryPurPeriodLimit
	}

	if create.GetShowPeriod() < minShowPeriod {
		return nil, types.ErrLotteryShowPeriodLimit
	}

	//no Limitation for MaxPurchaseNum currently

	_, err := findLottery(action.db, lotteryId)
	if err != types.ErrNotFound {
		llog.Error("LotteryCreate", "LotteryCreate repeated", lotteryId)
		return nil, types.ErrLotteryRepeatHash
	}

	lott := NewLotteryDB(lotteryId, create.PurchasePeriod,
		create.ShowPeriod, create.MaxPurchaseNum, action.blocktime, action.fromaddr)

	llog.Error("LotteryCreate created", "lotteryId", lotteryId)

	//lott.Records[action.fromaddr] = &types.PurchaseRecords{}

	lott.Save(action.db)
	kv = append(kv, lott.GetKVSet()...)

	receiptLog := action.GetReceiptLog(&lott.Lottery, 0, types.TyLogLotteryCreate, 0, 0, 0)
	logs = append(logs, receiptLog)

	receipt = &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil
}

//one bty for one fund ticket
func (action *Action) LotteryBuy(buy *types.LotteryBuy) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	//var receipt *types.Receipt

	lottery, err := findLottery(action.db, buy.LotteryId)
	if err != nil {
		llog.Error("LotteryBuy", "LotteryId", buy.LotteryId)
		return nil, err
	}

	lott := &LotteryDB{*lottery}
	preStatus := lott.Status

	if lott.Status != types.LotteryPurchase && lott.Status != types.LotteryCreated {
		llog.Error("LotteryBuy", "status", lott.Status)
		return nil, types.ErrLotteryStatus
	}

	if lott.CreateAddr == action.fromaddr {
		return nil, types.ErrLotteryCreatorBuy
	}

	if buy.GetAmount() <= 0 || buy.GetAmount() > maxAmountPerTicket {
		llog.Error("LotteryBuy", "buyAmount", buy.GetAmount())
		return nil, types.ErrLotteryBuyAmount
	}

	if lott.Records == nil {
		llog.Error("LotteryBuy records init")
		lott.Records = make(map[string]*types.PurchaseRecords)
	}

	if lott.Status != types.LotteryPurchase {
		llog.Error("LotteryBuy switch to purchasestate")
		lott.LastTransToPurState = action.blocktime
		lott.Status = types.LotteryPurchase
	}

	newRecord := &types.PurchaseRecord{buy.GetHashValue(), buy.GetAmount(), common.ToHex(action.txhash), false, 0}
	llog.Error("LotteryBuy", "Purhash", buy.GetHashValue(), "amount", buy.GetAmount(), "txhash", common.ToHex(action.txhash))
	/**********
	Once ExecTransfer succeed, ExecFrozen succeed, no roolback needed
	**********/

	receipt, err := action.coinsAccount.ExecTransfer(action.fromaddr, lott.CreateAddr, action.execaddr, buy.GetAmount()*100000000)
	if err != nil {
		llog.Error("LotteryBuy.ExecTransfer", "addr", action.fromaddr, "execaddr", action.execaddr, "amount", buy.GetAmount())
		return nil, err
	}
	logs = append(logs, receipt.Logs...)
	kv = append(kv, receipt.KV...)

	receipt, err = action.coinsAccount.ExecFrozen(lott.CreateAddr, action.execaddr, buy.GetAmount()*100000000)

	if err != nil {
		llog.Error("LotteryBuy.Frozen", "addr", lott.CreateAddr, "execaddr", action.execaddr, "amount", buy.GetAmount())
		return nil, err
	}
	logs = append(logs, receipt.Logs...)
	kv = append(kv, receipt.KV...)

	lott.Fund += buy.Amount

	if record, ok := lott.Records[action.fromaddr]; ok {
		record.Record = append(record.Record, newRecord)
	} else {
		initrecord := &types.PurchaseRecords{}
		initrecord.Record = append(initrecord.Record, newRecord)
		initrecord.FundWin = 0
		initrecord.AmountOneRound = 0
		lott.Records[action.fromaddr] = initrecord
	}
	lott.Records[action.fromaddr].AmountOneRound += buy.Amount
	lott.TotalPurchasedTxNum++

	//state auto switch
	if action.blocktime-lott.GetLastTransToPurState() > lott.GetPurchasePeriod() {
		llog.Error("LotteryBuy switch to showstate/auto switch")
		lott.Status = types.LotteryShowing
		lott.LastTransToShowState = action.blocktime
	}

	lott.Save(action.db)
	kv = append(kv, lott.GetKVSet()...)

	receiptLog := action.GetReceiptLog(&lott.Lottery, preStatus, types.TyLogLotteryBuy, 0, 0, 0)
	logs = append(logs, receiptLog)

	receipt = &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil
}

func (action *Action) LotteryShow(show *types.LotteryShow) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	var receipt *types.Receipt
	var showSuccess bool = false
	var tempAmount int64 = 0
	lottery, err := findLottery(action.db, show.LotteryId)
	if err != nil {
		llog.Error("LotteryShow", "LotteryId", show.LotteryId)
		return nil, err
	}
	lott := &LotteryDB{*lottery}

	preStatus := lott.Status

	if show.Number < 0 || show.Number >= luckyNumMol {
		llog.Error("LotteryShow", "show.Number", show.Number)
		return nil, types.ErrLotteryShowNumber
	}

	if lott.Status != types.LotteryShowing {
		if lott.Status != types.LotteryPurchase {
			llog.Error("LotteryShow", "lott.Status", lott.Status)
			return nil, types.ErrLotteryStatus
		} else {
			if action.blocktime-lott.GetLastTransToPurState() < lott.GetPurchasePeriod() {
				return nil, types.ErrLotteryStatus
			} else {
				llog.Error("LotteryShow switch to showStatus")
				lott.Status = types.LotteryShowing
				lott.LastTransToShowState = action.blocktime
			}
		}
	}

	//for convenience
	tempNumStr := fmt.Sprintf("%05d", show.Number)
	llog.Error("LotteryShow", "Secret", show.Secret, "txhash", show.TxHash, "number", show.Number, "tempNumStr", tempNumStr)

	if record, ok := lott.Records[action.fromaddr]; ok {
		llog.Error("LotteryShow find map")
		for index, rec := range record.Record {
			llog.Error("LotteryShow find one record")
			if rec.TxHash == show.TxHash {
				llog.Error("LotteryShow find txhash")
				if common.ToHex(common.Sha256([]byte(show.GetSecret()+tempNumStr))) == rec.HashValue {
					llog.Error("LotteryShow find the value")
					showSuccess = true
					if record.Record[index].IsShowed {
						return nil, types.ErrLotteryShowRepeated
					} else {
						record.Record[index].IsShowed = true
						record.Record[index].Number = show.Number
						tempAmount = record.Record[index].Amount
						lott.TotalShowedNum++
					}
				}
			}
		}
	}

	//optional
	if !showSuccess {
		return nil, types.ErrLotteryShowError
	}

	if lott.TotalShowedNum == lott.TotalPurchasedTxNum {
		llog.Error("LotteryShow auto draw")
		//auto switch
		rec, err := action.checkDraw(lott)
		if err != nil {
			return nil, err
		}
		kv = append(kv, rec.KV...)
		logs = append(logs, rec.Logs...)
	}

	lott.Save(action.db)
	kv = append(kv, lott.GetKVSet()...)

	receiptLog := action.GetReceiptLog(&lott.Lottery, preStatus, types.TyLogLotteryShow, lott.Round, show.Number, tempAmount)
	logs = append(logs, receiptLog)

	receipt = &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil
}

//Anyone who buy a ticket before can take a draw action
func (action *Action) LotteryDraw(draw *types.LotteryDraw) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	var receipt *types.Receipt

	lottery, err := findLottery(action.db, draw.LotteryId)
	if err != nil {
		llog.Error("LotteryBuy", "LotteryId", draw.LotteryId)
		return nil, err
	}

	lott := &LotteryDB{*lottery}

	preStatus := lott.Status

	if lott.Status != types.LotteryShowing {
		llog.Error("LotteryDraw", "lott.Status", lott.Status)
		return nil, types.ErrLotteryStatus
	}

	if action.blocktime-lott.GetLastTransToShowState() < lott.GetShowPeriod() {
		return nil, types.ErrLotteryStatus
	}

	if _, ok := lott.Records[action.fromaddr]; !ok {
		return nil, types.ErrLotteryDrawActionInvalid
	}

	rec, err := action.checkDraw(lott)
	if err != nil {
		return nil, err
	}
	kv = append(kv, rec.KV...)
	logs = append(logs, rec.Logs...)

	lott.Save(action.db)
	kv = append(kv, lott.GetKVSet()...)

	receiptLog := action.GetReceiptLog(&lott.Lottery, preStatus, types.TyLogLotteryDraw, 0, 0, 0)
	logs = append(logs, receiptLog)

	receipt = &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil
}

func (action *Action) LotteryClose(draw *types.LotteryClose) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	//var receipt *types.Receipt

	if !isEableToClose() {
		return nil, types.ErrLotteryErrUnableClose
	}

	lottery, err := findLottery(action.db, draw.LotteryId)
	if err != nil {
		llog.Error("LotteryBuy", "LotteryId", draw.LotteryId)
		return nil, err
	}

	lott := &LotteryDB{*lottery}
	preStatus := lott.Status

	if action.fromaddr != lott.CreateAddr {
		return nil, types.ErrLotteryErrCloser
	}

	if lott.Status == types.LotteryShowing || lott.Status == types.LotteryClosed {
		return nil, types.ErrLotteryStatus
	}

	addrkeys := make([]string, len(lott.Records))
	i := 0
	var totalReturn int64 = 0
	for addr, _ := range lott.Records {
		totalReturn += lott.Records[addr].AmountOneRound
		addrkeys[i] = addr
		i++
	}
	llog.Error("LotteryClose", "totalReturn", totalReturn)

	if totalReturn > 0 {

		if !action.CheckExecAccount(lott.CreateAddr, bty*totalReturn, true) {
			return nil, types.ErrLotteryFundNotEnough
		}

		sort.Strings(addrkeys)

		for _, addr := range addrkeys {
			if lott.Records[addr].AmountOneRound > 0 {
				receipt, err := action.coinsAccount.ExecTransferFrozen(lott.CreateAddr, addr, action.execaddr,
					bty*lott.Records[addr].AmountOneRound)
				if err != nil {
					return nil, err
				}

				kv = append(kv, receipt.KV...)
				logs = append(logs, receipt.Logs...)
			}
		}
	}

	for addr, _ := range lott.Records {
		lott.Records[addr].Record = lott.Records[addr].Record[0:0]
		delete(lott.Records, addr)
	}

	lott.LastTransToPurState = action.blocktime
	lott.TotalShowedNum = 0
	lott.TotalPurchasedTxNum = 0
	llog.Error("LotteryClose switch to closestate")
	lott.Status = types.LotteryClosed

	lott.Save(action.db)
	kv = append(kv, lott.GetKVSet()...)

	receiptLog := action.GetReceiptLog(&lott.Lottery, preStatus, types.TyLogLotteryClose, 0, 0, 0)
	logs = append(logs, receiptLog)

	return &types.Receipt{types.ExecOk, kv, logs}, nil

}

//random used for verfication in solo
func (action *Action) findLuckyNum(isSolo bool) int64 {
	var num int64 = 0
	if isSolo {
		random := rand.New(rand.NewSource(time.Now().UnixNano()))
		num = random.Int63() % luckyNumMol
		num = 12345
	} else {
		llog.Error("findLuckyNum", "height", action.height, "diffculty", action.difficulty)
		baseNum := action.difficulty % luckyNumMol

		llog.Error("findLuckyNum", "baseNum", baseNum)
		relationNum := baseNum%randMolNum + 1
		var i int64 = 0
		for {
			if action.height-i < 0 {
				break
			}

			formerHash, err := action.api.GetBlockHash(&types.ReqInt{action.height - i - 1})
			if err != nil {
				llog.Error("findLuckyNum", "err", err)
				return -1
			}
			//many empty block in parachain
			//if formerHash == 0 {
			//	continue
			//}
			hashNum, err := strconv.ParseUint(common.ToHex(formerHash.Hash[0:4]), 0, 64)
			if err != nil {
				llog.Error("findLuckyNum", "err", err)
				return -1
			}
			llog.Error("findLuckyNum", "hashNum", hashNum)
			baseNum += hashNum
			relationNum -= 1
			i += 1
			if relationNum == 0 {
				break
			}
		}
		num = int64(baseNum) % luckyNumMol
	}

	return num
}

func checkFundAmount(luckynum int64, guessnum int64) int64 {
	if luckynum == guessnum {
		return exciting
	} else if luckynum%1000 == guessnum%1000 {
		return lucky
	} else if luckynum%100 == guessnum%100 {
		return happy
	} else if luckynum%10 == guessnum%10 {
		return notbad
	} else {
		return 0
	}
}

func (action *Action) checkDraw(lott *LotteryDB) (*types.Receipt, error) {
	llog.Error("checkDraw")

	luckynum := action.findLuckyNum(false)
	if luckynum < 0 || luckynum >= luckyNumMol {
		return nil, types.ErrLotteryErrLuckyNum
	}

	llog.Error("checkDraw", "luckynum", luckynum)

	//var receipt *types.Receipt
	var logs []*types.ReceiptLog
	var kv []*types.KeyValue

	//calculate fund for all participant showed their number
	var tempFund int64 = 0
	var totalFund int64 = 0
	addrkeys := make([]string, len(lott.Records))
	i := 0
	for addr, _ := range lott.Records {
		addrkeys[i] = addr
		i++
		for _, rec := range lott.Records[addr].Record {
			if rec.IsShowed {
				fund := checkFundAmount(luckynum, rec.Number)
				tempFund = fund * rec.Amount
				lott.Records[addr].FundWin += tempFund
				totalFund += tempFund
			}
		}
	}
	var factor float64 = 0
	if totalFund > lott.GetFund()/2 {
		llog.Error("checkDraw ajust fund", "lott.Fund", lott.Fund, "totalFund", totalFund)
		factor = (float64)(lott.GetFund()) / 2 / (float64)(totalFund)
		lott.Fund = lott.Fund / 2
	} else {
		factor = 1.0
		lott.Fund -= totalFund
	}

	llog.Error("checkDraw", "factor", factor, "totalFund", totalFund)

	//protection for rollback
	if factor == 1.0 {
		if !action.CheckExecAccount(lott.CreateAddr, totalFund, true) {
			return nil, types.ErrLotteryFundNotEnough
		}
	} else {
		if !action.CheckExecAccount(lott.CreateAddr, bty*lott.Fund/2+1, true) {
			return nil, types.ErrLotteryFundNotEnough
		}
	}

	sort.Strings(addrkeys)

	for _, addr := range addrkeys {
		fund := (lott.Records[addr].FundWin * int64(factor*exciting)) * bty / exciting //any problem when too little?
		llog.Error("checkDraw", "fund", fund)
		if fund > 0 {
			receipt, err := action.coinsAccount.ExecTransferFrozen(lott.CreateAddr, addr, action.execaddr, fund)
			if err != nil {
				return nil, err
			}

			kv = append(kv, receipt.KV...)
			logs = append(logs, receipt.Logs...)
		}
	}

	for addr, _ := range lott.Records {
		lott.Records[addr].Record = lott.Records[addr].Record[0:0]
		delete(lott.Records, addr)
	}

	llog.Error("checkDraw lottery switch to createStatus")
	//lott.LastTransToPurState = action.blocktime
	lott.Status = types.LotteryCreated
	lott.TotalShowedNum = 0
	lott.TotalPurchasedTxNum = 0
	lott.Round += 1
	//if len(lott.LuckyNumber)
	lott.LuckyNumber = append(lott.LuckyNumber, luckynum)

	return &types.Receipt{types.ExecOk, kv, logs}, nil
}

func isRightCreator(addr string) bool {
	return true
}

func isEableToClose() bool {
	return true
}

func findLottery(db dbm.KV, lotteryId string) (*types.Lottery, error) {
	data, err := db.Get(Key(lotteryId))
	if err != nil {
		llog.Debug("findLottery", "get", err)
		return nil, err
	}
	var lott types.Lottery
	//decode
	err = types.Decode(data, &lott)
	if err != nil {
		llog.Debug("findLottery", "decode", err)
		return nil, err
	}
	return &lott, nil
}

func (action *Action) CheckExecAccount(addr string, amount int64, isFrozen bool) bool {
	acc := action.coinsAccount.LoadExecAccount(addr, action.execaddr)
	if isFrozen {
		if acc.GetFrozen() >= amount {
			return true
		}
	} else {
		if acc.GetBalance() >= amount {
			return true
		}
	}

	return false
}
