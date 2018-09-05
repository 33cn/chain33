package executor

import (
	"fmt"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	gt "gitlab.33.cn/chain33/chain33/pluginmanager/plugins/game/types"
	"gitlab.33.cn/chain33/chain33/types"
)

var glog = log.New("module", "execs.game")

func init() {
	//manager.RegisterExecutor(newGame().GetName(), newGame, 0)
}

type Game struct {
	drivers.DriverBase
}

func NewGame() drivers.Driver {
	t := &Game{}
	t.SetChild(t)
	return t
}

func GetName() string {
	return types.ExecName(gt.GameX)
}

func (g *Game) GetName() string {
	return GetName()
}

func (g *Game) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	var action gt.GameAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return nil, err
	}
	glog.Debug("exec Game tx=", "tx=", action)
	actiondb := NewAction(g, tx, index)
	if action.Ty == gt.GameActionCreate && action.GetCreate() != nil {
		return actiondb.GameCreate(action.GetCreate())
	} else if action.Ty == gt.GameActionCancel && action.GetCancel() != nil {
		return actiondb.GameCancel(action.GetCancel())
	} else if action.Ty == gt.GameActionClose && action.GetClose() != nil {
		return actiondb.GameClose(action.GetClose())
	} else if action.Ty == gt.GameActionMatch && action.GetMatch() != nil {
		return actiondb.GameMatch(action.GetMatch())
	}
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
			var Gamelog gt.ReceiptGame
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
			var Gamelog gt.ReceiptGame
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
func (g *Game) updateIndex(log *gt.ReceiptGame) (kvs []*types.KeyValue) {
	//先保存本次Action产生的索引
	kvs = append(kvs, addGameAddrIndex(log.Status, log.GameId, log.Addr, log.Index))
	kvs = append(kvs, addGameStatusIndex(log.Status, log.GameId, log.Index))
	if log.Status == gt.GameActionMatch {
		kvs = append(kvs, addGameAddrIndex(log.Status, log.GameId, log.CreateAddr, log.Index))
		kvs = append(kvs, delGameAddrIndex(gt.GameActionCreate, log.CreateAddr, log.PrevIndex))
		kvs = append(kvs, delGameStatusIndex(gt.GameActionCreate, log.PrevIndex))
	}
	if log.Status == gt.GameActionCancel {
		kvs = append(kvs, delGameAddrIndex(gt.GameActionCreate, log.CreateAddr, log.PrevIndex))
		kvs = append(kvs, delGameStatusIndex(gt.GameActionCreate, log.PrevIndex))
	}

	if log.Status == gt.GameActionClose {
		kvs = append(kvs, addGameAddrIndex(log.Status, log.GameId, log.MatchAddr, log.Index))
		kvs = append(kvs, delGameAddrIndex(gt.GameActionMatch, log.MatchAddr, log.PrevIndex))
		kvs = append(kvs, delGameAddrIndex(gt.GameActionMatch, log.CreateAddr, log.PrevIndex))
		kvs = append(kvs, delGameStatusIndex(gt.GameActionMatch, log.PrevIndex))
	}
	return kvs
}

