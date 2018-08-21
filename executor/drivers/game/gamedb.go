package game

//database opeartion for executor game
import (
	"bytes"
	"strconv"

	"fmt"

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/types"
)

const (
	//剪刀
	Scissor = 1
	//石头
	Rock = 2
	//布
	Paper = 3

	//游戏结果
	//平局
	IsDraw       = 1
	IsCreatorWin = 2
	IsMatcherWin = 3
	//开奖超时
	IsTimeOut = 4
	//从有matcher参与游戏开始计算本局游戏开奖的有效时间，单位为天
	Active_Time = 1

	ListDESC    = int32(0)
	ListASC     = int32(1)
	DefultCount = int32(20)  //默认一次取多少条记录
	MaxCount    = int32(100) //最多取100条

	CloseCount  = "CloseCount" //用于统计整个合约目前总共成功结束了多少场游戏。
	CreateCount = "CreateCount"
	MatchCount  = "MatchCount"
	CancelCount = "CancelCount"
)

//game 的状态变化：
// staus ==  1 (创建，开始猜拳游戏）
// status == 2 (匹配，参与)
// status == 3 (取消)
// status == 4 (Close的情况)
/*
  在石头剪刀布游戏中，一局游戏的生命周期只有四种状态，其中创建者参与了整个的生命周期，因此当一局游戏 发生状态变更时，
 都需要及时建立相应的索引与之关联，同时还要删除老状态时的索引，以免形成脏数据。

 分页查询接口的实现：
  1.索引建立规则;
     根据状态索引建立： key= status:HeightIndex
     状态地址索引建立：key= status:addr:HeightIndex
     value=gameId
    HeightIndex=fmt.Sprintf("%018d", d.GetHeight()*types.MaxTxsPerBlock+int64(d.GetIndex()))
  2.利用状态数据库中Game中存有相应的Action时的txhash，从而去查询相应action时的 height和index，
    继而可以得到准确的key,从而删除localDB中的臃余索引。

*/

func (action *Action) GetReceiptLog(game *types.Game) *types.ReceiptLog {
	log := &types.ReceiptLog{}
	r := &types.ReceiptGame{}
	//TODO 记录这个action由哪个地址触发的
	r.Addr = action.fromaddr
	if game.Status == types.GameActionCreate {
		log.Ty = types.TyLogCreateGame
		r.PrevStatus = -1
	} else if game.Status == types.GameActionCancel {
		log.Ty = types.TyLogCancleGame
		r.PrevStatus = types.GameActionCreate
	} else if game.Status == types.GameActionMatch {
		log.Ty = types.TyLogMatchGame
		r.PrevStatus = types.GameActionCreate
	} else if game.Status == types.GameActionClose {
		log.Ty = types.TyLogCloseGame
		r.PrevStatus = types.GameActionMatch
		r.Addr = game.GetCreateAddress()
	}
	r.GameId = game.GameId
	r.Status = game.Status
	r.CreateAddr = game.GetCreateAddress()
	r.MatchAddr = game.GetMatchAddress()
	r.Index = game.GetIndex()
	r.PrevIndex = game.GetPrevIndex()
	log.Log = types.Encode(r)
	return log
}
func (action *Action) GetIndex(game *types.Game) string {
	return fmt.Sprintf("%018d", action.height*types.MaxTxsPerBlock+int64(action.index))
}
func (action *Action) GetKVSet(game *types.Game) (kvset []*types.KeyValue) {
	value := types.Encode(game)
	kvset = append(kvset, &types.KeyValue{Key(game.GameId), value})
	return kvset
}

//gameId to save key
func Key(id string) (key []byte) {
	key = append(key, []byte("mavl-"+types.ExecName(types.GameX)+"-")...)
	key = append(key, []byte(id)...)
	return key
}
func CalcCountKey(status int32, countType string) (key []byte) {
	key = append(key, []byte("mavl-"+types.ExecName(types.GameX)+"-")...)
	key = append(key, []byte(fmt.Sprintf("%d:%s", status, countType))...)
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
	localDB      dbm.Lister
	index        int
}

