package executor

import (
	"errors"
	"fmt"
	"sort"
	"strconv"

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	pkt "gitlab.33.cn/chain33/chain33/plugin/dapp/pokerbull/types"
	"gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
)

const (
	ListDESC = int32(0)
	ListASC  = int32(1)

	DefaultCount   = int32(20) //默认一次取多少条记录
	MAX_PLAYER_NUM = 5
	MIN_PLAY_VALUE = 10 * types.Coin
	DefaultStyle   = pkt.PlayStyleDefault
)

type Action struct {
	coinsAccount *account.DB
	db           dbm.KV
	txhash       []byte
	fromaddr     string
	blocktime    int64
	height       int64
	execaddr     string
	localDB      dbm.Lister
	index        int
}

func NewAction(pb *PokerBull, tx *types.Transaction, index int) *Action {
	hash := tx.Hash()
	fromaddr := tx.From()

	return &Action{pb.GetCoinsAccount(), pb.GetStateDB(), hash, fromaddr,
		pb.GetBlockTime(), pb.GetHeight(), dapp.ExecAddress(string(tx.Execer)), pb.GetLocalDB(), index}
}

func (action *Action) CheckExecAccountBalance(fromAddr string, ToFrozen, ToActive int64) bool {
	// 赌注为零，按照最小赌注冻结
	if ToFrozen == 0 {
		ToFrozen = MIN_PLAY_VALUE
	}

	acc := action.coinsAccount.LoadExecAccount(fromAddr, action.execaddr)
	if acc.GetBalance() >= ToFrozen && acc.GetFrozen() >= ToActive {
		return true
	}
	return false
}

func Key(id string) (key []byte) {
	key = append(key, []byte("mavl-"+types.ExecName(pkt.PokerBullX)+"-")...)
	key = append(key, []byte(id)...)
	return key
}

func readGame(db dbm.KV, id string) (*pkt.PokerBull, error) {
	data, err := db.Get(Key(id))
	if err != nil {
		logger.Error("query data have err:", err.Error())
		return nil, err
	}
	var game pkt.PokerBull
	//decode
	err = types.Decode(data, &game)
	if err != nil {
		logger.Error("decode game have err:", err.Error())
		return nil, err
	}
	return &game, nil
}

//安全批量查询方式,防止因为脏数据导致查询接口奔溃
func GetGameList(db dbm.KV, values []string) []*pkt.PokerBull {
	var games []*pkt.PokerBull
	for _, value := range values {
		game, err := readGame(db, value)
		if err != nil {
			continue
		}
		games = append(games, game)
	}
	return games
}

func Infos(db dbm.KV, infos *pkt.QueryPBGameInfos) (types.Message, error) {
	var games []*pkt.PokerBull
	for i := 0; i < len(infos.GameIds); i++ {
		id := infos.GameIds[i]
		game, err := readGame(db, id)
		if err != nil {
			return nil, err
		}
		games = append(games, game)
	}
	return &pkt.ReplyPBGameList{Games: games}, nil
}

func getGameListByAddr(db dbm.Lister, addr string, index int64) (types.Message, error) {
	var values [][]byte
	var err error
	if index == 0 {
		values, err = db.List(calcPBGameAddrPrefix(addr), nil, DefaultCount, ListDESC)
	} else {
		values, err = db.List(calcPBGameAddrPrefix(addr), calcPBGameAddrKey(addr, index), DefaultCount, ListDESC)
	}
	if err != nil {
		return nil, err
	}

	var gameIds []*pkt.PBGameRecord
	for _, value := range values {
		var record pkt.PBGameRecord
		err := types.Decode(value, &record)
		if err != nil {
			continue
		}
		gameIds = append(gameIds, &record)
	}

	return &pkt.PBGameRecords{gameIds}, nil
}

