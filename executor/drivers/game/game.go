package game

import (
	"fmt"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/types"
)

var glog = log.New("module", "execs.game")

const (
	MaxGameAmount = 100 * types.Coin
	//查询方法名
	FuncName_QueryGameListByIds = "QueryGameListByIds"
	//FuncName_QueryGameListByPage          = "QueryGameListByPage"
	FuncName_QueryGameListCount           = "QueryGameListCount"
	FuncName_QueryGameListByStatusAndAddr = "QueryGameListByStatusAndAddr"
	FuncName_QueryGameById                = "QueryGameById"
)

func Init() {
	drivers.Register(newGame().GetName(), newGame, 0)
}

type Game struct {
	drivers.DriverBase
}

func newGame() drivers.Driver {
	t := &Game{}
	t.SetChild(t)
	return t
}

func (g *Game) GetName() string {
	return types.ExecName(types.GameX)
}

func (g *Game) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	var action types.GameAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return nil, err
	}
	glog.Debug("exec Game tx=", "tx=", action)
	actiondb := NewAction(g, tx, index)
	if action.Ty == types.GameActionCreate && action.GetCreate() != nil {
		create := action.GetCreate()
		if create.GetValue() > MaxGameAmount {
			glog.Error("Create the game, the deposit is too big  ", "value", create.GetValue())
			return nil, types.ErrGameCreateAmount
		}
		return actiondb.GameCreate(create)
	} else if action.Ty == types.GameActionCancel && action.GetCancel() != nil {
		return actiondb.GameCancel(action.GetCancel())
	} else if action.Ty == types.GameActionClose && action.GetClose() != nil {
		return actiondb.GameClose(action.GetClose())
	} else if action.Ty == types.GameActionMatch && action.GetMatch() != nil {
		return actiondb.GameMatch(action.GetMatch())
	}
	//return error
	return nil, types.ErrActionNotSupport
}

func (g *Game) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := g.DriverBase.ExecLocal(tx, receipt, index)
	if err != nil {
		return nil, err
	}
	if receipt.GetTy() != types.ExecOk {
		return set, nil
	}
	for i := 0; i < len(receipt.Logs); i++ {
		item := receipt.Logs[i]
		//这四个是Game 的log
		if item.Ty == types.TyLogCreateGame || item.Ty == types.TyLogMatchGame || item.Ty == types.TyLogCloseGame || item.Ty == types.TyLogCancleGame {
			var Gamelog types.ReceiptGame
			err := types.Decode(item.Log, &Gamelog)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			kv := g.updateIndex(&Gamelog)
			set.KV = append(set.KV, kv...)
		}
	}
	return set, nil
}

func (g *Game) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := g.DriverBase.ExecDelLocal(tx, receipt, index)
	if err != nil {
		return nil, err
	}
	if receipt.GetTy() != types.ExecOk {
		return set, nil
	}
	for i := 0; i < len(receipt.Logs); i++ {
		item := receipt.Logs[i]
		if item.Ty == types.TyLogCreateGame || item.Ty == types.TyLogMatchGame || item.Ty == types.TyLogCloseGame || item.Ty == types.TyLogCancleGame {
			var Gamelog types.ReceiptGame
			err := types.Decode(item.Log, &Gamelog)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			//状态数据库由于默克尔树特性，之前生成的索引无效，故不需要回滚，只回滚localDB
			kv := g.rollbackIndex(&Gamelog)
			set.KV = append(set.KV, kv...)
		}
	}
	return set, nil
}

//更新索引
func (g *Game) updateIndex(log *types.ReceiptGame) (kvs []*types.KeyValue) {
	//先保存本次Action产生的索引
	kvs = append(kvs, addGameAddrIndex(log.Status, log.GameId, log.Addr, log.Index))
	kvs = append(kvs, addGameStatusIndex(log.Status, log.GameId, log.Index))
	if log.Status == types.GameActionMatch {
		kvs = append(kvs, addGameAddrIndex(log.Status, log.GameId, log.CreateAddr, log.Index))
		prevIndex, err := g.queryIndex(log.GetGameId())
		if err != nil {
			panic(err) //数据损毁
		}
		kvs = append(kvs, delGameAddrIndex(types.GameActionCreate, log.CreateAddr, prevIndex))
		kvs = append(kvs, delGameStatusIndex(types.GameActionCreate, prevIndex))
	}
	if log.Status == types.GameActionCancel {
		prevIndex, err := g.queryIndex(log.GetGameId())
		if err != nil {
			panic(err) //数据损毁
		}
		kvs = append(kvs, delGameAddrIndex(types.GameActionCreate, log.CreateAddr, prevIndex))
		kvs = append(kvs, delGameStatusIndex(types.GameActionCreate, prevIndex))
	}

	if log.Status == types.GameActionClose {
		kvs = append(kvs, addGameAddrIndex(log.Status, log.GameId, log.MatchAddr, log.Index))
		prevIndex, err := g.queryIndex(log.GetMatchTxHash())
		if err != nil {
			panic(err) //数据损毁
		}
		kvs = append(kvs, delGameAddrIndex(types.GameActionMatch, log.MatchAddr, prevIndex))
		kvs = append(kvs, delGameAddrIndex(types.GameActionMatch, log.CreateAddr, prevIndex))
		kvs = append(kvs, delGameStatusIndex(types.GameActionMatch, prevIndex))
	}
	return kvs
}

