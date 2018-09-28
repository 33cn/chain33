package executor

import (
     log "github.com/inconshreveable/log15"
	"fmt"
	pkt "gitlab.33.cn/chain33/chain33/plugin/dapp/pokerbull/types"
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
)

var logger = log.New("module", "execs.pokerbull")

func Init(name string) {
	drivers.Register(newGame().GetName(), newGame, 0)
}

type PokerBull struct {
	drivers.DriverBase
}

func newGame() drivers.Driver {
	t := &PokerBull{}
	t.SetChild(t)
	return t
}

func GetName() string {
	return newGame().GetName()
}

func (g *PokerBull) GetDriverName() string {
	return pkt.PokerBullX
}

func (g *PokerBull) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	var action pkt.PBGameAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return nil, err
	}
	logger.Debug("exec PokerBull tx=", "tx=", action)
	actiondb := NewAction(g, tx, index)
	if action.Ty == pkt.PBGameActionStart && action.GetStart() != nil {
		return actiondb.GameStart(action.GetStart())
	} else if action.Ty == pkt.PBGameActionContinue && action.GetContinue() != nil {
		return actiondb.GameContinue(action.GetContinue())
	} else if action.Ty == pkt.PBGameActionQuit && action.GetQuit() != nil {
		return actiondb.GameQuit(action.GetQuit())
	}
	return nil, types.ErrActionNotSupport
}

//更新索引
func (g *PokerBull) updateIndex(log *pkt.ReceiptPBGame) (kvs []*types.KeyValue) {
	//先保存本次Action产生的索引
	kvs = append(kvs, addPBGameStatus(log.Status, log.PlayerNum, log.Index, log.GameId))

	//状态更新
	if log.Status == pkt.PBGameActionContinue {
		kvs = append(kvs, delPBGameStatus(pkt.PBGameActionStart, log.PlayerNum, log.PrevIndex))
	}

	if log.Status == pkt.PBGameActionQuit {
		kvs = append(kvs, delPBGameStatus(pkt.PBGameActionStart, log.PlayerNum, log.PrevIndex))
		kvs = append(kvs, delPBGameStatus(pkt.PBGameActionContinue, log.PlayerNum, log.PrevIndex))
	}
	return kvs
}

func (g *PokerBull) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := g.DriverBase.ExecLocal(tx, receipt, index)
	if err != nil {
		return nil, err
	}
	if receipt.GetTy() != types.ExecOk {
		return set, nil
	}
	for i := 0; i < len(receipt.Logs); i++ {
		item := receipt.Logs[i]
		if item.Ty == pkt.TyLogPBGameStart || item.Ty == pkt.TyLogPBGameContinue || item.Ty == pkt.TyLogPBGameQuit {
			var Gamelog pkt.ReceiptPBGame
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

func (g *PokerBull) rollbackIndex(log *pkt.ReceiptPBGame) (kvs []*types.KeyValue) {
	kvs = append(kvs, delPBGameStatus(log.Status, log.PlayerNum, log.Index))

	if log.Status == pkt.PBGameActionContinue {
		kvs = append(kvs, addPBGameStatus(log.Status, log.PlayerNum, log.PrevIndex, log.GameId))
	}

	if log.Status == pkt.PBGameActionQuit {
		kvs = append(kvs, addPBGameStatus(pkt.PBGameActionContinue, log.PlayerNum, log.PrevIndex, log.GameId))
	}
	return kvs
}

func (g *PokerBull) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
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
			var Gamelog pkt.ReceiptPBGame
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

func (g *PokerBull) Query(funcName string, params []byte) (types.Message, error) {
	if funcName == pkt.FuncName_QueryGameListByIds {
		var info pkt.QueryPBGameInfos
		err := types.Decode(params, &info)
		if err != nil {
			return nil, err
		}
		return Infos(g.GetStateDB(), &info)
	} else if funcName == pkt.FuncName_QueryGameById {
		var gameInfo pkt.QueryPBGameInfo
		err := types.Decode(params, &gameInfo)
		if err != nil {
			return nil, err
		}
		game, err := readGame(g.GetStateDB(), gameInfo.GetGameId())
		if err != nil {
			return nil, err
		}
		return &pkt.ReplyPBGame{game}, nil
	}

	return nil, types.ErrActionNotSupport
}

func calcPBGameStatusKey(status int32, player int32, index int64) []byte {
	key := fmt.Sprintf("PBgame-status:%d:%d:%d", status, player, index)
	return []byte(key)
}

func calcPBGameStatusPrefix(status int32) []byte {
	key := fmt.Sprintf("PBgame-status:%d:", status)
	return []byte(key)
}

func calcPBGameStatusAndPlayerPrefix(status int32, player int32) []byte {
	key := fmt.Sprintf("PBgame-status:%d:%d:", status, player)
	return []byte(key)
}

func addPBGameStatus(status int32, player int32, index int64, gameId string) *types.KeyValue {
	kv := &types.KeyValue{}
	kv.Key = calcPBGameStatusKey(status, player, index)
	record := &pkt.PBGameRecord{
		GameId: gameId,
	}
	kv.Value = types.Encode(record)
	return kv
}

func delPBGameStatus(status int32, player int32, index int64) *types.KeyValue {
	kv := &types.KeyValue{}
	kv.Key = calcPBGameStatusKey(status, player, index)
	kv.Value = nil
	return kv
}