func NewAction(g *Game, tx *types.Transaction, index int) *Action {
	hash := tx.Hash()
	fromaddr := tx.From()
	return &Action{g.GetCoinsAccount(), g.GetStateDB(), hash, fromaddr, g.GetBlockTime(), g.GetHeight(), g.GetAddr(), g.GetLocalDB(), index}
}

func (action *Action) saveGameToStateDB(game *types.Game) {
	value := types.Encode(game)
	action.db.Set(Key(game.GetGameId()), value)
}
func (action *Action) saveCount(status int32, countType string, height int64) {
	action.db.Set(CalcCountKey(status, countType), []byte(strconv.FormatInt(height, 10)))
}
func (action *Action) updateCount(status int32, countType string) {
	count, _ := queryCountByStatusAndCountType(action.db, status, countType)
	action.saveCount(status, countType, count+1)
}
func (action *Action) CheckExecAccountBalance(fromAddr string, ToFrozen, ToActive int64) bool {
	acc := action.coinsAccount.LoadExecAccount(fromAddr, action.execaddr)
	if acc.GetBalance() >= ToFrozen && acc.GetFrozen() >= ToActive {
		return true
	}
	return false
}
func (action *Action) GameCreate(create *types.GameCreate) (*types.Receipt, error) {
	gameId := common.ToHex(action.txhash)
	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	if !action.CheckExecAccountBalance(action.fromaddr, create.GetValue(), 0) {
		glog.Error("GameCreate", "addr", action.fromaddr, "execaddr", action.execaddr, "id",
			gameId, "err", types.ErrNoBalance)
		return nil, types.ErrNoBalance
	}
	//冻结子账户资金
	receipt, err := action.coinsAccount.ExecFrozen(action.fromaddr, action.execaddr, create.GetValue())
	if err != nil {
		glog.Error("GameCreate.ExecFrozen", "addr", action.fromaddr, "execaddr", action.execaddr, "amount", create.GetValue(), "err", err.Error())
		return nil, err
	}
	game := &types.Game{
		GameId:        gameId,
		Value:         create.GetValue(),
		HashType:      create.GetHashType(),
		HashValue:     create.GetHashValue(),
		CreateTime:    action.blocktime,
		CreateAddress: action.fromaddr,
		Status:        types.GameActionCreate,
		CreateTxHash:  gameId,
	}
	game.Index = action.GetIndex(game)
	action.saveGameToStateDB(game)
	action.updateCount(types.GameActionCreate, CreateCount)
	receiptLog := action.GetReceiptLog(game)
	logs = append(logs, receiptLog)
	kv = append(kv, action.GetKVSet(game)...)
	logs = append(logs, receipt.Logs...)
	kv = append(kv, receipt.KV...)
	receipt = &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil
}

