package token

import (

	"gitlab.33.cn/chain33/chain33/types"
	"encoding/json"
	"time"
	log "github.com/inconshreveable/log15"
	"math/rand"
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

	// init query rpc
	types.RegistorRpcType("GetTokens", &TokenGetTokens{})
	types.RegistorRpcType("GetTokenInfo", &TokenGetTokenInfo{})
	types.RegistorRpcType("GetAddrReceiverforTokens", &TokenGetAddrReceiverforTokens{})
	types.RegistorRpcType("GetAccountTokenAssets", &TokenGetAccountTokenAssets{})
}


// exec
type TokenType struct {
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

// TODO 暂时不修改实现， 先完成结构的重构
// TODO token 还有其他的交易
func (coins TokenType) NewTx(action string, message json.RawMessage) (*types.Transaction, error) {
	var param types.CreateTx
	err := json.Unmarshal(message, &param)
	if err != nil {
		tlog.Error("NewTx", "Error", err)
		return nil, types.ErrInputPara
	}

	if param.ExecName != "" && !types.IsAllowExecName(param.ExecName) {
		tlog.Error("NewTx", "Error", types.ErrExecNameNotMatch)
		return nil, types.ErrExecNameNotMatch
	}

	//to地址要么是普通用户地址，要么就是执行器地址，不能为空
	if param.To == "" {
		return nil, types.ErrAddrNotExist
	}

	var tx *types.Transaction
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

// log
type TokenTransferLog struct {
}

func (l TokenTransferLog) Name() string {
	return "LogTokenTransfer"
}

func (l TokenTransferLog) Decode(msg []byte) (interface{}, error){
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

func (l TokenDepositLog) Decode(msg []byte) (interface{}, error){
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

func (l TokenExecTransferLog) Decode(msg []byte) (interface{}, error){
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

func (l TokenExecWithdrawLog) Decode(msg []byte) (interface{}, error){
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

func (l TokenExecDepositLog) Decode(msg []byte) (interface{}, error){
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

func (l TokenExecFrozenLog) Decode(msg []byte) (interface{}, error){
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
	return "LogTokenGenesisTransfer"
}

func (l TokenExecActiveLog) Decode(msg []byte) (interface{}, error){
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

func (l TokenGenesisTransferLog) Decode(msg []byte) (interface{}, error){
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

func (l TokenGenesisDepositLog) Decode(msg []byte) (interface{}, error){
	var logTmp types.ReceiptExecAccountTransfer
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





