package executor

import (
	"fmt"
	"sort"
	"strconv"

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/common"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
	"google.golang.org/grpc"
)

const (
	exciting = 100000 / 2
	lucky    = 1000 / 2
	happy    = 100 / 2
	notbad   = 10 / 2
)

const (
	minPurBlockNum  = 40 //10 minutes
	minDrawBlockNum = 50
)

const (
	creatorKey = "lottery-creator"
)

const (
	ListDESC    = int32(0)
	ListASC     = int32(1)
	DefultCount = int32(20)  //默认一次取多少条记录
	MaxCount    = int32(100) //最多取100条
)

//const defaultAddrPurTimes = 10
const luckyNumMol = 100000
const decimal = 100000000 //1e8
const randMolNum = 5
const grpcRecSize int = 5 * 30 * 1024 * 1024
const blockNum = 5

type LotteryDB struct {
	types.Lottery
}

func NewLotteryDB(lotteryId string, purBlock int64, drawBlock int64,
	blockHeight int64, addr string) *LotteryDB {
	lott := &LotteryDB{}
	lott.LotteryId = lotteryId
	lott.PurBlockNum = purBlock
	lott.DrawBlockNum = drawBlock
	lott.CreateHeight = blockHeight
	lott.Fund = 0
	lott.Status = types.LotteryCreated
	lott.TotalPurchasedTxNum = 0
	lott.CreateAddr = addr
	lott.Round = 0
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
	key = append(key, []byte("mavl-"+types.LotteryX+"-")...)
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
	conn         *grpc.ClientConn
	grpcClient   types.Chain33Client
}

func NewLotteryAction(l *Lottery, tx *types.Transaction) *Action {
	hash := tx.Hash()
	fromaddr := tx.From()

	msgRecvOp := grpc.WithMaxMsgSize(grpcRecSize)
	conn, err := grpc.Dial(types.GetParaRemoteGrpcClient(), grpc.WithInsecure(), msgRecvOp)

	if err != nil {
		panic(err)
	}
	grpcClient := types.NewChain33Client(conn)

	return &Action{l.GetCoinsAccount(), l.GetStateDB(), hash, fromaddr, l.GetBlockTime(),
		l.GetHeight(), dapp.ExecAddress(string(tx.Execer)), l.GetDifficulty(), l.GetApi(), conn, grpcClient}
}

func (action *Action) GetReceiptLog(lottery *types.Lottery, preStatus int32, logTy int32,
	round int64, buyNumber int64, amount int64, luckyNum int64) *types.ReceiptLog {
	log := &types.ReceiptLog{}
	l := &types.ReceiptLottery{}

	log.Ty = logTy

	l.LotteryId = lottery.LotteryId
	l.Status = lottery.Status
	l.PrevStatus = preStatus
	if logTy == types.TyLogLotteryBuy {
		l.Round = round
		l.Number = buyNumber
		l.Amount = amount
		l.Addr = action.fromaddr
	}
	if logTy == types.TyLogLotteryDraw {
		l.Round = round
		l.LuckyNumber = luckyNum
	}

	log.Log = types.Encode(l)
	return log
}

func (action *Action) LotteryCreate(create *types.LotteryCreate) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	var receipt *types.Receipt

	lotteryId := common.ToHex(action.txhash)

	if !isRightCreator(action.fromaddr, action.db, false) {
		return nil, types.ErrNoPrivilege
	}

	if create.GetPurBlockNum() < minPurBlockNum {
		return nil, types.ErrLotteryPurBlockLimit
	}

	if create.GetDrawBlockNum() < minDrawBlockNum {
		return nil, types.ErrLotteryDrawBlockLimit
	}

	if create.GetPurBlockNum() > create.GetDrawBlockNum() {
		return nil, types.ErrLotteryDrawBlockLimit
	}

	_, err := findLottery(action.db, lotteryId)
	if err != types.ErrNotFound {
		llog.Error("LotteryCreate", "LotteryCreate repeated", lotteryId)
		return nil, types.ErrLotteryRepeatHash
	}

	lott := NewLotteryDB(lotteryId, create.GetPurBlockNum(),
		create.GetDrawBlockNum(), action.height, action.fromaddr)

	if types.IsPara() {
		mainHeight := action.GetMainHeightByTxHash(action.txhash)
		if mainHeight < 0 {
			llog.Error("LotteryCreate", "mainHeight", mainHeight)
			return nil, types.ErrLotteryStatus
		}
		lott.CreateOnMain = mainHeight
	}

	llog.Debug("LotteryCreate created", "lotteryId", lotteryId)

	lott.Save(action.db)
	kv = append(kv, lott.GetKVSet()...)

	receiptLog := action.GetReceiptLog(&lott.Lottery, 0, types.TyLogLotteryCreate, 0, 0, 0, 0)
	logs = append(logs, receiptLog)

	receipt = &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil
}

