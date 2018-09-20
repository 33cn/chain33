package token

import (
	"encoding/json"
	"math/rand"
	"time"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/common"
)

var nameX string

var tlog = log.New("module", types.TokenX)

//getRealExecName
//如果paraName == "", 那么自动用 types.types.ExecName("token")
//如果设置了paraName , 那么强制用paraName
//也就是说，我们可以构造其他平行链的交易
func getRealExecName(paraName string) string {
	return types.ExecName(paraName + types.TokenX)
}

func Init() {
	nameX = types.ExecName(types.TokenX)
	// init executor type
	types.RegistorExecutor(types.TokenX, &TokenType{})

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
	types.RegisterRPCQueryHandle("GetTokens", &TokenGetTokens{})
	types.RegisterRPCQueryHandle("GetTokenInfo", &TokenGetTokenInfo{})
	types.RegisterRPCQueryHandle("GetAddrReceiverforTokens", &TokenGetAddrReceiverforTokens{})
	types.RegisterRPCQueryHandle("GetAccountTokenAssets", &TokenGetAccountTokenAssets{})
	types.RegisterRPCQueryHandle("GetTxByToken", &TokenGetTxByToken{})
}

// exec
type TokenType struct {
	types.ExecTypeBase
}

func (token TokenType) GetRealToAddr(tx *types.Transaction) string {
	if string(tx.Execer) == types.TokenX {
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
		return "unknown-err"
	}

	if action.Ty == types.TokenActionPreCreate && action.GetTokenprecreate() != nil {
		return "preCreate"
	} else if action.Ty == types.TokenActionFinishCreate && action.GetTokenfinishcreate() != nil {
		return "finishCreate"
	} else if action.Ty == types.TokenActionRevokeCreate && action.GetTokenrevokecreate() != nil {
		return "revokeCreate"
	} else if action.Ty == types.ActionTransfer && action.GetTransfer() != nil {
		return "transfer"
	} else if action.Ty == types.ActionWithdraw && action.GetWithdraw() != nil {
		return "withdraw"
	}
	return "unknown"
}

func (token TokenType) DecodePayload(tx *types.Transaction) (interface{}, error) {
	var action types.TokenAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return nil, err
	}
	return &action, nil
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

		if param.ExecName != "" && !types.IsAllowExecName([]byte(param.ExecName), []byte(param.ExecName)) {
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
		//如果在平行链上构造，或者传入的execName是paraExecName,则加入ToAddr
		if types.IsPara() || types.IsParaExecName(param.GetExecName()) {
			v := &types.TokenAction_Transfer{Transfer: &types.AssetsTransfer{
				Cointoken: param.GetTokenSymbol(), Amount: param.Amount, Note: param.GetNote(), To: param.GetTo()}}
			transfer.Value = v
			transfer.Ty = types.ActionTransfer
		} else {
			v := &types.TokenAction_Transfer{Transfer: &types.AssetsTransfer{
				Cointoken: param.GetTokenSymbol(), Amount: param.Amount, Note: param.GetNote()}}
			transfer.Value = v
			transfer.Ty = types.ActionTransfer
		}

	} else {
		v := &types.TokenAction_Withdraw{Withdraw: &types.AssetsWithdraw{
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
	return &types.Transaction{Execer: []byte(nameX), Payload: types.Encode(transfer), To: param.GetTo()}
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
type TokenGetTokens struct {
}

func (t *TokenGetTokens) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req types.ReqTokens
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *TokenGetTokens) ProtoToJson(reply *types.Message) (interface{}, error) {
	return reply, nil
}

type TokenGetTokenInfo struct {
}

func (t *TokenGetTokenInfo) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req types.ReqString
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *TokenGetTokenInfo) ProtoToJson(reply *types.Message) (interface{}, error) {
	return reply, nil
}

type TokenGetAddrReceiverforTokens struct {
}

func (t *TokenGetAddrReceiverforTokens) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req types.ReqAddrTokens
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *TokenGetAddrReceiverforTokens) ProtoToJson(reply *types.Message) (interface{}, error) {
	return reply, nil
}

type TokenGetAccountTokenAssets struct {
}

func (t *TokenGetAccountTokenAssets) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req types.ReqAccountTokenAssets
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *TokenGetAccountTokenAssets) ProtoToJson(reply *types.Message) (interface{}, error) {
	return reply, nil
}

type TokenGetTxByToken struct {
}

func (t *TokenGetTxByToken) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req types.ReqTokenTx
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *TokenGetTxByToken) ProtoToJson(reply *types.Message) (interface{}, error) {
	type ReplyTxInfo struct {
		Hash   string `json:"hash"`
		Height int64  `json:"height"`
		Index  int64  `json:"index"`
	}
	type ReplyTxInfos struct {
		TxInfos []*ReplyTxInfo `json:"txInfos"`
	}

	txInfos := (*reply).(*types.ReplyTxInfos)
	var txinfos ReplyTxInfos
	infos := txInfos.GetTxInfos()
	for _, info := range infos {
		txinfos.TxInfos = append(txinfos.TxInfos, &ReplyTxInfo{Hash: common.ToHex(info.GetHash()),
			Height: info.GetHeight(), Index: info.GetIndex()})
	}
	return &txinfos, nil
}
