package game

import (
	"fmt"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/types"
)

var glog = log.New("module", "execs.game")

const MaxGameAmount = 100 * types.Coin

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

func (t *Game) GetName() string {
	return "game"
}

func (t *Game) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	var action types.GameAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return nil, err
	}
	glog.Debug("exec Game tx=", "tx=", action)
	actiondb := NewAction(t, tx)
	if action.Ty == types.GameActionOpen && action.GetOpen() != nil {
		open := action.GetOpen()
		if open.Amount > MaxGameAmount {
			glog.Error("open ", "value", open.Amount)
			return nil, types.ErrGameOpenAmount
		}
		return actiondb.GameOpen(open)
	} else if action.Ty == types.GameActionCancel && action.GetCancel() != nil {
		return actiondb.GameCancel(action.GetCancel())
	} else if action.Ty == types.GameActionClose && action.GetClose() != nil {
		return actiondb.GameClose(action.GetClose())
	} else if action.Ty == types.GameActionMatch && action.GetMatch() != nil {
		return actiondb.GameMatch(action.GetMatch(), index)
	}
	//return error
	return nil, types.ErrActionNotSupport
}

func (t *Game) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := t.DriverBase.ExecLocal(tx, receipt, index)
	if err != nil {
		return nil, err
	}
	if receipt.GetTy() != types.ExecOk {
		return set, nil
	}
	for i := 0; i < len(receipt.Logs); i++ {
		item := receipt.Logs[i]
		//这三个是Game 的log
		if item.Ty == types.TyLogNewGame || item.Ty == types.TyLogMatchGame || item.Ty == types.TyLogCloseGame {
			var Gamelog types.ReceiptGame
			err := types.Decode(item.Log, &Gamelog)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			kv := t.saveGame(&Gamelog)
			set.KV = append(set.KV, kv...)
		}
	}
	return set, nil
}

func (t *Game) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := t.DriverBase.ExecDelLocal(tx, receipt, index)
	if err != nil {
		return nil, err
	}
	if receipt.GetTy() != types.ExecOk {
		return set, nil
	}
	for i := 0; i < len(receipt.Logs); i++ {
		item := receipt.Logs[i]
		//这三个是Game 的log
		if item.Ty == types.TyLogNewGame || item.Ty == types.TyLogMatchGame || item.Ty == types.TyLogCloseGame {
			var Gamelog types.ReceiptGame
			err := types.Decode(item.Log, &Gamelog)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			kv := t.delGame(&Gamelog)
			set.KV = append(set.KV, kv...)
		}
	}
	return set, nil
}

func (t *Game) saveGame(gamelog *types.ReceiptGame) (kvs []*types.KeyValue) {
	if gamelog.PrevStatus > 0 {
		kv := delGame(gamelog.Addr, gamelog.GameId, gamelog.PrevStatus)
		kvs = append(kvs, kv)
	}
	kvs = append(kvs, addGame(gamelog.Addr, gamelog.GameId, gamelog.Status))
	return kvs
}

func (t *Game) delGame(gamelog *types.ReceiptGame) (kvs []*types.KeyValue) {
	if gamelog.PrevStatus > 0 {
		kv := addGame(gamelog.Addr, gamelog.GameId, gamelog.PrevStatus)
		kvs = append(kvs, kv)
	}
	kvs = append(kvs, delGame(gamelog.Addr, gamelog.GameId, gamelog.Status))
	return kvs
}

func (t *Game) Query(funcname string, params []byte) (types.Message, error) {
	if funcname == "GameInfos" {
		var info types.GameInfos
		err := types.Decode(params, &info)
		if err != nil {
			return nil, err
		}
		return Infos(t.GetStateDB(), &info)
	} else if funcname == "GameList" {
		var l types.GameList
		err := types.Decode(params, &l)
		if err != nil {
			return nil, err
		}
		return List(t.GetLocalDB(), t.GetStateDB(), &l)
	}
	return nil, types.ErrActionNotSupport
}

func calcGameKey(addr string, gameID string, status int32) []byte {
	key := fmt.Sprintf("game-gl:%s:%d:%s", addr, status, gameID)
	return []byte(key)
}

func calcGamePrefix(addr string, status int32) []byte {
	key := fmt.Sprintf("game-gl:%s:%d", addr, status)
	return []byte(key)
}

func addGame(addr string, gameID string, status int32) *types.KeyValue {
	kv := &types.KeyValue{}
	kv.Key = calcGameKey(addr, gameID, status)
	kv.Value = []byte(gameID)
	return kv
}

func delGame(addr string, gameID string, status int32) *types.KeyValue {
	kv := &types.KeyValue{}
	kv.Key = calcGameKey(addr, gameID, status)
	kv.Value = nil
	return kv
}
