package game

import (
	"fmt"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/types"
)

var glog = log.New("module", "execs.game")

const (
	MaxGameAmount = 100 * types.Coin
	//查询方法名
	FuncName_QueryGameListByIds = "QueryGameListByIds"
	//FuncName_QueryGameListByStatus = "QueryGameListByStatus"
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
	actiondb := NewAction(g, tx)
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
			kv := g.updateGame(&Gamelog)
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
			kv := g.rollbackGame(&Gamelog)
			set.KV = append(set.KV, kv...)
		}
	}
	return set, nil
}

//更新索引
func (g *Game) updateGame(gamelog *types.ReceiptGame) (kvs []*types.KeyValue) {
	//先保存本次Action产生的索引
	kvs = append(kvs, addGame(gamelog.GameId, gamelog.Status, gamelog.Addr, gamelog.ActionTime))
	if gamelog.Status < types.GameActionClose {
		kvs = append(kvs, delGame(gamelog.GetGameId(), gamelog.GetPrevStatus(), gamelog.GetCreateAddr(), gamelog.PrevActionTime))
	}
	if gamelog.Status == types.GameActionMatch {
		kvs = append(kvs, addGame(gamelog.GetGameId(), gamelog.Status, gamelog.GetCreateAddr(), gamelog.ActionTime))
	} else if gamelog.Status == types.GameActionClose {
		kvs = append(kvs, delGame(gamelog.GetGameId(), gamelog.GetPrevStatus(), gamelog.GetMatchAddr(), gamelog.PrevActionTime))
		kvs = append(kvs, addGame(gamelog.GetGameId(), gamelog.Status, gamelog.GetMatchAddr(), gamelog.ActionTime))
	}
	return kvs
}

//回滚索引
func (g *Game) rollbackGame(gamelog *types.ReceiptGame) (kvs []*types.KeyValue) {
	//先删除本次Action产生的索引
	kvs = append(kvs, delGame(gamelog.GameId, gamelog.Status, gamelog.Addr, gamelog.ActionTime))
	if gamelog.PrevStatus > types.GameActionCreate {
		kvs = append(kvs, addGame(gamelog.GetGameId(), gamelog.GetPrevStatus(), gamelog.GetCreateAddr(), gamelog.PrevActionTime))
	}
	if gamelog.PrevStatus == types.GameActionCreate {
		kvs = append(kvs, delGame(gamelog.GetGameId(), gamelog.GetStatus(), gamelog.GetCreateAddr(), gamelog.ActionTime))
	} else if gamelog.PrevStatus == types.GameActionMatch {
		//需要同时更新create,match状态索引
		kvs = append(kvs, delGame(gamelog.GetGameId(), gamelog.GetStatus(), gamelog.GetMatchAddr(), gamelog.ActionTime))
		kvs = append(kvs, addGame(gamelog.GetGameId(), gamelog.GetPrevStatus(), gamelog.GetMatchAddr(), gamelog.PrevActionTime))
	}
	return kvs
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
	}
	return nil, types.ErrActionNotSupport
}

func calcGameKey(gameId string, status int32, addr string, actionTime int64) []byte {
	key := fmt.Sprintf("game-gl:%d:%s:%d:%s", status, addr, actionTime, gameId)
	return []byte(key)
}

func calcGamePrefix(status int32, addr string) []byte {
	key := fmt.Sprintf("game-gl:%d:%s", status, addr)
	return []byte(key)
}

func addGame(gameId string, status int32, addr string, actionTime int64) *types.KeyValue {
	kv := &types.KeyValue{}
	kv.Key = calcGameKey(gameId, status, addr, actionTime)
	kv.Value = []byte(gameId)
	return kv
}

func delGame(gameId string, status int32, addr string, actionTime int64) *types.KeyValue {
	kv := &types.KeyValue{}
	kv.Key = calcGameKey(gameId, status, addr, actionTime)
	//value置nil,提交时，会自动执行删除操作
	kv.Value = nil
	return kv
}