//match game
func (action *Action) GameMatch(match *types.GameMatch) (*types.Receipt, error) {
	game, err := action.readGame(match.GetGameId())
	if err != nil {
		glog.Error("GameMatch", "addr", action.fromaddr, "execaddr", action.execaddr, "get game failed",
			match.GetGameId(), "err", err)
		return nil, err
	}
	if game.GetStatus() != types.GameActionCreate {
		glog.Error("GameMatch", "addr", action.fromaddr, "execaddr", action.execaddr, "id",
			match.GetGameId(), "err", types.ErrGameMatchStatus)
		return nil, types.ErrGameMatchStatus
	}
	if game.GetCreateAddress() == action.fromaddr {
		glog.Error("GameMatch", "addr", action.fromaddr, "execaddr", action.execaddr, "id",
			match.GetGameId(), "err", types.ErrGameMatch)
		return nil, types.ErrGameMatch
	}
	if !action.CheckExecAccountBalance(action.fromaddr, game.GetValue()/2, 0) {
		glog.Error("GameMatch", "addr", action.fromaddr, "execaddr", action.execaddr, "id",
			match.GetGameId(), "err", types.ErrNoBalance)
		return nil, types.ErrNoBalance
	}
	//冻结 game value 中资金的一半
	receipt, err := action.coinsAccount.ExecFrozen(action.fromaddr, action.execaddr, game.GetValue()/2)
	if err != nil {
		glog.Error("GameMatch.ExecFrozen", "addr", action.fromaddr, "execaddr", action.execaddr, "amount", game.GetValue()/2, "err", err.Error())
		return nil, err
	}
	game.Status = types.GameActionMatch
	game.Value = game.GetValue()/2 + game.GetValue()
	game.MatchAddress = action.fromaddr
	game.MatchTime = action.blocktime
	game.Guess = match.GetGuess()
	game.MatchTxHash = common.ToHex(action.txhash)
	game.PrevIndex = game.GetIndex()
	game.Index = action.GetIndex(game)
	action.saveGameToStateDB(game)
	action.updateCount(types.GameActionMatch, MatchCount)
	var logs []*types.ReceiptLog
	var kvs []*types.KeyValue
	receiptLog := action.GetReceiptLog(game)
	logs = append(logs, receiptLog)
	kvs = append(kvs, action.GetKVSet(game)...)
	logs = append(logs, receipt.Logs...)
	kvs = append(kvs, receipt.KV...)
	receipts := &types.Receipt{types.ExecOk, kvs, logs}
	return receipts, nil
}
func (action *Action) GameCancel(cancel *types.GameCancel) (*types.Receipt, error) {
	game, err := action.readGame(cancel.GetGameId())
	if err != nil {
		glog.Error("GameCancel ", "addr", action.fromaddr, "execaddr", action.execaddr, "get game failed",
			cancel.GetGameId(), "err", err)
		return nil, err
	}
	if game.GetCreateAddress() != action.fromaddr {
		glog.Error("GameCancel ", "addr", action.fromaddr, "execaddr", action.execaddr, "id",
			cancel.GetGameId(), "err", types.ErrGameCancleAddr)
		return nil, types.ErrGameCancleAddr
	}
	if game.GetStatus() != types.GameActionCreate {
		glog.Error("GameCancel ", "addr", action.fromaddr, "execaddr", action.execaddr, "id",
			cancel.GetGameId(), "err", types.ErrGameCancleStatus)
		return nil, types.ErrGameCancleStatus
	}
	if !action.CheckExecAccountBalance(action.fromaddr, 0, game.GetValue()) {
		glog.Error("GameCancel", "addr", action.fromaddr, "execaddr", action.execaddr, "id",
			game.GetGameId(), "err", types.ErrNoBalance)
		return nil, types.ErrNoBalance
	}
	receipt, err := action.coinsAccount.ExecActive(game.GetCreateAddress(), action.execaddr, game.GetValue())
	if err != nil {
		glog.Error("GameCancel ", "addr", action.fromaddr, "execaddr", action.execaddr, "id",
			cancel.GetGameId(), "amount", game.GetValue(), "err", err)
		return nil, err
	}
	game.Closetime = action.blocktime
	game.Status = types.GameActionCancel
	game.CancelTxHash = common.ToHex(action.txhash)
	game.PrevIndex = game.GetIndex()
	game.Index = action.GetIndex(game)
	action.saveGameToStateDB(game)
	action.updateCount(types.GameActionCancel, CancelCount)
	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	logs = append(logs, receipt.Logs...)
	receiptLog := action.GetReceiptLog(game)
	logs = append(logs, receiptLog)
	kvs := action.GetKVSet(game)
	kv = append(kv, receipt.KV...)
	kv = append(kv, kvs...)

	return &types.Receipt{types.ExecOk, kv, logs}, nil
}

