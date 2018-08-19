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

	ListDESC    = int32(0)
	ListASC     = int32(1)
	ListSeek    = int32(2)
	DefultCount = int32(20)         //默认一次取多少条记录
	MaxCount    = int32(100)        //最多取100条
	InitHeight  = int64(1000000000) //每个地址的初始索引高度,占位，在ASCII中数字的排序在字符之前

	LastHeight  = "LastHeight" //用于表示每个地址对应的高度是多少，用于记录个人成功完成多少局游戏
	CloseCount  = "CloseCount" //用于统计整个合约目前总共成功结束了多少场游戏。
	CreateCount = "CreateCount"
	MatchCount  = "MatchCount"
	CancelCount = "CancelCount"
)

//game 的状态变化：
// staus == 0  (创建，开始猜拳游戏）
// status == 1 (匹配，参与)
// status == 2 (取消)
// status == 3 (Close的情况)
/*
  在石头剪刀布游戏中，一局游戏的生命周期只有四种状态，其中创建者参与了整个的生命周期，因此当一局游戏 发生状态变更时，
 都需要及时建立相应的索引与之关联，同时还要删除老状态时的索引，以免形成脏数据。
 同样匹配者在游戏中与之相关的状态只有1和3，即匹配和开奖，只要基于这个出发点建立相应的索引即可。

 分页查询接口的实现：
  1.状态数据库中每个用户各自建立两条记录用户分别对应最终的close,cancel两种状态的高度
  2.状态数据库中建立四个Key,分别用于保存历史相应状态的count总数
  3.索引建立规则;
     create,match 状态索引建立： key= status:addr:blocktime:gameId
     cancel,close 状态索引建立：key= status:addr:Height  //height 为自增长
     value=gameId
    创建者：每次Action之后，都将对应的status, height 传入到createHeight中，更新信的状态索引，删除旧索引
     问题关键：1.对于create，match 状态的 game会存在旧索引，如何删除呢？
		这里的解决方法是，利用game中存储了ationTime,也就是打包时间戳,来解决排序，和删除旧的索引
       create---------->match-------->close
        \
      cancel
        当游戏变为cancel,close 这样最终状态时，就可以利用key= status:addr:Height去建立索引
         因为这种状态属于只进不出，不用删除，所以可能构造指定的索引去查询。

              2.有删就有回滚，localDB中的数据如何实现回滚呢？
                还是通过log,能根据log去建索引，反之也可以根据它来删除索引。
    这样做的优点：
                1.对cancel,close状态的游戏来讲，index高度是从1，自增长的，且中间不会出现index被删除状态
                   故所有close 状态的游戏，都可以通过分页接口进行准确查找，无论数据量有多大，也不影响
                   查找精度，因为index始终是准确的。

		不足之处：
                1.对于create,match 状态的游戏来讲，index 是不准确的，因为create,match 状态不是最终状态
                  随时会产生变化，且变化先后不可测，一次create和matcher 状态的数据这里用了比较粗笨的方法
                  去处理，当上下翻页超过1000条记录时，考虑数据库性能问题，就不让继续往下查找了，不过还好
                  create,match 只是临时状态，会不断地被处理
    匹配者：
*/