//回滚索引
func (g *Game) rollbackIndex(log *types.ReceiptGame) (kvs []*types.KeyValue) {
	//先删除本次Action产生的索引
	kvs = append(kvs, delGameAddrIndex(log.Status, log.Addr, log.Index))
	kvs = append(kvs, delGameStatusIndex(log.Status, log.Index))

	if log.Status == types.GameActionMatch {
		kvs = append(kvs, delGameAddrIndex(log.Status, log.CreateAddr, log.Index))

		prevIndex, err := g.queryIndex(log.GetGameId())
		if err != nil {
			panic(err) //数据损毁
		}
		kvs = append(kvs, addGameAddrIndex(types.GameActionCreate, log.GameId, log.CreateAddr, prevIndex))
		kvs = append(kvs, addGameStatusIndex(types.GameActionCreate, log.GameId, prevIndex))
	}
	if log.Status == types.GameActionCancel {
		prevIndex, err := g.queryIndex(log.GetGameId())
		if err != nil {
			panic(err) //数据损毁
		}
		kvs = append(kvs, addGameAddrIndex(types.GameActionCreate, log.GameId, log.CreateAddr, prevIndex))
		kvs = append(kvs, addGameStatusIndex(types.GameActionCreate, log.GameId, prevIndex))
	}

	if log.Status == types.GameActionClose {
		kvs = append(kvs, delGameAddrIndex(log.Status, log.MatchAddr, log.Index))

		prevIndex, err := g.queryIndex(log.GetMatchTxHash())
		if err != nil {
			panic(err) //数据损毁
		}
		kvs = append(kvs, addGameAddrIndex(types.GameActionMatch, log.GameId, log.MatchAddr, prevIndex))
		kvs = append(kvs, addGameAddrIndex(types.GameActionMatch, log.GameId, log.CreateAddr, prevIndex))
		kvs = append(kvs, addGameStatusIndex(types.GameActionMatch, log.GameId, prevIndex))
	}
	return kvs
}

//根据txhash查询 index
func (g *Game) queryIndex(txHash string) (string, error) {
	var data types.ReqHash
	hash, err := common.FromHex(txHash)
	if err != nil {
		return "", err
	}
	data.Hash = hash
	d, err := g.GetApi().QueryTx(&data)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%018d", d.GetHeight()*types.MaxTxsPerBlock+int64(d.GetIndex())), nil
}
func (g *Game) Query(funcName string, params []byte) (types.Message, error) {
	if funcName == FuncName_QueryGameListByIds {
		var info types.QueryGameInfos
		err := types.Decode(params, &info)
		if err != nil {
			return nil, err
		}
		return Infos(g.GetStateDB(), &info)
	} else if funcName == FuncName_QueryGameById {
		var gameInfo types.QueryGameInfo
		err := types.Decode(params, &gameInfo)
		if err != nil {
			return nil, err
		}
		game, err := readGame(g.GetStateDB(), gameInfo.GetGameId())
		if err != nil {
			return nil, err
		}
		return &types.ReplyGame{game}, nil
	} else if funcName == FuncName_QueryGameListByStatusAndAddr {
		var q types.QueryGameListByStatusAndAddr
		err := types.Decode(params, &q)
		if err != nil {
			return nil, err
		}
		return List(g.GetLocalDB(), g.GetStateDB(), &q)
	} else if funcName == FuncName_QueryGameListCount {
		var q types.QueryGameListCount
		err := types.Decode(params, &q)
		if err != nil {
			return nil, err
		}
		return QueryGameListCount(g.GetLocalDB(), g.GetStateDB(), &q)
	}
	return nil, types.ErrActionNotSupport
}

func calcGameStatusIndexKey(status int32, index string) []byte {
	key := fmt.Sprintf("game-status:%d:%s", status, index)
	return []byte(key)
}

func calcGameStatusIndexPrefix(status int32) []byte {
	key := fmt.Sprintf("game-status:%d:%s", status)
	return []byte(key)
}
func calcGameAddrIndexKey(status int32, addr, index string) []byte {
	key := fmt.Sprintf("game-addr:%d:%s:%s", status, addr, index)
	return []byte(key)
}
func calcGameAddrIndexPrefix(status int32, addr string) []byte {
	key := fmt.Sprintf("game-addr:%d:%s", status, addr)
	return []byte(key)
}
func addGameStatusIndex(status int32, gameId, index string) *types.KeyValue {
	kv := &types.KeyValue{}
	kv.Key = calcGameStatusIndexKey(status, index)
	record := &types.GameRecord{
		GameId: gameId,
		Index:  index,
	}
	kv.Value = []byte(types.Encode(record))
	return kv
}
func addGameAddrIndex(status int32, gameId, addr, index string) *types.KeyValue {
	kv := &types.KeyValue{}
	kv.Key = calcGameAddrIndexKey(status, addr, index)
	record := &types.GameRecord{
		GameId: gameId,
		Index:  index,
	}
	kv.Value = []byte(types.Encode(record))
	return kv
}
func delGameStatusIndex(status int32, index string) *types.KeyValue {
	kv := &types.KeyValue{}
	kv.Key = calcGameStatusIndexKey(status, index)
	kv.Value = nil
	return kv
}
func delGameAddrIndex(status int32, addr, index string) *types.KeyValue {
	kv := &types.KeyValue{}
	kv.Key = calcGameAddrIndexKey(status, addr, index)
	//value置nil,提交时，会自动执行删除操作
	kv.Value = nil
	return kv
}