//one bty for one ticket
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

	if lott.Status == types.LotteryClosed {
		llog.Error("LotteryBuy", "status", lott.Status)
		return nil, types.ErrLotteryStatus
	}

	if lott.Status == types.LotteryDrawed {
		//no problem both on main and para
		if action.height <= lott.LastTransToDrawState {
			llog.Error("LotteryBuy", "action.heigt", action.height, "lastTransToDrawState", lott.LastTransToDrawState)
			return nil, types.ErrLotteryStatus
		}
	}

	if lott.Status == types.LotteryCreated || lott.Status == types.LotteryDrawed {
		llog.Debug("LotteryBuy switch to purchasestate")
		lott.LastTransToPurState = action.height
		lott.Status = types.LotteryPurchase
		lott.Round += 1
		if types.IsPara() {
			mainHeight := action.GetMainHeightByTxHash(action.txhash)
			if mainHeight < 0 {
				llog.Error("LotteryBuy", "mainHeight", mainHeight)
				return nil, types.ErrLotteryStatus
			}
			lott.LastTransToPurStateOnMain = mainHeight
		}
	}

	if lott.Status == types.LotteryPurchase {
		if types.IsPara() {
			mainHeight := action.GetMainHeightByTxHash(action.txhash)
			if mainHeight < 0 {
				llog.Error("LotteryBuy", "mainHeight", mainHeight)
				return nil, types.ErrLotteryStatus
			}
			if mainHeight-lott.LastTransToPurStateOnMain > lott.GetPurBlockNum() {
				llog.Error("LotteryBuy", "action.height", action.height, "mainHeight", mainHeight, "LastTransToPurStateOnMain", lott.LastTransToPurStateOnMain)
				return nil, types.ErrLotteryStatus
			}
		} else {
			if action.height-lott.LastTransToPurState > lott.GetPurBlockNum() {
				llog.Error("LotteryBuy", "action.height", action.height, "LastTransToPurState", lott.LastTransToPurState)
				return nil, types.ErrLotteryStatus
			}
		}
	}

	if lott.CreateAddr == action.fromaddr {
		return nil, types.ErrLotteryCreatorBuy
	}

	if buy.GetAmount() <= 0 {
		llog.Error("LotteryBuy", "buyAmount", buy.GetAmount())
		return nil, types.ErrLotteryBuyAmount
	}

	if buy.GetNumber() < 0 || buy.GetNumber() >= luckyNumMol {
		llog.Error("LotteryBuy", "buyNumber", buy.GetNumber())
		return nil, types.ErrLotteryBuyNumber
	}

	if lott.Records == nil {
		llog.Debug("LotteryBuy records init")
		lott.Records = make(map[string]*types.PurchaseRecords)
	}

	newRecord := &types.PurchaseRecord{buy.GetAmount(), buy.GetNumber()}
	llog.Debug("LotteryBuy", "amount", buy.GetAmount(), "number", buy.GetNumber())

	/**********
	Once ExecTransfer succeed, ExecFrozen succeed, no roolback needed
	**********/

	receipt, err := action.coinsAccount.ExecTransfer(action.fromaddr, lott.CreateAddr, action.execaddr, buy.GetAmount()*decimal)
	if err != nil {
		llog.Error("LotteryBuy.ExecTransfer", "addr", action.fromaddr, "execaddr", action.execaddr, "amount", buy.GetAmount())
		return nil, err
	}
	logs = append(logs, receipt.Logs...)
	kv = append(kv, receipt.KV...)

	receipt, err = action.coinsAccount.ExecFrozen(lott.CreateAddr, action.execaddr, buy.GetAmount()*decimal)

	if err != nil {
		llog.Error("LotteryBuy.Frozen", "addr", lott.CreateAddr, "execaddr", action.execaddr, "amount", buy.GetAmount())
		return nil, err
	}
	logs = append(logs, receipt.Logs...)
	kv = append(kv, receipt.KV...)

	lott.Fund += buy.GetAmount()

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

	lott.Save(action.db)
	kv = append(kv, lott.GetKVSet()...)

	receiptLog := action.GetReceiptLog(&lott.Lottery, preStatus, types.TyLogLotteryBuy, lott.Round, buy.GetNumber(), buy.GetAmount(), 0)
	logs = append(logs, receiptLog)

	receipt = &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil
}

