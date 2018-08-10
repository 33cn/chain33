package blackwhite

import (
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/types"
	gt "gitlab.33.cn/chain33/chain33/types/executor/blackwhite"
)

const (
	MaxAmount      int64 = 20 * types.Coin
	MinAmount      int64 = 1 * types.Coin
	minPlayerCount int32 = 3
	MaxMatchCount  int   = 10
	lockAmount     int64 = types.Coin / 100
)

type action struct {
	coinsAccount *account.DB
	db           dbm.KV
	txhash       []byte
	fromaddr     string
	blocktime    int64
	height       int64
	execaddr     string
}

type addressResultX struct {
	Addr    string
	IsWin   bool
	IsBlack []bool
}

func newAction(t *Blackwhite, tx *types.Transaction) *action {
	hash := tx.Hash()
	fromaddr := tx.From()
	return &action{t.GetCoinsAccount(), t.GetStateDB(), hash, fromaddr,
		t.GetBlockTime(), t.GetHeight(), t.GetAddr()}
}

func (a *action) Create(create *types.BlackwhiteCreate) (*types.Receipt, error) {
	if create.PlayAmount < MinAmount || create.PlayAmount > MaxAmount {
		return nil, types.ErrInputPara
	}
	if create.PlayerCount < minPlayerCount {
		return nil, types.ErrInputPara
	}

	receipt, err := a.coinsAccount.ExecFrozen(a.fromaddr, a.execaddr, lockAmount)
	if err != nil {
		clog.Error("blackwhite create ExecFrozen ", "addr", a.fromaddr, "execaddr", a.execaddr, "amount", lockAmount)
		return nil, err
	}

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue

	logs = append(logs, receipt.Logs...)
	kv = append(kv, receipt.KV...)

	round := newRound(create, a.fromaddr)

	round.GameID = common.ToHex(a.txhash)
	key := calcRoundKey(round.GameID)
	value := types.Encode(round)

	kv = append(kv, &types.KeyValue{key, value})

	log := &types.ReceiptBlackwhite{round}
	logs = append(logs, &types.ReceiptLog{types.TyLogBlackwhiteCreate, types.Encode(log)})

	return &types.Receipt{types.ExecOk, kv, logs}, nil
}

func (a *action) Play(play *types.BlackwhitePlay) (*types.Receipt, error) {
	// 获取GameID
	value, err := a.db.Get(calcRoundKey(play.GameID))
	if err != nil {
		clog.Error("blackwhite play ", "addr", a.fromaddr, "execaddr", a.execaddr, "get round failed",
			play.GameID, "err", err)
		return nil, err
	}
	var round types.BlackwhiteRound
	err = types.Decode(value, &round)
	if err != nil {
		clog.Error("blackwhite play ", "addr", a.fromaddr, "execaddr", a.execaddr, "decode round failed",
			play.GameID, "err", err)
		return nil, err
	}

	// 检查当前状态
	if gt.BlackwhiteStatusPlay != round.Status && gt.BlackwhiteStatusReady != round.Status {
		err := types.ErrGameOver
		clog.Error("blackwhite play ", "addr", a.fromaddr, "round status", round.Status, "GameID ",
			play.GameID, "err", err)
		return nil, err
	}

	// 检查是否有重复
	for _, addrResult := range round.AddrResult {
		if addrResult.Addr == a.fromaddr {
			err := types.ErrOnceRoundRepeatPlay
			clog.Error("blackwhite play ", "addr", a.fromaddr, "execaddr", a.execaddr, "Repeat GameID",
				play.GameID, "err", err)
			return nil, err
		}
	}

	if round.PlayAmount != play.Amount {
		clog.Error("blackwhite play ", "addr", a.fromaddr, "execaddr", a.execaddr, "have not same amount in once round",
			play.GameID)
		return nil, types.ErrInputPara
	}

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue

	receipt, err := a.coinsAccount.ExecFrozen(a.fromaddr, a.execaddr, play.Amount)
	if err != nil {
		clog.Error("blackwhite Play", "addr", a.fromaddr, "execaddr", a.execaddr, "amount", play.Amount)
		return nil, err
	}
	logs = append(logs, receipt.Logs...)
	kv = append(kv, receipt.KV...)

	round.Status = gt.BlackwhiteStatusPlay
	addrRes := &types.AddressResult{
		Addr:    a.fromaddr,
		IsBlack: play.IsBlack,
	}
	round.AddrResult = append(round.AddrResult, addrRes)
	round.CurPlayerCount++

	if round.CurPlayerCount >= round.PlayerCount {
		// 触发开奖
		round.Status = gt.BlackwhiteStatusDone
		receipt, err := a.StatTransfer(&round)
		if err != nil {
			clog.Error("blackwhite done fail", "StatTransfer err", err)
			return nil, err
		}
		logs = append(logs, receipt.Logs...)
		kv = append(kv, receipt.KV...)
	} else {
		log := &types.ReceiptBlackwhite{&round}
		logs = append(logs, &types.ReceiptLog{types.TyLogBlackwhitePlay, types.Encode(log)})
	}

	key1 := calcRoundKey(round.GameID)
	value1 := types.Encode(&round)
	kv = append(kv, &types.KeyValue{key1, value1})

	return &types.Receipt{types.ExecOk, kv, logs}, nil
}