func getGameListByStatus(db dbm.Lister, status int32, index int64) (types.Message, error) {
	var values [][]byte
	var err error
	if index == 0 {
		values, err = db.List(calcPBGameStatusPrefix(status), nil, DefaultCount, ListDESC)
	} else {
		values, err = db.List(calcPBGameStatusPrefix(status), calcPBGameStatusKey(status, index), DefaultCount, ListDESC)
	}
	if err != nil {
		return nil, err
	}

	var gameIds []*pkt.PBGameRecord
	for _, value := range values {
		var record pkt.PBGameRecord
		err := types.Decode(value, &record)
		if err != nil {
			continue
		}
		gameIds = append(gameIds, &record)
	}

	return &pkt.PBGameRecords{gameIds}, nil
}

func queryGameListByStatusAndPlayer(db dbm.Lister, stat int32, player int32, value int64) ([]string, error) {
	values, err := db.List(calcPBGameStatusAndPlayerPrefix(stat, player, value), nil, DefaultCount, ListDESC)
	if err != nil {
		return nil, err
	}

	var gameIds []string
	for _, value := range values {
		var record pkt.PBGameIndexRecord
		err := types.Decode(value, &record)
		if err != nil {
			continue
		}
		gameIds = append(gameIds, record.GetGameId())
	}

	return gameIds, nil
}

func (action *Action) saveGame(game *pkt.PokerBull) (kvset []*types.KeyValue) {
	value := types.Encode(game)
	action.db.Set(Key(game.GetGameId()), value)
	kvset = append(kvset, &types.KeyValue{Key(game.GameId), value})
	return kvset
}

func (action *Action) getIndex(game *pkt.PokerBull) int64 {
	return action.height*types.MaxTxsPerBlock + int64(action.index)
}

func (action *Action) GetReceiptLog(game *pkt.PokerBull) *types.ReceiptLog {
	log := &types.ReceiptLog{}
	r := &pkt.ReceiptPBGame{}
	r.Addr = action.fromaddr
	if game.Status == pkt.PBGameActionStart {
		log.Ty = pkt.TyLogPBGameStart
	} else if game.Status == pkt.PBGameActionContinue {
		log.Ty = pkt.TyLogPBGameContinue
	} else if game.Status == pkt.PBGameActionQuit {
		log.Ty = pkt.TyLogPBGameQuit
	}

	r.GameId = game.GameId
	r.Status = game.Status
	r.Index = game.GetIndex()
	r.PrevIndex = game.GetPrevIndex()
	r.PlayerNum = game.PlayerNum
	r.Value = game.Value
	r.IsWaiting = game.IsWaiting
	if !r.IsWaiting {
		for _, v := range game.Players {
			r.Players = append(r.Players, v.Address)
		}
	}
	r.PreStatus = game.PreStatus
	log.Log = types.Encode(r)
	return log
}

func (action *Action) readGame(id string) (*pkt.PokerBull, error) {
	data, err := action.db.Get(Key(id))
	if err != nil {
		return nil, err
	}
	var game pkt.PokerBull
	//decode
	err = types.Decode(data, &game)
	if err != nil {
		return nil, err
	}
	return &game, nil
}

func (action *Action) calculate(game *pkt.PokerBull) *pkt.PBResult {
	var handS HandSlice
	for _, player := range game.Players {
		hand := &pkt.PBHand{}
		hand.Cards = Deal(game.Poker, player.TxHash) //发牌
		hand.Result = Result(hand.Cards)             //计算结果
		hand.Address = player.Address

		//存入玩家数组
		player.Hands = append(player.Hands, hand)

		//存入临时切片待比大小排序
		handS = append(handS, hand)

		//为下一个continue状态初始化player
		player.Ready = false
	}

	// 升序排列
	if !sort.IsSorted(handS) {
		sort.Sort(handS)
	}
	winner := handS[len(handS)-1]

	// 将有序的临时切片加入到结果数组
	result := &pkt.PBResult{}
	result.Winner = winner.Address
	//TODO Dealer:暂时不支持倍数
	//result.Leverage = Leverage(winner)
	result.Hands = make([]*pkt.PBHand, len(handS))
	copy(result.Hands, handS)

	game.Results = append(game.Results, result)
	return result
}

