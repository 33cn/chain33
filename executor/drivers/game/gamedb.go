package game

//database opeartion for executor game
import (
	"bytes"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/types"
	"sort"
)

const (
	//剪刀
	Scissor = 0
	//石头
	Rock = 1
	//布
	Paper = 2

	//游戏结果
	//平局
	IsDraw       = 0
	IsCreatorWin = 1
	IsMatcherWin = 2
	//开奖超时
	IsTimeOut = 3
	//从有matcher参与游戏开始计算本局游戏开奖的有效时间，单位为天
	Active_Time = 1
)

var (
	//游戏状态
	GameStatus = []int32{types.GameActionCreate, types.GameActionMatch, types.GameActionCancel, types.GameActionClose}
)

//game 的状态变化：
// staus == 0  (创建，开始猜拳游戏）
// status == 1 (匹配，参与)
// status == 2 (取消)
// status == 3 (Close的情况)

//list 保存的方法:
//key=status:addr:gameId
func (action *Action) GetReceiptLog(game *types.Game) *types.ReceiptLog {
	log := &types.ReceiptLog{}
	if game.Status == types.GameActionCreate {
		log.Ty = types.TyLogCreateGame
	} else if game.Status == types.GameActionCancel {
		log.Ty = types.TyLogCancleGame
	} else if game.Status == types.GameActionMatch {
		log.Ty = types.TyLogMatchGame
	} else if game.Status == types.GameActionClose {
		log.Ty = types.TyLogCloseGame
	}
	r := &types.ReceiptGame{}
	r.GameId = game.GameId
	r.Status = game.Status
	//TODO 记录这个action由哪个地址触发的
	if action.fromaddr != game.GetCreateAddress() && game.GetCreateAddress() != "" {
		r.Addr = game.GetCreateAddress()
	} else {
		r.Addr = action.fromaddr
	}
	log.Log = types.Encode(r)
	return log
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

type Action struct {
	coinsAccount *account.DB
	db           dbm.KV
	txhash       []byte
	fromaddr     string
	blocktime    int64
	height       int64
	execaddr     string
}

func NewAction(g *Game, tx *types.Transaction) *Action {
	hash := tx.Hash()
	fromaddr := tx.From()
	return &Action{g.GetCoinsAccount(), g.GetStateDB(), hash, fromaddr, g.GetBlockTime(), g.GetHeight(), g.GetAddr()}
}
func (action *Action) saveGameToStateDB(game *types.Game) {
	value := types.Encode(game)
	action.db.Set(Key(game.GetGameId()), value)
}
func (action *Action) GameCreate(create *types.GameCreate) (*types.Receipt, error) {
	gameId := common.ToHex(action.txhash)
	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
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
	}
	action.saveGameToStateDB(game)
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
			match.GetGameId(), "err", "can't join the game, the game has started or finished!")
		return nil, types.ErrInputPara
	}
	acc := action.coinsAccount.LoadExecAccount(action.fromaddr, action.execaddr)
	if acc.Balance < game.GetValue()/2 {
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
	action.saveGameToStateDB(game)
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
			cancel.GetGameId(), "err", "not creator")
		return nil, types.ErrInputPara
	}
	if game.GetStatus() != types.GameActionCreate {
		glog.Error("GameCancel ", "addr", action.fromaddr, "execaddr", action.execaddr, "id",
			cancel.GetGameId(), "err", "can't cancel game, the game has started or finished!")
		return nil, types.ErrInputPara
	}
	receipt, err := action.coinsAccount.ExecActive(game.GetCreateAddress(), action.execaddr, game.GetValue())
	if err != nil {
		glog.Error("GameCancel ", "addr", action.fromaddr, "execaddr", action.execaddr, "id",
			cancel.GetGameId(), "amount", game.GetValue(), "err", err)
		return nil, err
	}
	game.Status = types.GameActionCancel
	action.saveGameToStateDB(game)
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
	//TODO:开奖涉及的一系列参数检查，后面需要补充
	if game.GetStatus() != types.GameActionMatch {
		glog.Error("the game status is not match.")
		return nil, types.ErrInputPara
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
	game.Result = close.GetResult()
	action.saveGameToStateDB(game)
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
	if action.blocktime > (game.GetMatchTime() + int64(DurTime)) {
		return true
	}
	return false
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

//TODO:这块可能需要做个分页，防止数据过大?这里需要做补强
func List(db dbm.Lister, stateDB dbm.KV, glist *types.QueryGameListByStatusAndAddr) (types.Message, error) {
	values, err := db.List(calcGamePrefix(glist.GetStatus(), glist.GetAddress()), nil, 0, 0)
	if err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return &types.ReplyGameList{}, nil
	}
	queryGameInfo := &types.QueryGameInfos{CheckGameIdDup(values)}
	//return Infos(stateDB, queryGameInfo)
	return BatchGetGameList(stateDB, queryGameInfo)
}