func (action *Action) GameClose(close *types.GameClose) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	game, err := action.readGame(close.GetGameId())
	if err != nil {
		glog.Error("GameClose ", "addr", action.fromaddr, "execaddr", action.execaddr, "get game failed",
			close.GetGameId(), "err", err)
		return nil, err
	}
	//开奖时间控制
	if action.fromaddr != game.GetCreateAddress() && !action.checkGameIsTimeOut(game) {
		//如果不是游戏创建者开奖，则检查是否超时
		glog.Error(types.ErrGameCloseAddr.Error())
		return nil, types.ErrGameCloseAddr
	} else if action.fromaddr == game.GetCreateAddress() && action.checkGameIsTimeOut(game) {
		//当超过游戏开奖时间，此时，禁止游戏创建者去开奖，只能由其他人开奖
		glog.Error(types.ErrGameTimeOut.Error())
		return nil, types.ErrGameTimeOut
	}
	if game.GetStatus() != types.GameActionMatch {
		glog.Error(types.ErrGameCloseStatus.Error())
		return nil, types.ErrGameCloseStatus
	}
	//各自冻结余额检查
	if !action.CheckExecAccountBalance(game.GetCreateAddress(), 0, 2*game.GetValue()/3) {
		glog.Error("GameClose", "addr", game.GetCreateAddress(), "execaddr", action.execaddr, "id",
			game.GetGameId(), "err", types.ErrNoBalance)
		return nil, types.ErrNoBalance
	}
	if !action.CheckExecAccountBalance(game.GetMatchAddress(), 0, game.GetValue()/3) {
		glog.Error("GameClose", "addr", game.GetMatchAddress(), "execaddr", action.execaddr, "id",
			game.GetGameId(), "err", types.ErrNoBalance)
		return nil, types.ErrNoBalance
	}
	result := action.checkGameResult(game, close)
	if result == IsCreatorWin {
		//如果是庄家赢了，则解冻所有钱,并将对赌者相应冻结的钱转移到庄家的合约账户中
		//TODO:账户amount 使用int64,而不是float64，可能存在精度问题
		receipt, err := action.coinsAccount.ExecActive(game.GetCreateAddress(), action.execaddr, 2*game.GetValue()/3)
		if err != nil {
			glog.Error("GameClose.execActive", "addr", game.GetCreateAddress(), "execaddr", action.execaddr, "amount", 2*game.GetValue()/3,
				"err", err)
			return nil, err
		}
		logs = append(logs, receipt.Logs...)
		kv = append(kv, receipt.KV...)
		receipt, err = action.coinsAccount.ExecTransferFrozen(game.GetMatchAddress(), game.GetCreateAddress(), action.execaddr, game.GetValue()/3)
		if err != nil {
			action.coinsAccount.ExecFrozen(game.GetCreateAddress(), action.execaddr, 2*game.GetValue()/3) // rollback
			glog.Error("GameClose.ExecTransferFrozen", "addr", game.GetCreateAddress(), "execaddr", action.execaddr, "amount", 2*game.GetValue()/3,
				"err", err)
			return nil, err
		}
		logs = append(logs, receipt.Logs...)
		kv = append(kv, receipt.KV...)
	} else if result == IsMatcherWin {
		//如果是庄家输了，则反向操作
		receipt, err := action.coinsAccount.ExecActive(game.GetCreateAddress(), action.execaddr, game.GetValue()/3)
		if err != nil {
			glog.Error("GameClose.ExecActive", "addr", game.GetCreateAddress(), "execaddr", action.execaddr, "amount", game.GetValue()/3,
				"err", err)
			return nil, err
		}
		logs = append(logs, receipt.Logs...)
		kv = append(kv, receipt.KV...)
		receipt, err = action.coinsAccount.ExecActive(game.GetMatchAddress(), action.execaddr, game.GetValue()/3)
		if err != nil {
			action.coinsAccount.ExecFrozen(game.GetCreateAddress(), action.execaddr, game.GetValue()/3) // rollback
			glog.Error("GameClose.ExecActive", "addr", game.GetCreateAddress(), "execaddr", action.execaddr, "amount", game.GetValue()/3,
				"err", err)
			return nil, err
		}
		logs = append(logs, receipt.Logs...)
		kv = append(kv, receipt.KV...)
		receipt, err = action.coinsAccount.ExecTransferFrozen(game.GetCreateAddress(), game.GetMatchAddress(), action.execaddr, game.GetValue()/3)
		if err != nil {
			action.coinsAccount.ExecFrozen(game.GetCreateAddress(), action.execaddr, game.GetValue()/3) // rollback
			action.coinsAccount.ExecFrozen(game.GetMatchAddress(), action.execaddr, game.GetValue()/3)  // rollback
			glog.Error("GameClose.ExecTransferFrozen", "addr", game.GetCreateAddress(), "execaddr", action.execaddr, "amount", game.GetValue()/3,
				"err", err)
			return nil, err
		}
		logs = append(logs, receipt.Logs...)
		kv = append(kv, receipt.KV...)

	} else if result == IsDraw {
		//平局是解冻各自的押注即可
		receipt, err := action.coinsAccount.ExecActive(game.GetCreateAddress(), action.execaddr, 2*game.GetValue()/3)
		if err != nil {
			glog.Error("GameClose.ExecActive", "addr", game.GetCreateAddress(), "execaddr", action.execaddr, "amount", 2*game.GetValue()/3,
				"err", err)
			return nil, err
		}
		logs = append(logs, receipt.Logs...)
		kv = append(kv, receipt.KV...)
		receipt, err = action.coinsAccount.ExecActive(game.GetMatchAddress(), action.execaddr, game.GetValue()/3)
		if err != nil {
			action.coinsAccount.ExecFrozen(game.GetCreateAddress(), action.execaddr, 2*game.GetValue()/3) // rollback
			glog.Error("GameClose.ExecActive", "addr", game.GetMatchAddress(), "execaddr", action.execaddr, "amount", game.GetValue()/3,
				"err", err)
			return nil, err
		}
		logs = append(logs, receipt.Logs...)
		kv = append(kv, receipt.KV...)
	} else if result == IsTimeOut {
		//开奖超时，庄家输掉所有筹码
		receipt, err := action.coinsAccount.ExecActive(game.GetMatchAddress(), action.execaddr, game.GetValue()/3)
		if err != nil {
			glog.Error("GameClose.ExecActive", "addr", game.GetCreateAddress(), "execaddr", action.execaddr, "amount", 2*game.GetValue()/3,
				"err", err)
			return nil, err
		}
		logs = append(logs, receipt.Logs...)
		kv = append(kv, receipt.KV...)
		receipt, err = action.coinsAccount.ExecTransferFrozen(game.GetCreateAddress(), game.GetMatchAddress(), action.execaddr, 2*game.GetValue()/3)
		if err != nil {
			action.coinsAccount.ExecFrozen(game.GetMatchAddress(), action.execaddr, game.GetValue()/3) // rollback
			glog.Error("GameClose.ExecTransferFrozen", "addr", game.GetMatchAddress(), "execaddr", action.execaddr, "amount", game.GetValue()/3,
				"err", err)
			return nil, err
		}
		logs = append(logs, receipt.Logs...)
		kv = append(kv, receipt.KV...)
	}
	game.Closetime = action.blocktime
	game.Status = types.GameActionClose
	game.Secret = close.GetSecret()
	game.Result = int32(result)
	game.CloseTxHash = common.ToHex(action.txhash)
	game.PrevIndex = game.GetIndex()
	game.Index = action.GetIndex(game)
	action.saveGameToStateDB(game)
	action.updateCount(types.GameActionClose, CloseCount)
	receiptLog := action.GetReceiptLog(game)
	logs = append(logs, receiptLog)
	kvs := action.GetKVSet(game)
	kv = append(kv, kvs...)
	return &types.Receipt{types.ExecOk, kv, logs}, nil
}