//1.Anyone who buy a ticket
//2.Creator
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

	if lott.Status != types.LotteryPurchase {
		llog.Error("LotteryDraw", "lott.Status", lott.Status)
		return nil, types.ErrLotteryStatus
	}

	if types.IsPara() {
		mainHeight := action.GetMainHeightByTxHash(action.txhash)
		if mainHeight < 0 {
			llog.Error("LotteryBuy", "mainHeight", mainHeight)
			return nil, types.ErrLotteryStatus
		}
		if mainHeight-lott.GetLastTransToPurStateOnMain() < lott.GetDrawBlockNum() {
			llog.Error("LotteryDraw", "action.height", action.height, "mainHeight", mainHeight, "GetLastTransToPurStateOnMain", lott.GetLastTransToPurState())
			return nil, types.ErrLotteryStatus
		}
	} else {
		if action.height-lott.GetLastTransToPurState() < lott.GetDrawBlockNum() {
			llog.Error("LotteryDraw", "action.height", action.height, "GetLastTransToPurState", lott.GetLastTransToPurState())
			return nil, types.ErrLotteryStatus
		}
	}

	if action.fromaddr != lott.GetCreateAddr() {
		if _, ok := lott.Records[action.fromaddr]; !ok {
			llog.Error("LotteryDraw", "action.fromaddr", action.fromaddr)
			return nil, types.ErrLotteryDrawActionInvalid
		}
	}

	rec, err := action.checkDraw(lott)
	if err != nil {
		return nil, err
	}
	kv = append(kv, rec.KV...)
	logs = append(logs, rec.Logs...)

	lott.Save(action.db)
	kv = append(kv, lott.GetKVSet()...)

	receiptLog := action.GetReceiptLog(&lott.Lottery, preStatus, types.TyLogLotteryDraw, lott.Round, 0, 0, lott.LuckyNumber)
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

	if lott.Status == types.LotteryClosed {
		return nil, types.ErrLotteryStatus
	}

	addrkeys := make([]string, len(lott.Records))
	i := 0
	var totalReturn int64 = 0
	for addr := range lott.Records {
		totalReturn += lott.Records[addr].AmountOneRound
		addrkeys[i] = addr
		i++
	}
	llog.Debug("LotteryClose", "totalReturn", totalReturn)

	if totalReturn > 0 {

		if !action.CheckExecAccount(lott.CreateAddr, decimal*totalReturn, true) {
			return nil, types.ErrLotteryFundNotEnough
		}

		sort.Strings(addrkeys)

		for _, addr := range addrkeys {
			if lott.Records[addr].AmountOneRound > 0 {
				receipt, err := action.coinsAccount.ExecTransferFrozen(lott.CreateAddr, addr, action.execaddr,
					decimal*lott.Records[addr].AmountOneRound)
				if err != nil {
					return nil, err
				}

				kv = append(kv, receipt.KV...)
				logs = append(logs, receipt.Logs...)
			}
		}
	}

	for addr := range lott.Records {
		lott.Records[addr].Record = lott.Records[addr].Record[0:0]
		delete(lott.Records, addr)
	}

	lott.TotalPurchasedTxNum = 0
	llog.Debug("LotteryClose switch to closestate")
	lott.Status = types.LotteryClosed

	lott.Save(action.db)
	kv = append(kv, lott.GetKVSet()...)

	receiptLog := action.GetReceiptLog(&lott.Lottery, preStatus, types.TyLogLotteryClose, 0, 0, 0, 0)
	logs = append(logs, receiptLog)

	return &types.Receipt{types.ExecOk, kv, logs}, nil
}

