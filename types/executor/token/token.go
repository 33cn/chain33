package token

import (
	"encoding/json"
	"math/rand"
	"time"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/types"
)

const name = "token"

var tlog = log.New("module", name)

func init() {
	// init executor type
	types.RegistorExecutor(name, &TokenType{})

	// init log
	types.RegistorLog(types.TyLogTokenTransfer, &TokenTransferLog{})
	types.RegistorLog(types.TyLogTokenDeposit, &TokenDepositLog{})
	types.RegistorLog(types.TyLogTokenExecTransfer, &TokenExecTransferLog{})
	types.RegistorLog(types.TyLogTokenExecWithdraw, &TokenExecWithdrawLog{})
	types.RegistorLog(types.TyLogTokenExecDeposit, &TokenExecDepositLog{})
	types.RegistorLog(types.TyLogTokenExecFrozen, &TokenExecFrozenLog{})
	types.RegistorLog(types.TyLogTokenExecActive, &TokenExecActiveLog{})

	types.RegistorLog(types.TyLogTokenGenesisTransfer, &TokenGenesisTransferLog{})
	types.RegistorLog(types.TyLogTokenGenesisDeposit, &TokenGenesisDepositLog{})

	types.RegistorLog(types.TyLogPreCreateToken, &TokenPreCreateLog{})
	types.RegistorLog(types.TyLogFinishCreateToken, &TokenFinishCreateLog{})
	types.RegistorLog(types.TyLogRevokeCreateToken, &TokenRevokeCreateLog{})

	// init query rpc
	types.RegistorRpcType("GetTokens", &TokenGetTokens{})
	types.RegistorRpcType("GetTokenInfo", &TokenGetTokenInfo{})
	types.RegistorRpcType("GetAddrReceiverforTokens", &TokenGetAddrReceiverforTokens{})
	types.RegistorRpcType("GetAccountTokenAssets", &TokenGetAccountTokenAssets{})
}

// exec
type TokenType struct {
	types.ExecTypeBase
}

func (token TokenType) GetRealToAddr(tx *types.Transaction) string {
	if string(tx.Execer) == name {
		return tx.To
	}
	var action types.TokenAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return tx.To
	}
	if action.Ty == types.ActionTransfer && action.GetTransfer() != nil {
		return action.GetTransfer().GetTo()
	}
	return tx.To
}

func (token TokenType) ActionName(tx *types.Transaction) string {
	var action types.TokenAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return "unknow-err"
	}

	if action.Ty == types.TokenActionPreCreate && action.GetTokenprecreate() != nil {
		return "preCreate"
	} else if action.Ty == types.TokenActionFinishCreate && action.GetTokenfinishcreate() != nil {
		return "finishCreate"
	} else if action.Ty == types.TokenActionRevokeCreate && action.GetTokenrevokecreate() != nil {
		return "revokeCreate"
	} else if action.Ty == types.ActionTransfer && action.GetTransfer() != nil {
		return "transferToken"
	} else if action.Ty == types.ActionWithdraw && action.GetWithdraw() != nil {
		return "withdrawToken"
	}
	return "unknow"
}

func (token TokenType) Amount(tx *types.Transaction) (int64, error) {
	//TODO: 补充和完善token和trade分支的amount的计算, added by hzj
	var action types.TokenAction
	err := types.Decode(tx.GetPayload(), &action)
	if err != nil {
		return 0, types.ErrDecode
	}

	if types.TokenActionPreCreate == action.Ty && action.GetTokenprecreate() != nil {
		precreate := action.GetTokenprecreate()
		return precreate.Price, nil
	} else if types.TokenActionFinishCreate == action.Ty && action.GetTokenfinishcreate() != nil {
		return 0, nil
	} else if types.TokenActionRevokeCreate == action.Ty && action.GetTokenrevokecreate() != nil {
		return 0, nil
	} else if types.ActionTransfer == action.Ty && action.GetTransfer() != nil {
		return 0, nil
	} else if types.ActionWithdraw == action.Ty && action.GetWithdraw() != nil {
		return 0, nil
	}
	return 0, nil
}

// TODO 暂时不修改实现， 先完成结构的重构
func (coins TokenType) CreateTx(action string, message json.RawMessage) (*types.Transaction, error) {
	tlog.Debug("token.CreateTx", "action", action)
	var tx *types.Transaction
	// transfer/withdraw 原来实现混在一起， 简单的从原来的代码移进来
	if action == "" {
		var param types.CreateTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			tlog.Error("CreateTx", "Error", err)
			return nil, types.ErrInputPara
		}

		if param.ExecName != "" && !types.IsAllowExecName(param.ExecName) {
			tlog.Error("CreateTx", "Error", types.ErrExecNameNotMatch)
			return nil, types.ErrExecNameNotMatch
		}

		//to地址要么是普通用户地址，要么就是执行器地址，不能为空
		if param.To == "" {
			return nil, types.ErrAddrNotExist
		}

		if param.Amount < 0 {
			return nil, types.ErrAmount
		}
		if param.IsToken {
			tx = CreateTokenTransfer(&param)
		} else {
			return nil, types.ErrNotSupport
		}

		tx.Fee, err = tx.GetRealFee(types.MinFee)
		if err != nil {
			return nil, err
		}

		random := rand.New(rand.NewSource(time.Now().UnixNano()))
		tx.Nonce = random.Int63()
	} else if action == "TokenPreCreate" {
		var param TokenPreCreateTx
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
		v := &types.TokenAction_Transfer{Transfer: &types.CoinsTransfer{
			Cointoken: param.GetTokenSymbol(), Amount: param.Amount, Note: param.GetNote()}}
		transfer.Value = v
		transfer.Ty = types.ActionTransfer
	} else {
		v := &types.TokenAction_Withdraw{Withdraw: &types.CoinsWithdraw{
			Cointoken: param.GetTokenSymbol(), Amount: param.Amount, Note: param.GetNote()}}
		transfer.Value = v
		transfer.Ty = types.ActionWithdraw
	}
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
		Execer:  []byte("token"),
		Payload: types.Encode(precreate),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress("token"),
	}

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
		Execer:  []byte("token"),
		Payload: types.Encode(finish),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress("token"),
	}

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
		Execer:  []byte("token"),
		Payload: types.Encode(revoke),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress("token"),
	}

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
type TokenGetTokens struct {
}

func (t *TokenGetTokens) Input(message json.RawMessage) ([]byte, error) {
	var req types.ReqTokens
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *TokenGetTokens) Output(reply interface{}) (interface{}, error) {
	return reply, nil
}

type TokenGetTokenInfo struct {
}

func (t *TokenGetTokenInfo) Input(message json.RawMessage) ([]byte, error) {
	var req types.ReqString
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *TokenGetTokenInfo) Output(reply interface{}) (interface{}, error) {
	return reply, nil
}

type TokenGetAddrReceiverforTokens struct {
}

func (t *TokenGetAddrReceiverforTokens) Input(message json.RawMessage) ([]byte, error) {
	var req types.ReqAddrTokens
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *TokenGetAddrReceiverforTokens) Output(reply interface{}) (interface{}, error) {
	return reply, nil
}

type TokenGetAccountTokenAssets struct {
}

func (t *TokenGetAccountTokenAssets) Input(message json.RawMessage) ([]byte, error) {
	var req types.ReqAccountTokenAssets
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *TokenGetAccountTokenAssets) Output(reply interface{}) (interface{}, error) {
	return reply, nil
}