// 检查开奖是否超时，若超过一天，则不让庄家开奖，但其他人可以开奖，
// 若没有一天，则其他人没有开奖权限，只有庄家有开奖权限
func (action *Action) checkGameIsTimeOut(game *types.Game) bool {
	DurTime := 60 * 60 * 24 * Active_Time
	return action.blocktime > (game.GetMatchTime() + int64(DurTime))
}

//根据传入密钥，揭晓游戏结果
func (action *Action) checkGameResult(game *types.Game, close *types.GameClose) int {
	//如果超时，直接走超时开奖逻辑
	if action.checkGameIsTimeOut(game) {
		return IsTimeOut
	}
	if bytes.Equal(common.Sha256([]byte(close.GetSecret()+string(Rock))), game.GetHashValue()) {
		//此刻庄家出的是石头
		if game.GetGuess() == Rock {
			return IsDraw
		} else if game.GetGuess() == Scissor {
			return IsCreatorWin
		} else if game.GetGuess() == Paper {
			return IsMatcherWin
		} else {
			//其他情况说明matcher 使坏，填了其他值，当做作弊处理
			return IsCreatorWin
		}
	} else if bytes.Equal(common.Sha256([]byte(close.GetSecret()+string(Scissor))), game.GetHashValue()) {
		//此刻庄家出的剪刀
		if game.GetGuess() == Rock {
			return IsMatcherWin
		} else if game.GetGuess() == Scissor {
			return IsDraw
		} else if game.GetGuess() == Paper {
			return IsCreatorWin
		} else {
			return IsCreatorWin
		}
	} else if bytes.Equal(common.Sha256([]byte(close.GetSecret()+string(Paper))), game.GetHashValue()) {
		//此刻庄家出的是布
		if game.GetGuess() == Rock {
			return IsCreatorWin
		} else if game.GetGuess() == Scissor {
			return IsMatcherWin
		} else if game.GetGuess() == Paper {
			return IsDraw
		} else {
			return IsCreatorWin
		}
	}
	//其他情况默认是matcher win
	return IsMatcherWin
}