func (action *Action) GetModify(beg, end int64, randMolNum int64) ([]byte, error) {
	//通过某个区间计算modify
	timeSource := int64(0)
	total := int64(0)
	//last := []byte("last")
	newmodify := ""
	for i := beg; i < end; i += randMolNum {
		req := &types.ReqBlocks{i, i, false, []string{""}}
		blocks, err := action.api.GetBlocks(req)
		if err != nil {
			return []byte{}, err
		}
		block := blocks.Items[0].Block
		timeSource += block.BlockTime
		total += block.BlockTime
	}

	//for main chain, 5 latest block
	//for para chain, 5 latest block -- 5 sequence main block
	txActions, err := action.getTxActions(end, blockNum)
	if err != nil {
		return nil, err
	}

	//modify, bits, id
	var modifies []byte
	var bits uint32
	var ticketIds string

	for _, ticketAction := range txActions {
		llog.Debug("GetModify", "modify", ticketAction.GetMiner().GetModify(), "bits", ticketAction.GetMiner().GetBits(), "ticketId", ticketAction.GetMiner().GetTicketId())
		modifies = append(modifies, ticketAction.GetMiner().GetModify()...)
		bits += ticketAction.GetMiner().GetBits()
		ticketIds += ticketAction.GetMiner().GetTicketId()
	}

	newmodify = fmt.Sprintf("%s:%s:%d:%d", string(modifies), ticketIds, total, bits)

	modify := common.Sha256([]byte(newmodify))
	return modify, nil
}