func (action *Action) GetReceiptLog(game *types.Game) *types.ReceiptLog {
	log := &types.ReceiptLog{}
	r := &types.ReceiptGame{}
	//TODO 记录这个action由哪个地址触发的
	r.Addr = action.fromaddr
	if game.Status == types.GameActionCreate {
		log.Ty = types.TyLogCreateGame
		r.PrevStatus = -1
		r.PrevActionTime = -1
	} else if game.Status == types.GameActionCancel {
		log.Ty = types.TyLogCancleGame
		r.PrevStatus = types.GameActionCreate
		r.PrevActionTime = game.GetCreateTime()
		//只有close,cancel 操作时，才涉及最终状态高度变更
		r.CreateHeight = action.GetLastHeight(game.Status, game.CreateAddress)
		r.MatchHeight = action.GetLastHeight(game.Status, game.MatchAddress)
	} else if game.Status == types.GameActionMatch {
		log.Ty = types.TyLogMatchGame
		r.PrevStatus = types.GameActionCreate
		r.PrevActionTime = game.GetCreateTime()
	} else if game.Status == types.GameActionClose {
		log.Ty = types.TyLogCloseGame
		r.PrevStatus = types.GameActionMatch
		r.PrevActionTime = game.GetMatchTime()
		//只有close,cancel 操作时，才涉及最终状态高度变更
		r.CreateHeight = action.GetLastHeight(game.Status, game.CreateAddress)
		r.MatchHeight = action.GetLastHeight(game.Status, game.MatchAddress)
		r.Addr = game.GetCreateAddress()
	}
	r.GameId = game.GameId
	r.Status = game.Status
	r.CreateAddr = game.GetCreateAddress()
	r.MatchAddr = game.GetMatchAddress()
	r.ActionTime = action.blocktime
	log.Log = types.Encode(r)
	return log
}
func (action *Action) GetLastHeight(status int32, addr string) int64 {
	height, err := queryHeightByStatusAndHeightType(action.db, status, addr, LastHeight)
	if err != nil {
		return 0
	}
	return height
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
func CalcKeyByHeight(status int32, addr string, heightType string) (key []byte) {
	key = append(key, []byte("mavl-"+types.ExecName(types.GameX)+"-")...)
	key = append(key, []byte(fmt.Sprintf("%d:%s:%s", status, addr, heightType))...)
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
func (action *Action) saveHeight(status int32, addr, heightType string, height int64) {
	action.db.Set(CalcKeyByHeight(status, addr, heightType), []byte(strconv.FormatInt(height, 10)))
}
func (action *Action) updateHeight(addr string, status int32) {
	height, _ := queryHeightByStatusAndHeightType(action.db, status, addr, LastHeight)
	action.saveHeight(status, addr, LastHeight, height+1)
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
	}
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
	action.saveGameToStateDB(game)
	action.updateHeight(action.fromaddr, types.GameActionCancel)
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
	action.saveGameToStateDB(game)
	//更新状态数据库中处于close状态的数据高度
	action.updateHeight(game.GetCreateAddress(), types.GameActionClose)
	action.updateHeight(game.MatchAddress, types.GameActionClose)
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

//TODO:前序，后序段首查询，再修改一下可以根据某个时间端来查询信息
func List(db dbm.Lister, stateDB dbm.KV, param *types.QueryGameListByStatusAndAddr) (types.Message, error) {
	return QueryGameListByPage(db, stateDB, param)
}

//分页查询,pageNum从0开始，为了兼容以前接口接受默认值为0，按第一页处理
func QueryGameListByPage(db dbm.Lister, stateDB dbm.KV, param *types.QueryGameListByStatusAndAddr) (types.Message, error) {
	//无论前台传来什么值，要确保direction为0或者1
	direction := ListDESC
	if param.GetDirection() == ListASC {
		direction = ListASC
	}
	count := DefultCount
	if 0 < param.GetCount() && param.GetCount() <= MaxCount {
		count = param.GetCount()
	}
	if param.GetPageNum() < 0 {
		return nil, fmt.Errorf("Number of bad pages!")
	}
	pageNum := param.GetPageNum() + 1
	switch param.GetStatus() {
	case types.GameActionCancel, types.GameActionClose:
		//查询cancel,close 状态必须地址，不携带支持禁止调用接口
		if param.GetAddress() == "" {
			return nil, fmt.Errorf("The address cannot be empty!")
		}
		Height, err := queryHeightByStatusAndHeightType(stateDB, param.Status, param.Address, LastHeight)
		if err != nil {
			return nil, err
		}
		var values [][]byte
		var height int64
		if direction == ListDESC {
			height = Height - int64(count*(pageNum-1))
			if height <= 0 {
				return nil, fmt.Errorf("The index of crossing the line.")
			}
			for i := height; i > 0; i-- {
				value, err := db.List(calcGameIdPrefix(param.GetStatus(), param.GetAddress()), calcGameIdKey(param.Status, param.Address, height), count, direction)
				if err != nil {
					continue
				}
				values = append(values, value...)
			}
		}
		if direction == ListASC {
			if int64(count*(pageNum-1)) > Height {
				return nil, fmt.Errorf("The index of crossing the line.")
			}
			height = int64(count * (pageNum - 1))
			for i := height; i <= Height; i++ {
				value, err := db.List(calcGameIdPrefix(param.GetStatus(), param.GetAddress()), calcGameIdKey(param.Status, param.Address, height), count, direction)
				if err != nil {
					continue
				}
				values = append(values, value...)
			}
		}
		//TODO:这样一次性把整页查出来数据返回是否合理？
		if len(values) == 0 {
			return &types.ReplyGameList{}, nil
		}
		return GetGameList(stateDB, values)
	case types.GameActionCreate, types.GameActionMatch:
		//此刻没有准确的索引，需要根据单页返回的count值来处理
		if pageNum*count <= MaxCount*5 { //这里正序和反序取逻辑一样
			values, err := db.List(calcGamePrefix(param.GetStatus(), param.GetAddress()), nil, pageNum*count, direction)
			if err != nil {
				return nil, err
			}
			if len(values) <= int(pageNum*count) && len(values) > int((pageNum-1)*count) {
				return GetGameList(stateDB, CheckGameIdDup(values[(pageNum-1)*count-1:], param))
			}
			return nil, fmt.Errorf("The index of crossing the line.")
		}

		if pageNum*count > MaxCount*5 { //超过500条，需要查询总数
			if int64(pageNum*count) > 1000 { //限制最多让其向上向下探查1000条记录
				return nil, fmt.Errorf("Your search is so deep, please choose the game before it.")
			}
			total := db.PrefixCount(calcGamePrefix(param.GetStatus(), param.GetAddress()))
			//TODO create,match 状态都可能随时发生变化，状态不断地被消退，对于用户而言，处于
			// 这两个状态的数据应该不会太多,因此这样处理一般没有大的问题。
			if total < int64(pageNum*count) {
				return nil, fmt.Errorf("The index of crossing the line.")
			}
			values, err := db.List(calcGamePrefix(param.GetStatus(), param.GetAddress()), nil, pageNum*count, direction)
			if err != nil {
				return nil, err
			}
			if len(values) <= int(pageNum*count) && len(values) > int((pageNum-1)*count) {
				return GetGameList(stateDB, CheckGameIdDup(values[(pageNum-1)*count-1:], param))
			}
			return nil, fmt.Errorf("The index of crossing the line.")

		}
	}
	return nil, fmt.Errorf("the status only fill in 0,1,2,3!")
}

//count数查询
func QueryGameListCount(db dbm.Lister, stateDB dbm.KV, param *types.QueryGameListCount) (types.Message, error) {
	if param.Status < 0 || param.Status > 3 {
		return nil, fmt.Errorf("the status only fill in 0,1,2,3!")
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
	if param.GetStatus() == types.GameActionCancel || param.GetStatus() == types.GameActionClose {
		//直接查询状态数据库中的height,即为总数
		height, err := queryHeightByStatusAndHeightType(stateDB, param.Status, param.Address, LastHeight)
		if err != nil {
			return nil, err
		}
		return &types.ReplyGameListCount{height}, nil
	}
	return &types.ReplyGameListCount{db.PrefixCount(calcGamePrefix(param.GetStatus(), param.GetAddress()))}, nil
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
func queryHeightByStatusAndHeightType(stateDB dbm.KV, status int32, addr, heightType string) (int64, error) {
	data, err := stateDB.Get(CalcKeyByHeight(status, addr, heightType))
	if err != nil {
		glog.Error("query data have err:", err.Error())
		return 0, err
	}
	height, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		glog.Error("Type conversion error:", err.Error())
		return 0, err
	}
	return height, nil
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
func GetGameList(db dbm.KV, values [][]byte) (types.Message, error) {
	var games []*types.Game
	for _, value := range values {
		game, err := readGame(db, string(value))
		if err != nil {
			continue
		}
		games = append(games, game)
	}
	return &types.ReplyGameList{Games: games}, nil
}

//当state为1,2,3 ,addr为空时，存在重复key 因此要去重
func CheckGameIdDup(values [][]byte, query *types.QueryGameListByStatusAndAddr) [][]byte {
	//如果地址不为空，则不需要过滤
	if query.GetAddress() != "" {
		return values
	}
	//根据别人测试总结，使用最节省时间的去重方式
	if len(values) < 1024 {
		// 切片长度小于1024的时候，循环来过滤
		return removeDupByLoop(values)
	} else {
		// 大于的时候，通过map来过滤
		return removeDupByMap(values)
	}
}
func removeDupByMap(values [][]byte) [][]byte {
	var result [][]byte
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
			result = append(result, value)
		}
	}
	return result
}
func removeDupByLoop(values [][]byte) [][]byte {
	var result [][]byte
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
			if bytes.Equal(values[i], result[j]) {
				flag = false // 存在重复元素，标识为false
				break
			}
		}
		if flag { // 标识为false，不添加进结果
			result = append(result, values[i])
		}
	}
	return result
}