func (action *Action) readGame(id string) (*types.Game, error) {
	data, err := action.db.Get(Key(id))
	if err != nil {
		return nil, err
	}
	var game types.Game
	//decode
	err = types.Decode(data, &game)
	if err != nil {
		return nil, err
	}
	return &game, nil
}

func List(db dbm.Lister, stateDB dbm.KV, param *types.QueryGameListByStatusAndAddr) (types.Message, error) {
	return QueryGameListByPage(db, stateDB, param)
}

//分页查询
func QueryGameListByPage(db dbm.Lister, stateDB dbm.KV, param *types.QueryGameListByStatusAndAddr) (types.Message, error) {

	switch param.GetStatus() {
	case types.GameActionCreate, types.GameActionMatch, types.GameActionClose, types.GameActionCancel:
		queryGameListByStatusAndAddr(db, stateDB, param)
	}

	return nil, fmt.Errorf("the status only fill in 1,2,3,4!")
}

func queryGameListByStatusAndAddr(db dbm.Lister, stateDB dbm.KV, param *types.QueryGameListByStatusAndAddr) (types.Message, error) {
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
	if param.GetAddress() == "" {
		prefix = calcGameStatusIndexPrefix(param.Status)
		key = calcGameStatusIndexKey(param.Status, param.GetIndex())
	} else {
		prefix = calcGameAddrIndexPrefix(param.Status, param.GetAddress())
		key = calcGameAddrIndexKey(param.Status, param.GetAddress(), param.GetIndex())
	}

	if param.GetIndex() == "" { //第一次查询
		values, err := db.List(prefix, nil, count, direction)
		if err != nil {
			return nil, err
		}
		var gameIds []string
		for _, value := range values {
			var record types.GameRecord
			err := types.Decode(value, &record)
			if err != nil {
				continue
			}
			gameIds = append(gameIds, record.GetGameId())
		}
		games := GetGameList(stateDB, gameIds)
		index := games[len(games)-1].GetIndex()
		if len(games) < int(count) {
			return &types.ReplyGameListPage{games, "", ""}, nil
		}
		return &types.ReplyGameListPage{games, "", index}, nil

	} else {
		values, err := db.List(prefix, key, count, direction)
		if err != nil {
			return nil, err
		}
		var gameIds []string
		for _, value := range values {
			var record types.GameRecord
			err := types.Decode(value, &record)
			if err != nil {
				continue
			}
			gameIds = append(gameIds, record.GetGameId())
		}
		games := GetGameList(stateDB, gameIds)
		index := games[len(games)-1].GetIndex()
		if len(games) == 0 {
			return &types.ReplyGameListPage{nil, param.GetIndex(), ""}, nil
		}
		if len(games) < int(count) {
			return &types.ReplyGameListPage{games, param.GetIndex(), ""}, nil

		}
		return &types.ReplyGameListPage{games, param.GetIndex(), index}, nil
	}
}