//random used for verfication in solo
func (action *Action) findLuckyNum(isSolo bool, lott *LotteryDB) int64 {
	var num int64 = 0
	if isSolo {
		//used for internal verfication
		num = 12345
	} else {
		randMolNum := (lott.TotalPurchasedTxNum+action.height-lott.LastTransToPurState)%3 + 2 //3~5

		modify, err := action.GetModify(lott.LastTransToPurState, action.height-1, randMolNum)
		llog.Error("findLuckyNum", "begin", lott.LastTransToPurState, "end", action.height-1, "randMolNum", randMolNum)

		if err != nil {
			llog.Error("findLuckyNum", "err", err)
			return -1
		}

		baseNum, err := strconv.ParseUint(common.ToHex(modify[0:4]), 0, 64)
		if err != nil {
			llog.Error("findLuckyNum", "err", err)
			return -1
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
	llog.Debug("checkDraw")

	luckynum := action.findLuckyNum(false, lott)
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
	for addr := range lott.Records {
		addrkeys[i] = addr
		i++
		for _, rec := range lott.Records[addr].Record {
			fund := checkFundAmount(luckynum, rec.Number)
			tempFund = fund * rec.Amount
			lott.Records[addr].FundWin += tempFund
			totalFund += tempFund
		}
	}
	var factor float64 = 0
	if totalFund > lott.GetFund()/2 {
		llog.Debug("checkDraw ajust fund", "lott.Fund", lott.Fund, "totalFund", totalFund)
		factor = (float64)(lott.GetFund()) / 2 / (float64)(totalFund)
		lott.Fund = lott.Fund / 2
	} else {
		factor = 1.0
		lott.Fund -= totalFund
	}

	llog.Debug("checkDraw", "factor", factor, "totalFund", totalFund)

	//protection for rollback
	if factor == 1.0 {
		if !action.CheckExecAccount(lott.CreateAddr, totalFund, true) {
			return nil, types.ErrLotteryFundNotEnough
		}
	} else {
		if !action.CheckExecAccount(lott.CreateAddr, decimal*lott.Fund/2+1, true) {
			return nil, types.ErrLotteryFundNotEnough
		}
	}

	sort.Strings(addrkeys)

	for _, addr := range addrkeys {
		fund := (lott.Records[addr].FundWin * int64(factor*exciting)) * decimal / exciting //any problem when too little?
		llog.Debug("checkDraw", "fund", fund)
		if fund > 0 {
			receipt, err := action.coinsAccount.ExecTransferFrozen(lott.CreateAddr, addr, action.execaddr, fund)
			if err != nil {
				return nil, err
			}

			kv = append(kv, receipt.KV...)
			logs = append(logs, receipt.Logs...)
		}
	}

	for addr := range lott.Records {
		lott.Records[addr].Record = lott.Records[addr].Record[0:0]
		delete(lott.Records, addr)
	}

	llog.Debug("checkDraw lottery switch to drawed")
	lott.LastTransToDrawState = action.height
	lott.Status = types.LotteryDrawed
	lott.TotalPurchasedTxNum = 0
	lott.LuckyNumber = luckynum

	if types.IsPara() {
		mainHeight := action.GetMainHeightByTxHash(action.txhash)
		if mainHeight < 0 {
			llog.Error("LotteryBuy", "mainHeight", mainHeight)
			return nil, types.ErrLotteryStatus
		}
		lott.LastTransToDrawStateOnMain = mainHeight
	}

	return &types.Receipt{types.ExecOk, kv, logs}, nil
}

func getManageKey(key string, db dbm.KV) ([]byte, error) {
	manageKey := types.ManageKey(key)
	value, err := db.Get([]byte(manageKey))
	if err != nil {
		return nil, err
	}
	return value, nil
}

func isRightCreator(addr string, db dbm.KV, isSolo bool) bool {
	if isSolo {
		return true
	} else {
		value, err := getManageKey(creatorKey, db)
		if err != nil {
			llog.Error("LotteryCreate", "creatorKey", creatorKey)
			return false
		}
		if value == nil {
			llog.Error("LotteryCreate found nil value")
			return false
		}

		var item types.ConfigItem
		err = types.Decode(value, &item)
		if err != nil {
			llog.Error("LotteryCreate", "Decode", value)
			return false
		}

		for _, op := range item.GetArr().Value {
			if op == addr {
				return true
			}
		}
		return false
	}
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

func ListLotteryLuckyHistory(db dbm.Lister, stateDB dbm.KV, param *types.ReqLotteryLuckyHistory) (types.Message, error) {
	direction := ListDESC
	if param.GetDirection() == ListASC {
		direction = ListASC
	}
	count := DefultCount
	if 0 < param.GetCount() && param.GetCount() <= MaxCount {
		count = param.GetCount()
	}
	var prefix []byte
	var key []byte
	var values [][]byte
	var err error

	prefix = calcLotteryDrawPrefix(param.LotteryId)
	key = calcLotteryDrawKey(param.LotteryId, param.GetRound())

	if param.GetRound() == 0 { //第一次查询
		values, err = db.List(prefix, nil, count, direction)
	} else {
		values, err = db.List(prefix, key, count, direction)
	}
	if err != nil {
		return nil, err
	}

	var records types.LotteryDrawRecords
	for _, value := range values {
		var record types.LotteryDrawRecord
		err := types.Decode(value, &record)
		if err != nil {
			continue
		}
		records.Records = append(records.Records, &record)
	}

	return &records, nil
}

func ListLotteryBuyRecords(db dbm.Lister, stateDB dbm.KV, param *types.ReqLotteryBuyHistory) (types.Message, error) {
	direction := ListDESC
	if param.GetDirection() == ListASC {
		direction = ListASC
	}
	count := DefultCount
	if 0 < param.GetCount() && param.GetCount() <= MaxCount {
		count = param.GetCount()
	}
	var prefix []byte
	var key []byte
	var values [][]byte
	var err error

	prefix = calcLotteryBuyPrefix(param.LotteryId, param.Addr)
	key = calcLotteryBuyRoundPrefix(param.LotteryId, param.Addr, param.GetRound())

	if param.GetRound() == 0 { //第一次查询
		values, err = db.List(prefix, nil, count, direction)
	} else {
		values, err = db.List(prefix, key, count, direction)
	}

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