func (action *Action) calculateDealer(game *pkt.PokerBull) *pkt.PBResult {
	var handS HandSlice
	var dealer *pkt.PBHand
	for _, player := range game.Players {
		hand := &pkt.PBHand{}
		hand.Cards = Deal(game.Poker, player.TxHash) //发牌
		hand.Result = Result(hand.Cards)             //计算结果
		hand.Address = player.Address

		//存入玩家数组
		player.Hands = append(player.Hands, hand)

		//存入临时切片待比大小排序
		handS = append(handS, hand)

		//为下一个continue状态初始化player
		player.Ready = false

		//记录庄家
		if player.Address == game.DealerAddr {
			dealer = hand
		}
	}

	for _, hand := range handS {
		if hand.Address == game.DealerAddr {
			continue
		}

		if CompareResult(hand, dealer) {
			hand.IsWin = false
		} else {
			hand.IsWin = true
			hand.Leverage = Leverage(hand)
		}
	}

	// 将有序的临时切片加入到结果数组
	result := &pkt.PBResult{}
	result.Dealer = game.DealerAddr
	result.DealerLeverage = Leverage(dealer)
	result.Hands = make([]*pkt.PBHand, len(handS))
	copy(result.Hands, handS)

	game.Results = append(game.Results, result)
	return result
}

func (action *Action) nextDealer(game *pkt.PokerBull) string {
	var flag = -1
	for i, player := range game.Players {
		if player.Address == game.DealerAddr {
			flag = i
		}
	}
	if flag == -1 {
		logger.Error("Get next dealer failed.")
		return game.DealerAddr
	}

	if flag == len(game.Players)-1 {
		return game.Players[0].Address
	}

	return game.Players[flag+1].Address
}

func (action *Action) settleDealerAccount(lastAddress string, game *pkt.PokerBull) ([]*types.ReceiptLog, []*types.KeyValue, error) {
	var logs []*types.ReceiptLog
	var kv []*types.KeyValue

	result := action.calculateDealer(game)
	for _, hand := range result.Hands {
		// 最后一名玩家没有冻结
		if hand.Address != lastAddress {
			receipt, err := action.coinsAccount.ExecActive(hand.Address, action.execaddr, game.GetValue()*POKERBULL_LEVERAGE_MAX)
			if err != nil {
				logger.Error("GameSettleDealer.ExecActive", "addr", hand.Address, "execaddr", action.execaddr, "amount", game.GetValue(),
					"err", err)
				return nil, nil, err
			}
			logs = append(logs, receipt.Logs...)
			kv = append(kv, receipt.KV...)
		}

		//给赢家转账
		var receipt *types.Receipt
		var err error
		if hand.Address != result.Dealer {
			if hand.IsWin {
				receipt, err = action.coinsAccount.ExecTransfer(result.Dealer, hand.Address, action.execaddr, game.GetValue()*int64(hand.Leverage))
				if err != nil {
					action.coinsAccount.ExecFrozen(hand.Address, action.execaddr, game.GetValue()) // rollback
					logger.Error("GameSettleDealer.ExecTransfer", "addr", hand.Address, "execaddr", action.execaddr,
						"amount", game.GetValue()*int64(hand.Leverage), "err", err)
					return nil, nil, err
				}
			} else {
				receipt, err = action.coinsAccount.ExecTransfer(hand.Address, result.Dealer, action.execaddr, game.GetValue()*int64(result.DealerLeverage))
				if err != nil {
					action.coinsAccount.ExecFrozen(hand.Address, action.execaddr, game.GetValue()) // rollback
					logger.Error("GameSettleDealer.ExecTransfer", "addr", hand.Address, "execaddr", action.execaddr,
						"amount", game.GetValue()*int64(result.DealerLeverage), "err", err)
					return nil, nil, err
				}
			}
			logs = append(logs, receipt.Logs...)
			kv = append(kv, receipt.KV...)
		}
	}
	game.DealerAddr = action.nextDealer(game)

	return logs, kv, nil
}