//count数查询
func QueryGameListCount(db dbm.Lister, stateDB dbm.KV, param *types.QueryGameListCount) (types.Message, error) {
	if param.Status < 1 || param.Status > 4 {
		return nil, fmt.Errorf("the status only fill in 1,2,3,4!")
	}
	if param.GetAddress() == "" {
		//直接从状态数据库中取
		createCount := QueryCountByStatus(stateDB, types.GameActionCreate)
		closeCount := QueryCountByStatus(stateDB, types.GameActionClose)
		cancleCount := QueryCountByStatus(stateDB, types.GameActionCancel)
		matchCount := QueryCountByStatus(stateDB, types.GameActionMatch)
		switch param.GetStatus() {
		case types.GameActionCreate:
			return &types.ReplyGameListCount{createCount - matchCount - cancleCount}, nil
		case types.GameActionMatch:
			return &types.ReplyGameListCount{matchCount - closeCount}, nil
		case types.GameActionCancel:
			return &types.ReplyGameListCount{cancleCount}, nil
		case types.GameActionClose:
			return &types.ReplyGameListCount{closeCount}, nil
		}
	}
	return &types.ReplyGameListCount{db.PrefixCount(calcGameAddrIndexPrefix(param.Status, param.Address))}, nil
}
func QueryCountByStatus(stateDB dbm.KV, status int32) int64 {
	switch status {
	case types.GameActionCreate:
		count, _ := queryCountByStatusAndCountType(stateDB, status, CreateCount)
		return count
	case types.GameActionMatch:
		count, _ := queryCountByStatusAndCountType(stateDB, status, MatchCount)
		return count
	case types.GameActionCancel:
		count, _ := queryCountByStatusAndCountType(stateDB, status, CancelCount)
		return count
	case types.GameActionClose:
		count, _ := queryCountByStatusAndCountType(stateDB, status, CloseCount)
		return count
	}
	return 0
}
func queryCountByStatusAndCountType(stateDB dbm.KV, status int32, countType string) (int64, error) {
	data, err := stateDB.Get(CalcCountKey(status, countType))
	if err != nil {
		glog.Error("query data have err:", err.Error())
		return 0, err
	}
	count, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		glog.Error("Type conversion error:", err.Error())
		return 0, err
	}
	return count, nil
}

func readGame(db dbm.KV, id string) (*types.Game, error) {
	data, err := db.Get(Key(id))
	if err != nil {
		glog.Error("query data have err:", err.Error())
		return nil, err
	}
	var game types.Game
	//decode
	err = types.Decode(data, &game)
	if err != nil {
		glog.Error("decode data have err:", err.Error())
		return nil, err
	}
	return &game, nil
}

func Infos(db dbm.KV, infos *types.QueryGameInfos) (types.Message, error) {
	var games []*types.Game
	for i := 0; i < len(infos.GameIds); i++ {
		id := infos.GameIds[i]
		game, err := readGame(db, id)
		if err != nil {
			return nil, err
		}
		games = append(games, game)
	}
	return &types.ReplyGameList{Games: games}, nil
}

//安全批量查询方式,防止因为脏数据导致查询接口奔溃
func GetGameList(db dbm.KV, values []string) []*types.Game {
	var games []*types.Game
	for _, value := range values {
		game, err := readGame(db, value)
		if err != nil {
			continue
		}
		games = append(games, game)
	}
	return games
}
