package executor

import (
	"encoding/json"
	"math/rand"
	"time"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/types"
	gt "gitlab.33.cn/chain33/chain33/pluginmanager/plugins/game/types"
)


var name string

var tlog = log.New("module", name)

//getRealExecName
//如果paraName == "", 那么自动用 types.ExecName("game")
//如果设置了paraName , 那么强制用paraName
//也就是说，我们可以构造其他平行链的交易
func getRealExecName(paraName string) string {
	return types.ExecName(paraName + gt.GameX)
}

func Init() {
	name = types.ExecName(gt.GameX)
	// init executor type
	types.RegistorExecutor(types.ExecName(name), &GameType{})

	// init log
	types.RegistorLog(types.TyLogCreateGame, &CreateGameLog{})
	types.RegistorLog(types.TyLogCancleGame, &CancelGameLog{})
	types.RegistorLog(types.TyLogMatchGame, &MatchGameLog{})
	types.RegistorLog(types.TyLogCloseGame, &CloseGameLog{})

	// init query rpc
	types.RegistorRpcType(FuncName_QueryGameListByIds, &GameGetList{})
	types.RegistorRpcType(FuncName_QueryGameById, &GameGetInfo{})
	types.RegistorRpcType(FuncName_QueryGameListByStatusAndAddr, &GameQueryList{})
	types.RegistorRpcType(FuncName_QueryGameListCount, &GameQueryListCount{})
}

// exec
type GameType struct {
	types.ExecTypeBase
}

func (game GameType) GetRealToAddr(tx *types.Transaction) string {
	if string(tx.Execer) == gt.GameX {
		return tx.To
	}
	var action gt.GameAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return tx.To
	}
	return tx.To
}

func (game GameType) ActionName(tx *types.Transaction) string {
	var action gt.GameAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return "unknown-err"
	}

	if action.Ty == gt.GameActionCreate && action.GetCreate() != nil {
		return Action_CreateGame
	} else if action.Ty == gt.GameActionMatch && action.GetMatch() != nil {
		return Action_MatchGame
	} else if action.Ty == gt.GameActionCancel && action.GetCancel() != nil {
		return Action_CancelGame
	} else if action.Ty == gt.GameActionClose && action.GetClose() != nil {
		return Action_CloseGame
	}
	return "unknown"
}
func (game GameType) Amount(tx *types.Transaction) (int64, error) {
	return 0, nil
}

// TODO createTx接口暂时没法用，作为一个预留接口
func (game GameType) CreateTx(action string, message json.RawMessage) (*types.Transaction, error) {
	tlog.Debug("Game.CreateTx", "action", action)
	var tx *types.Transaction
	if action == Action_CreateGame {
		var param GamePreCreateTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			tlog.Error("CreateTx", "Error", err)
			return nil, types.ErrInputPara
		}

		return CreateRawGamePreCreateTx(&param)
	} else if action == Action_MatchGame {
		var param GamePreMatchTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			tlog.Error("CreateTx", "Error", err)
			return nil, types.ErrInputPara
		}

		return CreateRawGamePreMatchTx(&param)
	} else if action == Action_CancelGame {
		var param GamePreCancelTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			tlog.Error("CreateTx", "Error", err)
			return nil, types.ErrInputPara
		}

		return CreateRawGamePreCancelTx(&param)
	} else if action == Action_CloseGame {
		var param GamePreCloseTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			tlog.Error("CreateTx", "Error", err)
			return nil, types.ErrInputPara
		}

		return CreateRawGamePreCloseTx(&param)
	} else {
		return nil, types.ErrNotSupport
	}

	return tx, nil
}