func (action *Action) settleDefaultAccount(lastAddress string, game *pkt.PokerBull) ([]*types.ReceiptLog, []*types.KeyValue, error) {
	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	result := action.calculate(game)

	for _, player := range game.Players {
		// 最后一名玩家没有冻结
		if player.Address != lastAddress {
			receipt, err := action.coinsAccount.ExecActive(player.GetAddress(), action.execaddr, game.GetValue()*POKERBULL_LEVERAGE_MAX)
			if err != nil {
				logger.Error("GameSettleDefault.ExecActive", "addr", player.GetAddress(), "execaddr", action.execaddr,
					"amount", game.GetValue()*POKERBULL_LEVERAGE_MAX, "err", err)
				return nil, nil, err
			}
			logs = append(logs, receipt.Logs...)
			kv = append(kv, receipt.KV...)
		}

		//给赢家转账
		if player.Address != result.Winner {
			receipt, err := action.coinsAccount.ExecTransfer(player.Address, result.Winner, action.execaddr, game.GetValue() /**int64(result.Leverage)*/) //TODO Dealer:暂时不支持倍数
			if err != nil {
				action.coinsAccount.ExecFrozen(result.Winner, action.execaddr, game.GetValue()) // rollback
				logger.Error("GameSettleDefault.ExecTransfer", "addr", result.Winner, "execaddr", action.execaddr,
					"amount", game.GetValue() /**int64(result.Leverage)*/, "err", err) //TODO Dealer:暂时不支持倍数
				return nil, nil, err
			}
			logs = append(logs, receipt.Logs...)
			kv = append(kv, receipt.KV...)
		}
	}

	return logs, kv, nil
}

func (action *Action) settleAccount(lastAddress string, game *pkt.PokerBull) ([]*types.ReceiptLog, []*types.KeyValue, error) {
	if DefaultStyle == pkt.PlayStyleDealer {
		return action.settleDealerAccount(lastAddress, game)
	} else {
		return action.settleDefaultAccount(lastAddress, game)
	}
}

func (action *Action) genTxRnd(txhash []byte) (int64, error) {
	randbyte := make([]byte, 7)
	for i := 0; i < 7; i++ {
		randbyte[i] = txhash[i]
	}

	randstr := common.ToHex(randbyte)
	randint, err := strconv.ParseInt(randstr, 0, 64)
	if err != nil {
		return 0, err
	}

	return randint, nil
}

func (action *Action) checkDupPlayerAddress(id string, pbPlayers []*pkt.PBPlayer) error {
	for _, player := range pbPlayers {
		if action.fromaddr == player.Address {
			logger.Error("Poker bull game start", "addr", action.fromaddr, "execaddr", action.execaddr, "Already in a game", id)
			return errors.New("Address is already in a game")
		}
	}

	return nil
}

// 新建一局游戏
func (action *Action) newGame(gameId string, start *pkt.PBGameStart) (*pkt.PokerBull, error) {
	var game *pkt.PokerBull

	// 不指定赌注，默认按照最低赌注
	if start.GetValue() == 0 {
		start.Value = MIN_PLAY_VALUE
	}

	//TODO 庄家检查闲家数量倍数的资金
	if DefaultStyle == pkt.PlayStyleDealer {
		if !action.CheckExecAccountBalance(action.fromaddr, start.GetValue()*POKERBULL_LEVERAGE_MAX*int64(start.PlayerNum-1), 0) {
			logger.Error("GameStart", "addr", action.fromaddr, "execaddr", action.execaddr, "id",
				gameId, "err", types.ErrNoBalance)
			return nil, types.ErrNoBalance
		}
	}

	game = &pkt.PokerBull{
		GameId:      gameId,
		Status:      pkt.PBGameActionStart,
		StartTime:   action.blocktime,
		StartTxHash: gameId,
		Value:       start.GetValue(),
		Poker:       NewPoker(),
		PlayerNum:   start.PlayerNum,
		Index:       action.getIndex(game),
		DealerAddr:  action.fromaddr,
		IsWaiting:   true,
		PreStatus:   0,
	}

	Shuffle(game.Poker, action.blocktime) //洗牌

	return game, nil
}

