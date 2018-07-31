package game

import (
	"encoding/json"
	"math/rand"
	"time"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/types"
)

const (
	FuncName_QueryGameListByIds = "QueryGameListByIds"
	//FuncName_QueryGameListByStatus = "QueryGameListByStatus"
	FuncName_QueryGameListByStatusAndAddr = "QueryGameListByStatusAndAddr"
	FuncName_QueryGameById                = "QueryGameById"

	Action_CreateGame = "createGame"
	Action_MatchGame  = "matchGame"
	Action_CancelGame = "cancelGame"
	Action_CloseGame  = "closeGame"
)

var name string

var tlog = log.New("module", name)

//getRealExecName
//如果paraName == "", 那么自动用 types.ExecName("game")
//如果设置了paraName , 那么强制用paraName
//也就是说，我们可以构造其他平行链的交易
func getRealExecName(paraName string) string {
	return types.ExecName(paraName + types.GameX)
}

func Init() {
	name = types.ExecName(types.GameX)
	// init executor type
	types.RegistorExecutor(types.ExecName(name), &TokenType{})

	// init log
	types.RegistorLog(types.TyLogCreateGame, &TokenTransferLog{})
	types.RegistorLog(types.TyLogCancleGame, &TokenDepositLog{})
	types.RegistorLog(types.TyLogMatchGame, &TokenExecTransferLog{})
	types.RegistorLog(types.TyLogCloseGame, &TokenExecWithdrawLog{})

	// init query rpc
	types.RegistorRpcType(FuncName_QueryGameListByIds, &GameGetList{})
	types.RegistorRpcType(FuncName_QueryGameById, &GameGetInfo{})
	types.RegistorRpcType(FuncName_QueryGameListByStatusAndAddr, &GameQueryList{})
}

// exec
type GameType struct {
	types.ExecTypeBase
}

func (game GameType) GetRealToAddr(tx *types.Transaction) string {
	if string(tx.Execer) == types.GameX {
		return tx.To
	}
	var action types.GameAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return tx.To
	}
	return tx.To
}

func (game GameType) ActionName(tx *types.Transaction) string {
	var action types.GameAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return "unknow-err"
	}

	if action.Ty == types.GameActionCreate && action.GetCreate() != nil {
		return Action_CreateGame
	} else if action.Ty == types.GameActionMatch && action.GetMatch() != nil {
		return Action_MatchGame
	} else if action.Ty == types.GameActionCancel && action.GetCancel() != nil {
		return Action_CancelGame
	} else if action.Ty == types.GameActionClose && action.GetClose() != nil {
		return Action_CloseGame
	}
	return "unknow"
}

// TODO 暂时不修改实现， 先完成结构的重构
func (game GameType) CreateTx(action string, message json.RawMessage) (*types.Transaction, error) {
	tlog.Debug("token.CreateTx", "action", action)
	var tx *types.Transaction
	if action == Action_CreateGame {
		var param GamePreCreateTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			tlog.Error("CreateTx", "Error", err)
			return nil, types.ErrInputPara
		}

		return CreateRawTokenPreCreateTx(&param)
	} else if action == "TokenFinish" {
		var param TokenFinishTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			tlog.Error("CreateTx", "Error", err)
			return nil, types.ErrInputPara
		}

		return CreateRawTokenFinishTx(&param)
	} else if action == "TokenRevoke" {
		var param TokenRevokeTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			tlog.Error("CreateTx", "Error", err)
			return nil, types.ErrInputPara
		}

		return CreateRawTokenRevokeTx(&param)
	} else {
		return nil, types.ErrNotSupport
	}

	return tx, nil
}

func CreateTokenTransfer(param *types.CreateTx) *types.Transaction {
	transfer := &types.TokenAction{}
	if !param.IsWithdraw {
		//如果在平行链上构造，或者传入的execName是paraExecName,则加入ToAddr
		if types.IsPara() || types.IsParaExecName(param.GetExecName()) {
			v := &types.TokenAction_Transfer{Transfer: &types.CoinsTransfer{
				Cointoken: param.GetTokenSymbol(), Amount: param.Amount, Note: param.GetNote(), To: param.GetTo()}}
			transfer.Value = v
			transfer.Ty = types.ActionTransfer
		} else {
			v := &types.TokenAction_Transfer{Transfer: &types.CoinsTransfer{
				Cointoken: param.GetTokenSymbol(), Amount: param.Amount, Note: param.GetNote()}}
			transfer.Value = v
			transfer.Ty = types.ActionTransfer
		}

	} else {
		v := &types.TokenAction_Withdraw{Withdraw: &types.CoinsWithdraw{
			Cointoken: param.GetTokenSymbol(), Amount: param.Amount, Note: param.GetNote()}}
		transfer.Value = v
		transfer.Ty = types.ActionWithdraw
	}
	//在平行链上，execName=token或者没加，默认构造平行链的交易
	if types.IsPara() && (param.GetExecName() == types.TokenX || param.GetExecName() == "") {
		return &types.Transaction{Execer: []byte(getRealExecName(types.GetParaName())), Payload: types.Encode(transfer), To: address.ExecAddress(getRealExecName(types.GetParaName()))}
	}
	//如果传入 execName 是平行链的execName的话，按照传入的execName构造交易
	if types.IsParaExecName(param.GetExecName()) {
		return &types.Transaction{Execer: []byte(param.GetExecName()), Payload: types.Encode(transfer), To: address.ExecAddress(param.GetExecName())}
	}
	//其他情况，默认构造主链交易
	return &types.Transaction{Execer: []byte(name), Payload: types.Encode(transfer), To: param.GetTo()}
}