func readGame(db dbm.KV, id string) (*types.Game, error) {
	data, err := db.Get(Key(id))
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

//批量查询接口
func BatchGetGameList(db dbm.KV, infos *types.QueryGameInfos) (types.Message, error) {
	var games []*types.Game
	var keys [][]byte
	for _, gameId := range infos.GameIds {
		keys = append(keys, Key(gameId))
	}
	values, err := db.BatchGet(keys)
	if err != nil {
		return nil, err
	}
	for _, value := range values {
		var game types.Game
		err = types.Decode(value, &game)
		if err != nil {
			return nil, err
		}
		games = append(games, &game)
	}

	return &types.ReplyGameList{Games: games}, nil
}

func CheckGameIdDup(values [][]byte) []string {
	//根据别人测试总结，使用最节省时间的去重方式
	if len(values) < 1024 {
		// 切片长度小于1024的时候，循环来过滤
		return removeDupByLoop(values)
	} else {
		// 大于的时候，通过map来过滤
		return removeDupByMap(values)
	}
}
func removeDupByMap(values [][]byte) []string {
	var result []string
	//利用map 中的key去覆盖，过滤
	var tempMap map[string]byte
	for _, value := range values {
		//过滤空值
		if bytes.Equal(value, types.EmptyValue) {
			continue
		}
		lenth := len(tempMap)
		tempMap[string(value)] = 0
		if len(tempMap) != lenth {
			result = append(result, string(value))
		}
	}
	return result
}
func removeDupByLoop(values [][]byte) []string {
	var result []string
	for i := range values {
		//过滤空值
		if bytes.Equal(values[i], types.EmptyValue) {
			continue
		}
		flag := true
		for j := range result {
			//过滤空值
			if bytes.Equal(values[i], types.EmptyValue) {
				continue
			}
			if string(values[i]) == result[j] {
				flag = false // 存在重复元素，标识为false
				break
			}
		}
		if flag { // 标识为false，不添加进结果
			result = append(result, string(values[i]))
		}
	}
	return result
}

//TODO:复写sort方法，后面可以根据业务需求调整相关接口，进行升降排序
//根据createTime递增排序
func SortGameByCreateTimeAsc(games []*types.Game) {
	SortGame(games, func(p, q *types.Game) bool {
		return p.GetCreateTime() < q.GetCreateTime()
	})
}

//降序
func SortGameByCreateTimeDesc(games []*types.Game) {
	SortGame(games, func(p, q *types.Game) bool {
		return p.GetCreateTime() > q.GetCreateTime()
	})
}

//根据matchTime递增排序
func SortGameByMatchTimeAsc(games []*types.Game) {
	SortGame(games, func(p, q *types.Game) bool {
		return p.GetMatchTime() < q.GetMatchTime()
	})
}
func SortGameByMatchTimeDesc(games []*types.Game) {
	SortGame(games, func(p, q *types.Game) bool {
		return p.GetMatchTime() > q.GetMatchTime()
	})
}

type GameWrapper struct {
	Games []*types.Game
	by    func(p, q *types.Game) bool
}
type SortBy func(p, q *types.Game) bool

func (gw GameWrapper) Len() int { // 重写 Len() 方法
	return len(gw.Games)
}
func (gw GameWrapper) Swap(i, j int) { // 重写 Swap() 方法
	gw.Games[i], gw.Games[j] = gw.Games[j], gw.Games[i]
}
func (gw GameWrapper) Less(i, j int) bool { // 重写 Less() 方法
	return gw.by(gw.Games[i], gw.Games[j])
}

// 封装成 SortGame 方法
func SortGame(games []*types.Game, by SortBy) {
	sort.Sort(GameWrapper{games, by})
}