// 筛选合适的牌局
func (action *Action) selectGameFromIds(ids []string, value int64) *pkt.PokerBull {
	var gameRet *pkt.PokerBull = nil
	for _, id := range ids {
		game, err := action.readGame(id)
		if err != nil {
			logger.Error("Poker bull game start", "addr", action.fromaddr, "execaddr", action.execaddr, "get game failed", id, "err", err)
			continue
		}

		//不能自己和自己玩
		if action.checkDupPlayerAddress(id, game.Players) != nil {
			continue
		}

		//选择合适赌注的游戏
		if value == 0 && game.GetValue() != MIN_PLAY_VALUE {
			if !action.CheckExecAccountBalance(action.fromaddr, game.GetValue(), 0) {
				logger.Error("GameStart", "addr", action.fromaddr, "execaddr", action.execaddr, "id", id, "err", types.ErrNoBalance)
				continue
			}
		}

		gameRet = game
		break
	}
	return gameRet
}

func (action *Action) checkPlayerExistInGame() bool {
	values, err := action.localDB.List(calcPBGameAddrPrefix(action.fromaddr), nil, DefaultCount, ListDESC)
	if err == types.ErrNotFound {
		return false
	}

	var value pkt.PBGameRecord
	lenght := len(values)
	if lenght != 0 {
		valueBytes := values[lenght-1]
		err := types.Decode(valueBytes, &value)
		if err == nil && value.Status == pkt.PBGameActionQuit {
			return false
		}
	}
	return true
}

func (action *Action) GameStart(start *pkt.PBGameStart) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kv []*types.KeyValue

	if start.PlayerNum > MAX_PLAYER_NUM {
		logger.Error("GameStart", "addr", action.fromaddr, "execaddr", action.execaddr,
			"err", fmt.Sprintf("The maximum player number is %d", MAX_PLAYER_NUM))
		return nil, types.ErrInvalidParam
	}

	gameId := common.ToHex(action.txhash)
	if !action.CheckExecAccountBalance(action.fromaddr, start.GetValue()*POKERBULL_LEVERAGE_MAX, 0) {
		logger.Error("GameStart", "addr", action.fromaddr, "execaddr", action.execaddr, "id", gameId, "err", types.ErrNoBalance)
		return nil, types.ErrNoBalance
	}

	if action.checkPlayerExistInGame() {
		logger.Error("GameStart", "addr", action.fromaddr, "execaddr", action.execaddr, "err", "Address is already in a game")
		return nil, fmt.Errorf("Address is already in a game")
	}

	var game *pkt.PokerBull = nil
	ids, err := queryGameListByStatusAndPlayer(action.localDB, pkt.PBGameActionStart, start.PlayerNum, start.Value)
	if err != nil || len(ids) == 0 {
		if err != types.ErrNotFound {
			return nil, err
		}

		game, err = action.newGame(gameId, start)
		if err != nil {
			return nil, err
		}
	} else {
		game = action.selectGameFromIds(ids, start.GetValue())
		if game == nil {
			// 如果也没有匹配到游戏，则按照最低赌注创建
			game, err = action.newGame(gameId, start)
			if err != nil {
				return nil, err
			}
		}
	}

	//发牌随机数取txhash
	txrng, err := action.genTxRnd(action.txhash)
	if err != nil {
		return nil, err
	}

	//加入当前玩家信息
	game.Players = append(game.Players, &pkt.PBPlayer{
		Address: action.fromaddr,
		TxHash:  txrng,
		Ready:   false,
	})

	// 如果人数达标，则发牌计算斗牛结果
	if len(game.Players) == int(game.PlayerNum) {
		logsH, kvH, err := action.settleAccount(action.fromaddr, game)
		if err != nil {
			return nil, err
		}
		logs = append(logs, logsH...)
		kv = append(kv, kvH...)

		game.PrevIndex = game.Index
		game.Index = action.getIndex(game)
		game.Status = pkt.PBGameActionContinue // 更新游戏状态
		game.PreStatus = pkt.PBGameActionStart
		game.IsWaiting = false
	} else {
		receipt, err := action.coinsAccount.ExecFrozen(action.fromaddr, action.execaddr, start.GetValue()*POKERBULL_LEVERAGE_MAX) //冻结子账户资金, 最后一位玩家不需要冻结
		if err != nil {
			logger.Error("GameCreate.ExecFrozen", "addr", action.fromaddr, "execaddr", action.execaddr, "amount", start.GetValue(), "err", err.Error())
			return nil, err
		}
		logs = append(logs, receipt.Logs...)
		kv = append(kv, receipt.KV...)
	}
	receiptLog := action.GetReceiptLog(game)
	logs = append(logs, receiptLog)
	kv = append(kv, action.saveGame(game)...)

	return &types.Receipt{types.ExecOk, kv, logs}, nil
}