func CreateRawTokenPreCreateTx(parm *TokenPreCreateTx) (*types.Transaction, error) {
	if parm == nil {
		tlog.Error("CreateRawTokenPreCreateTx", "parm", parm)
		return nil, types.ErrInvalidParam
	}
	v := &types.TokenPreCreate{
		Name:         parm.Name,
		Symbol:       parm.Symbol,
		Introduction: parm.Introduction,
		Total:        parm.Total,
		Price:        parm.Price,
		Owner:        parm.OwnerAddr,
	}
	precreate := &types.TokenAction{
		Ty:    types.TokenActionPreCreate,
		Value: &types.TokenAction_Tokenprecreate{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(getRealExecName(parm.ParaName)),
		Payload: types.Encode(precreate),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(getRealExecName(parm.ParaName)),
	}

	tx.SetRealFee(types.MinFee)

	return tx, nil
}

func CreateRawTokenFinishTx(parm *TokenFinishTx) (*types.Transaction, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}

	v := &types.TokenFinishCreate{Symbol: parm.Symbol, Owner: parm.OwnerAddr}
	finish := &types.TokenAction{
		Ty:    types.TokenActionFinishCreate,
		Value: &types.TokenAction_Tokenfinishcreate{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(getRealExecName(parm.ParaName)),
		Payload: types.Encode(finish),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(getRealExecName(parm.ParaName)),
	}

	tx.SetRealFee(types.MinFee)

	return tx, nil
}

func CreateRawTokenRevokeTx(parm *TokenRevokeTx) (*types.Transaction, error) {
	if parm == nil {
		return nil, types.ErrInvalidParam
	}
	v := &types.TokenRevokeCreate{Symbol: parm.Symbol, Owner: parm.OwnerAddr}
	revoke := &types.TokenAction{
		Ty:    types.TokenActionRevokeCreate,
		Value: &types.TokenAction_Tokenrevokecreate{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(getRealExecName(parm.ParaName)),
		Payload: types.Encode(revoke),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(getRealExecName(parm.ParaName)),
	}

	tx.SetRealFee(types.MinFee)

	return tx, nil
}

// log
type TokenTransferLog struct {
}

func (l TokenTransferLog) Name() string {
	return "LogTokenTransfer"
}

func (l TokenTransferLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptAccountTransfer
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type TokenDepositLog struct {
}

func (l TokenDepositLog) Name() string {
	return "LogTokenDeposit"
}

func (l TokenDepositLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptAccountTransfer
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type TokenExecTransferLog struct {
}

func (l TokenExecTransferLog) Name() string {
	return "LogTokenExecTransfer"
}

func (l TokenExecTransferLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptExecAccountTransfer
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type TokenExecWithdrawLog struct {
}

func (l TokenExecWithdrawLog) Name() string {
	return "LogTokenExecWithdraw"
}

func (l TokenExecWithdrawLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptExecAccountTransfer
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type TokenExecDepositLog struct {
}

func (l TokenExecDepositLog) Name() string {
	return "LogTokenExecDeposit"
}

func (l TokenExecDepositLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptExecAccountTransfer
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type TokenExecFrozenLog struct {
}

func (l TokenExecFrozenLog) Name() string {
	return "LogTokenExecFrozen"
}

func (l TokenExecFrozenLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptExecAccountTransfer
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type TokenExecActiveLog struct {
}

func (l TokenExecActiveLog) Name() string {
	return "LogTokenExecActive"
}

func (l TokenExecActiveLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptExecAccountTransfer
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type TokenGenesisTransferLog struct {
}

func (l TokenGenesisTransferLog) Name() string {
	return "LogTokenGenesisTransfer"
}

func (l TokenGenesisTransferLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptAccountTransfer
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type TokenGenesisDepositLog struct {
}

func (l TokenGenesisDepositLog) Name() string {
	return "LogTokenGenesisDeposit"
}

func (l TokenGenesisDepositLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptExecAccountTransfer
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type TokenPreCreateLog struct {
}

func (l TokenPreCreateLog) Name() string {
	return "LogPreCreateToken"
}

func (l TokenPreCreateLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptToken
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type TokenFinishCreateLog struct {
}

func (l TokenFinishCreateLog) Name() string {
	return "LogFinishCreateToken"
}

func (l TokenFinishCreateLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptToken
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type TokenRevokeCreateLog struct {
}

func (l TokenRevokeCreateLog) Name() string {
	return "LogRevokeCreateToken"
}

func (l TokenRevokeCreateLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptToken
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
	var req types.ReqTokens
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *GameGetList) Output(reply interface{}) (interface{}, error) {
	return reply, nil
}

type GameGetInfo struct {
}

func (g *GameGetInfo) Input(message json.RawMessage) ([]byte, error) {
	var req types.ReqString
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (g *GameGetInfo) Output(reply interface{}) (interface{}, error) {
	return reply, nil
}

type GameQueryList struct {
}

func (g *GameQueryList) Input(message json.RawMessage) ([]byte, error) {
	var req types.ReqAddrTokens
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (g *GameQueryList) Output(reply interface{}) (interface{}, error) {
	return reply, nil
}