//回滚索引
func (g *Game) rollbackIndex(log *gt.ReceiptGame) (kvs []*types.KeyValue) {
	//先删除本次Action产生的索引
	kvs = append(kvs, delGameAddrIndex(log.Status, log.Addr, log.Index))
	kvs = append(kvs, delGameStatusIndex(log.Status, log.Index))

	if log.Status == gt.GameActionMatch {
		kvs = append(kvs, delGameAddrIndex(log.Status, log.CreateAddr, log.Index))
		kvs = append(kvs, addGameAddrIndex(gt.GameActionCreate, log.GameId, log.CreateAddr, log.PrevIndex))
		kvs = append(kvs, addGameStatusIndex(gt.GameActionCreate, log.GameId, log.PrevIndex))
	}

	if log.Status == gt.GameActionCancel {
		kvs = append(kvs, addGameAddrIndex(gt.GameActionCreate, log.GameId, log.CreateAddr, log.PrevIndex))
		kvs = append(kvs, addGameStatusIndex(gt.GameActionCreate, log.GameId, log.PrevIndex))
	}

	if log.Status == gt.GameActionClose {
		kvs = append(kvs, delGameAddrIndex(log.Status, log.MatchAddr, log.Index))
		kvs = append(kvs, addGameAddrIndex(gt.GameActionMatch, log.GameId, log.MatchAddr, log.PrevIndex))
		kvs = append(kvs, addGameAddrIndex(gt.GameActionMatch, log.GameId, log.CreateAddr, log.PrevIndex))
		kvs = append(kvs, addGameStatusIndex(gt.GameActionMatch, log.GameId, log.PrevIndex))
	}
	return kvs
}
func (g *Game) Query(funcName string, params []byte) (types.Message, error) {
	if funcName == FuncName_QueryGameListByIds {
		var info gt.QueryGameInfos
		err := types.Decode(params, &info)
		if err != nil {
			return nil, err
		}
		return Infos(g.GetStateDB(), &info)
	} else if funcName == FuncName_QueryGameById {
		var gameInfo gt.QueryGameInfo
		err := types.Decode(params, &gameInfo)
		if err != nil {
			return nil, err
		}
		game, err := readGame(g.GetStateDB(), gameInfo.GetGameId())
		if err != nil {
			return nil, err
		}
		return &gt.ReplyGame{game}, nil
	} else if funcName == FuncName_QueryGameListByStatusAndAddr {
		var q gt.QueryGameListByStatusAndAddr
		err := types.Decode(params, &q)
		if err != nil {
			return nil, err
		}
		return List(g.GetLocalDB(), g.GetStateDB(), &q)
	} else if funcName == FuncName_QueryGameListCount {
		var q gt.QueryGameListCount
		err := types.Decode(params, &q)
		if err != nil {
			return nil, err
		}
		return QueryGameListCount(g.GetStateDB(), &q)
	}
	return nil, types.ErrActionNotSupport
}

func calcGameStatusIndexKey(status int32, index int64) []byte {
	key := fmt.Sprintf("game-status:%d:%018d", status, index)
	return []byte(key)
}

func calcGameStatusIndexPrefix(status int32) []byte {
	key := fmt.Sprintf("game-status:%d:", status)
	return []byte(key)
}
func calcGameAddrIndexKey(status int32, addr string, index int64) []byte {
	key := fmt.Sprintf("game-addr:%d:%s:%018d", status, addr, index)
	return []byte(key)
}
func calcGameAddrIndexPrefix(status int32, addr string) []byte {
	key := fmt.Sprintf("game-addr:%d:%s:", status, addr)
	return []byte(key)
}
func addGameStatusIndex(status int32, gameId string, index int64) *types.KeyValue {
	kv := &types.KeyValue{}
	kv.Key = calcGameStatusIndexKey(status, index)
	record := &gt.GameRecord{
		GameId: gameId,
		Index:  index,
	}
	kv.Value = types.Encode(record)
	return kv
}
func addGameAddrIndex(status int32, gameId, addr string, index int64) *types.KeyValue {
	kv := &types.KeyValue{}
	kv.Key = calcGameAddrIndexKey(status, addr, index)
	record := &gt.GameRecord{
		GameId: gameId,
		Index:  index,
	}
	kv.Value = types.Encode(record)
	return kv
}
func delGameStatusIndex(status int32, index int64) *types.KeyValue {
	kv := &types.KeyValue{}
	kv.Key = calcGameStatusIndexKey(status, index)
	kv.Value = nil
	return kv
}
func delGameAddrIndex(status int32, addr string, index int64) *types.KeyValue {
	kv := &types.KeyValue{}
	kv.Key = calcGameAddrIndexKey(status, addr, index)
	//value置nil,提交时，会自动执行删除操作
	kv.Value = nil
	return kv
}