func getReadyPlayerNum(players []*pkt.PBPlayer) int {
	var readyC = 0
	for _, player := range players {
		if player.Ready {
			readyC++
		}
	}
	return readyC
}

func getPlayerFromAddress(players []*pkt.PBPlayer, addr string) *pkt.PBPlayer {
	for _, player := range players {
		if player.Address == addr {
			return player
		}
	}
	return nil
}

func (action *Action) GameContinue(pbcontinue *pkt.PBGameContinue) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kv []*types.KeyValue

	game, err := action.readGame(pbcontinue.GetGameId())
	if err != nil {
		logger.Error("GameContinue", "addr", action.fromaddr, "execaddr", action.execaddr, "get game failed",
			pbcontinue.GetGameId(), "err", err)
		return nil, err
	}

	if game.Status != pkt.PBGameActionContinue {
		logger.Error("GameContinue", "addr", action.fromaddr, "execaddr", action.execaddr, "Status error",
			pbcontinue.GetGameId())
		return nil, err
	}

	// 检查余额，庄家检查闲家数量倍数的资金
	checkValue := game.GetValue() * POKERBULL_LEVERAGE_MAX
	if action.fromaddr == game.DealerAddr {
		checkValue = checkValue * int64(game.PlayerNum-1)
	}
	if !action.CheckExecAccountBalance(action.fromaddr, checkValue, 0) {
		logger.Error("GameContinue", "addr", action.fromaddr, "execaddr", action.execaddr, "id",
			pbcontinue.GetGameId(), "err", types.ErrNoBalance)
		return nil, types.ErrNoBalance
	}

	// 寻找对应玩家
	pbplayer := getPlayerFromAddress(game.Players, action.fromaddr)
	if pbplayer == nil {
		logger.Error("GameContinue", "addr", action.fromaddr, "execaddr", action.execaddr, "get game player failed",
			pbcontinue.GetGameId(), "err", types.ErrNotFound)
		return nil, types.ErrNotFound
	}
	if pbplayer.Ready {
		logger.Error("GameContinue", "addr", action.fromaddr, "execaddr", action.execaddr, "player has been ready",
			pbcontinue.GetGameId(), "player", pbplayer.Address)
		return nil, fmt.Errorf("player %s has been ready", pbplayer.Address)
	}

	//发牌随机数取txhash
	txrng, err := action.genTxRnd(action.txhash)
	if err != nil {
		return nil, err
	}
	pbplayer.TxHash = txrng
	pbplayer.Ready = true

	if getReadyPlayerNum(game.Players) == int(game.PlayerNum) {
		logsH, kvH, err := action.settleAccount(action.fromaddr, game)
		if err != nil {
			return nil, err
		}
		logs = append(logs, logsH...)
		kv = append(kv, kvH...)
		game.PrevIndex = game.Index
		game.Index = action.getIndex(game)
		game.IsWaiting = false
		game.PreStatus = pkt.PBGameActionContinue
	} else {
		receipt, err := action.coinsAccount.ExecFrozen(action.fromaddr, action.execaddr, game.GetValue()*POKERBULL_LEVERAGE_MAX) //冻结子账户资金,最后一位玩家不需要冻结
		if err != nil {
			logger.Error("GameCreate.ExecFrozen", "addr", action.fromaddr, "execaddr", action.execaddr, "amount", game.GetValue(), "err", err.Error())
			return nil, err
		}
		logs = append(logs, receipt.Logs...)
		kv = append(kv, receipt.KV...)
		game.IsWaiting = true
	}

	receiptLog := action.GetReceiptLog(game)
	logs = append(logs, receiptLog)
	kv = append(kv, action.saveGame(game)...)

	return &types.Receipt{types.ExecOk, kv, logs}, nil
}