func CreateRawGamePreCreateTx(parm *GamePreCreateTx) (*types.Transaction, error) {
	if parm == nil {
		tlog.Error("CreateRawGamePreCreateTx", "parm", parm)
		return nil, types.ErrInvalidParam
	}
	v := &gt.GameCreate{
		Value:     parm.Amount,
		HashType:  parm.HashType,
		HashValue: parm.HashValue,
	}
	precreate := &gt.GameAction{
		Ty:    gt.GameActionCreate,
		Value: &gt.GameAction_Create{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(getRealExecName(types.GetParaName())),
		Payload: types.Encode(precreate),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(getRealExecName(types.GetParaName())),
	}

	tx.SetRealFee(types.MinFee)

	return tx, nil
}

func CreateRawGamePreMatchTx(parm *GamePreMatchTx) (*types.Transaction, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}

	v := &gt.GameMatch{
		GameId: parm.GameId,
		Guess:  parm.Guess,
	}
	game := &gt.GameAction{
		Ty:    gt.GameActionMatch,
		Value: &gt.GameAction_Match{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(getRealExecName(types.GetParaName())),
		Payload: types.Encode(game),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(getRealExecName(types.GetParaName())),
	}

	tx.SetRealFee(types.MinFee)

	return tx, nil
}

func CreateRawGamePreCancelTx(parm *GamePreCancelTx) (*types.Transaction, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &gt.GameCancel{
		GameId: parm.GameId,
	}
	cancel := &gt.GameAction{
		Ty:    gt.GameActionCancel,
		Value: &gt.GameAction_Cancel{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(getRealExecName(types.GetParaName())),
		Payload: types.Encode(cancel),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(getRealExecName(types.GetParaName())),
	}

	tx.SetRealFee(types.MinFee)

	return tx, nil
}

//CreateRawGamePreCloseTx
func CreateRawGamePreCloseTx(parm *GamePreCloseTx) (*types.Transaction, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &gt.GameClose{
		GameId: parm.GameId,
		Secret: parm.Secret,
	}
	close := &gt.GameAction{
		Ty:    gt.GameActionClose,
		Value: &gt.GameAction_Close{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(getRealExecName(types.GetParaName())),
		Payload: types.Encode(close),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(getRealExecName(types.GetParaName())),
	}

	tx.SetRealFee(types.MinFee)

	return tx, nil
}

//log
type CreateGameLog struct {
}

func (l CreateGameLog) Name() string {
	return "LogGameCreate"
}

func (l CreateGameLog) Decode(msg []byte) (interface{}, error) {
	var logTmp gt.ReceiptGame
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type MatchGameLog struct {
}

func (l MatchGameLog) Name() string {
	return "LogMatchGame"
}

func (l MatchGameLog) Decode(msg []byte) (interface{}, error) {
	var logTmp gt.ReceiptGame
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type CancelGameLog struct {
}

func (l CancelGameLog) Name() string {
	return "LogCancelGame"
}

func (l CancelGameLog) Decode(msg []byte) (interface{}, error) {
	var logTmp gt.ReceiptGame
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type CloseGameLog struct {
}

func (l CloseGameLog) Name() string {
	return "LogCloseGame"
}

func (l CloseGameLog) Decode(msg []byte) (interface{}, error) {
	var logTmp gt.ReceiptGame
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

// query
type GameGetList struct {
}

func (g *GameGetList) Input(message json.RawMessage) ([]byte, error) {
	var req gt.QueryGameInfos
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *GameGetList) Output(reply interface{}) (interface{}, error) {
	if replyData, ok := reply.(*types.Message); ok {
		if replyGameList, ok := (*replyData).(*gt.ReplyGameList); ok {
			var gameList []*GameData
			for _, game := range replyGameList.GetGames() {
				g := &GameData{
					GameId:        game.GetGameId(),
					Status:        game.GetStatus(),
					CreateAddress: game.GetCreateAddress(),
					MatchAddress:  game.GetMatchAddress(),
					CreateTime:    game.GetCreateTime(),
					MatchTime:     game.GetMatchTime(),
					Closetime:     game.GetClosetime(),
					Value:         game.GetValue(),
					HashType:      game.GetHashType(),
					HashValue:     game.GetHashValue(),
					Secret:        game.GetSecret(),
					Result:        game.GetResult(),
					MatcherGuess:  game.GetMatcherGuess(),
					CreateTxHash:  game.GetCreateTxHash(),
					CancelTxHash:  game.GetCancelTxHash(),
					MatchTxHash:   game.GetMatchTxHash(),
					CloseTxHash:   game.GetCloseTxHash(),
					CreatorGuess:  game.GetCreatorGuess(),
					Index:         game.GetIndex(),
				}
				gameList = append(gameList, g)
			}
			return gameList, nil
		}
	}
	return reply, nil
}

type GameQueryListCount struct {
}

func (g *GameQueryListCount) Input(message json.RawMessage) ([]byte, error) {
	var req gt.QueryGameListCount
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (g *GameQueryListCount) Output(reply interface{}) (interface{}, error) {
	if replyData, ok := reply.(*types.Message); ok {
		if replyCount, ok := (*replyData).(*gt.ReplyGameListCount); ok {
			count := replyCount.GetCount()
			return count, nil
		}
	}
	return reply, nil
}

type GameGetInfo struct {
}

func (g *GameGetInfo) Input(message json.RawMessage) ([]byte, error) {
	var req gt.QueryGameInfo
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (g *GameGetInfo) Output(reply interface{}) (interface{}, error) {
	if replyData, ok := reply.(*types.Message); ok {
		if replyGame, ok := (*replyData).(*gt.ReplyGame); ok {
			game := replyGame.GetGame()
			g := &GameData{
				GameId:        game.GetGameId(),
				Status:        game.GetStatus(),
				CreateAddress: game.GetCreateAddress(),
				MatchAddress:  game.GetMatchAddress(),
				CreateTime:    game.GetCreateTime(),
				MatchTime:     game.GetMatchTime(),
				Closetime:     game.GetClosetime(),
				Value:         game.GetValue(),
				HashType:      game.GetHashType(),
				HashValue:     game.GetHashValue(),
				Secret:        game.GetSecret(),
				Result:        game.GetResult(),
				MatcherGuess:  game.GetMatcherGuess(),
				CreateTxHash:  game.GetCreateTxHash(),
				CancelTxHash:  game.GetCancelTxHash(),
				MatchTxHash:   game.GetMatchTxHash(),
				CloseTxHash:   game.GetCloseTxHash(),
				CreatorGuess:  game.GetCreatorGuess(),
				Index:         game.GetIndex(),
			}
			return g, nil
		}
	}
	return reply, nil
}

type GameQueryList struct {
}

func (g *GameQueryList) Input(message json.RawMessage) ([]byte, error) {
	var req gt.QueryGameListByStatusAndAddr
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (g *GameQueryList) Output(reply interface{}) (interface{}, error) {
	if replyData, ok := reply.(*types.Message); ok {
		if replyGameList, ok := (*replyData).(*gt.ReplyGameList); ok {
			var gameList []*GameData
			for _, game := range replyGameList.GetGames() {
				g := &GameData{
					GameId:        game.GetGameId(),
					Status:        game.GetStatus(),
					CreateAddress: game.GetCreateAddress(),
					MatchAddress:  game.GetMatchAddress(),
					CreateTime:    game.GetCreateTime(),
					MatchTime:     game.GetMatchTime(),
					Closetime:     game.GetClosetime(),
					Value:         game.GetValue(),
					HashType:      game.GetHashType(),
					HashValue:     game.GetHashValue(),
					Secret:        game.GetSecret(),
					Result:        game.GetResult(),
					MatcherGuess:  game.GetMatcherGuess(),
					CreateTxHash:  game.GetCreateTxHash(),
					CancelTxHash:  game.GetCancelTxHash(),
					MatchTxHash:   game.GetMatchTxHash(),
					CloseTxHash:   game.GetCloseTxHash(),
					CreatorGuess:  game.GetCreatorGuess(),
					Index:         game.GetIndex(),
				}
				gameList = append(gameList, g)
			}
			return gameList, nil
		}
	}
	return reply, nil
}