func (a *action) Cancel(cancel *types.BlackwhiteCancel) (*types.Receipt, error) {
	value, err := a.db.Get(calcRoundKey(cancel.GameID))
	if err != nil {
		clog.Error("blackwhite cancel ", "addr", a.fromaddr, "execaddr", a.execaddr, "get round failed",
			cancel.GameID, "err", err)
		return nil, err
	}

	var round types.BlackwhiteRound
	err = types.Decode(value, &round)
	if err != nil {
		clog.Error("blackwhite cancel ", "addr", a.fromaddr, "execaddr", a.execaddr, "decode round failed",
			cancel.GameID, "err", err)
		return nil, err
	}

	// 检查当前状态
	if gt.BlackwhiteStatusReady != round.Status {
		err := types.ErrNoCancel
		clog.Error("blackwhite cancel ", "addr", a.fromaddr, "execaddr", a.execaddr, "GameID Status",
			cancel.GameID, "err", err)
		return nil, err
	}

	if round.CreateAddr != a.fromaddr {
		err := types.ErrNoCancel
		clog.Error("blackwhite cancel ", "addr", a.fromaddr, "execaddr", a.execaddr, "not create addr,can not cancel ",
			cancel.GameID, "err", err)
		return nil, err
	}

	round.Status = gt.BlackwhiteStatusCancel

	key1 := calcRoundKey(cancel.GameID)
	value1 := types.Encode(&round)
	//order.save(a.db, key, value)

	var kv []*types.KeyValue

	kv = append(kv, &types.KeyValue{key1, value1})

	return &types.Receipt{types.ExecOk, kv, nil}, nil
}

func (a *action) TimeoutDone(done *types.BlackwhiteTimeoutDone) (*types.Receipt, error) {
	value, err := a.db.Get(calcRoundKey(done.GameID))
	if err != nil {
		clog.Error("blackwhite timeout done ", "addr", a.fromaddr, "execaddr", a.execaddr, "get round failed",
			done.GameID, "err", err)
		return nil, err
	}

	var round types.BlackwhiteRound
	err = types.Decode(value, &round)
	if err != nil {
		clog.Error("blackwhite timeout done ", "addr", a.fromaddr, "execaddr", a.execaddr, "decode round failed",
			done.GameID, "err", err)
		return nil, err
	}

	// 检查当前状态
	if gt.BlackwhiteStatusPlay != round.Status{
		err := types.ErrTimeout
		clog.Error("blackwhite timeout done ", "addr", a.fromaddr, "execaddr", a.execaddr, "GameID",
			done.GameID, "Status", round.Status, "err", err)
		return nil, err
	}

	if round.CreateAddr != a.fromaddr {
		err := types.ErrTimeout
		clog.Error("blackwhite timeout done ", "addr", a.fromaddr, "execaddr", a.execaddr, "not create addr,can not cancel ",
			done.GameID, "err", err)
		return nil, err
	}

	round.Status = gt.BlackwhiteStatusTimeoutDone
	receipt, err := a.StatTransfer(&round)
	if err != nil {
		clog.Error("blackwhite timeout done ", "StatTransfer err", err)
		return nil, err
	}

	var kv []*types.KeyValue
	var logs []*types.ReceiptLog

	logs = append(logs, receipt.Logs...)
	kv = append(kv, receipt.KV...)

	return &types.Receipt{types.ExecOk, kv, logs}, nil
}