func (action *Action) GameQuit(pbend *pkt.PBGameQuit) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kv []*types.KeyValue

	game, err := action.readGame(pbend.GetGameId())
	if err != nil {
		logger.Error("GameEnd", "addr", action.fromaddr, "execaddr", action.execaddr, "get game failed",
			pbend.GetGameId(), "err", err)
		return nil, err
	}

	// 如果游戏没有开始，激活冻结账户
	if game.IsWaiting {
		if game.Status == pkt.PBGameActionStart {
			for _, player := range game.Players {
				receipt, err := action.coinsAccount.ExecActive(player.Address, action.execaddr, game.GetValue()*POKERBULL_LEVERAGE_MAX)
				if err != nil {
					logger.Error("GameSettleDealer.ExecActive", "addr", player.Address, "execaddr", action.execaddr, "amount", game.GetValue(),
						"err", err)
					continue
				}
				logs = append(logs, receipt.Logs...)
				kv = append(kv, receipt.KV...)
			}
		} else if game.Status == pkt.PBGameActionContinue {
			for _, player := range game.Players {
				if !player.Ready {
					continue
				}

				receipt, err := action.coinsAccount.ExecActive(player.Address, action.execaddr, game.GetValue()*POKERBULL_LEVERAGE_MAX)
				if err != nil {
					logger.Error("GameSettleDealer.ExecActive", "addr", player.Address, "execaddr", action.execaddr, "amount", game.GetValue(),
						"err", err)
					continue
				}
				logs = append(logs, receipt.Logs...)
				kv = append(kv, receipt.KV...)
			}
		}
		game.IsWaiting = false
	}
	game.PreStatus = game.Status
	game.Status = pkt.PBGameActionQuit
	game.PrevIndex = game.Index
	game.Index = action.getIndex(game)
	game.QuitTime = action.blocktime
	game.QuitTxHash = common.ToHex(action.txhash)

	receiptLog := action.GetReceiptLog(game)
	logs = append(logs, receiptLog)
	kv = append(kv, action.saveGame(game)...)
	return &types.Receipt{types.ExecOk, kv, logs}, nil
}

type HandSlice []*pkt.PBHand

func (h HandSlice) Len() int {
	return len(h)
}

func (h HandSlice) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h HandSlice) Less(i, j int) bool {
	if i >= h.Len() || j >= h.Len() {
		logger.Error("length error. slice length:", h.Len(), " compare lenth: ", i, " ", j)
	}

	if h[i] == nil || h[j] == nil {
		logger.Error("nil pointer at ", i, " ", j)
	}

	return CompareResult(h[i], h[j])
}