func (a *action) StatTransfer(round *types.BlackwhiteRound) (*types.Receipt, error) {

	winers := a.getWinner(round.AddrResult)

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue

	roundAmount := round.PlayAmount

	if len(winers) == 0 {
		// 将所有参与人员都解冻
		for _, addrRes := range round.AddrResult {
			receipt, err := a.coinsAccount.ExecActive(addrRes.Addr, a.execaddr, roundAmount)
			if err != nil {
				clog.Error("StatTransfer execActive no winers", "addr", a.fromaddr, "execaddr", a.execaddr, "err", err)
				return nil, err
			}
			logs = append(logs, receipt.Logs...)
			kv = append(kv, receipt.KV...)
		}
	} else {
		Losers := a.getLoser(round.AddrResult)
		sumAmount := roundAmount * int64(len(Losers))

		var winNum int64
		winNum = int64(len(winers))
		averAmount := sumAmount / winNum

		//先将所有loster的金额转到其中一个赢家中
		winer := winers[0]
		for _, Loser := range Losers {
			receipt, err := a.coinsAccount.ExecTransferFrozen(Loser, winer, a.execaddr, roundAmount)
			if err != nil {
				a.coinsAccount.ExecFrozen(winer, a.execaddr, roundAmount) // rollback
				clog.Error("StatTransfer all losers to once winer fail", "addr", winer, "execaddr", a.execaddr, "err", err)
				return nil, err
			}
			logs = append(logs, receipt.Logs...)
			kv = append(kv, receipt.KV...)
		}

		// 从其中一个赢家中转帐给它获胜用户
		for i, winer := range winers {
			if i != 0 {
				receipt, err := a.coinsAccount.ExecTransfer(winers[0], winer, a.execaddr, averAmount)
				if err != nil {
					a.coinsAccount.ExecFrozen(winers[0], a.execaddr, roundAmount) // rollback
					clog.Error("StatTransfer one winer to any other winers fail", "addr", winer, "execaddr", a.execaddr, "err", err)
					return nil, err
				}
				logs = append(logs, receipt.Logs...)
				kv = append(kv, receipt.KV...)
			}
		}

		// 胜利人员都解冻
		for _, winer := range winers {
			receipt, err := a.coinsAccount.ExecActive(winer, a.execaddr, roundAmount)
			if err != nil {
				clog.Error("StatTransfer execActive have winers", "addr", a.fromaddr, "execaddr", a.execaddr, "err", err)
				return nil, err
			}
			logs = append(logs, receipt.Logs...)
			kv = append(kv, receipt.KV...)
		}
	}

	for _, winer := range winers {
		round.Winner = append(round.Winner, winer)
	}

	// 将创建游戏者解冻
	receipt, err := a.coinsAccount.ExecActive(round.CreateAddr, a.execaddr, lockAmount)
	if err != nil {
		a.coinsAccount.ExecFrozen(round.CreateAddr, a.execaddr, lockAmount) // rollback
		clog.Error("blackwhite ExecActive create ExecFrozen ", "addr", round.CreateAddr, "execaddr", a.execaddr, "amount", lockAmount)
		return nil, err
	}
	logs = append(logs, receipt.Logs...)
	kv = append(kv, receipt.KV...)

	log := &types.ReceiptBlackwhite{round}
	if gt.BlackwhiteStatusTimeoutDone == round.Status {
		logs = append(logs, &types.ReceiptLog{types.TyLogBlackwhiteTimeoutDone, types.Encode(log)})
	} else {
		logs = append(logs, &types.ReceiptLog{types.TyLogBlackwhiteDone, types.Encode(log)})
	}

	return &types.Receipt{types.ExecOk, kv, logs}, nil
}

func (a *action) getWinner(addrRes []*types.AddressResult) []string {

	var addresXs []*addressResultX

	for _, addres := range addrRes {
		lenth := len(addres.IsBlack)
		if lenth < MaxMatchCount {
			for i := 0; i < (MaxMatchCount - lenth); i++ {
				addres.IsBlack = append(addres.IsBlack, false)
			}
		}

		addresX := &addressResultX{
			Addr:    addres.Addr,
			IsWin:   true,
			IsBlack: addres.IsBlack,
		}
		addresXs = append(addresXs, addresX)
	}

	for index := 0; index < MaxMatchCount; index++ {
		blackNum := 0
		whiteNUm := 0
		curWinNum := 0

		for _, addres := range addresXs {
			if addres.IsWin {
				curWinNum++
			}
		}

		if 0 == curWinNum {
			for _, addres := range addresXs {
				addres.IsWin = true
			}
		}

		for _, addr := range addresXs {
			if addr.IsWin {
				if addr.IsBlack[index] {
					blackNum++
				} else {
					whiteNUm++
				}
			}
		}

		if blackNum < whiteNUm {
			for _, addr := range addresXs {
				if addr.IsBlack[index] {
					addr.IsWin = true
				} else {
					addr.IsWin = false
				}
			}
		} else if blackNum > whiteNUm {
			for _, addr := range addresXs {
				if addr.IsBlack[index] {
					addr.IsWin = false
				} else {
					addr.IsWin = true
				}
			}
		}

		winNum := 0
		for _, addr := range addresXs {
			if addr.IsWin {
				winNum++
			}
		}

		if 1 == winNum {
			break
		}
	}

	var results []string
	for _, addr := range addresXs {
		if addr.IsWin {
			result := addr.Addr
			results = append(results, result)
		}
	}

	return results
}

func (a *action) getLoser(addrRes []*types.AddressResult) []string {

	wins := a.getWinner(addrRes)

	addMap := make(map[string]bool)
	for _, win := range wins {
		addMap[win] = true
	}

	var results []string
	for _, addr := range addrRes {
		if ok := addMap[addr.Addr]; !ok {
			result := addr.Addr
			results = append(results, result)
		}
	}

	return results
}
