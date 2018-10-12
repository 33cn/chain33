package types

import (
	"encoding/json"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	nameX      string
	tlog       = log.New("module", types.TokenX)
	actionName = map[string]int32{
		"Transfer":          ActionTransfer,
		"Genesis":           ActionGenesis,
		"Withdraw":          ActionWithdraw,
		"Tokenprecreate":    TokenActionPreCreate,
		"Tokenfinishcreate": TokenActionFinishCreate,
		"Tokenrevokecreate": TokenActionRevokeCreate,
		"TransferToExec":    TokenActionTransferToExec,
	}
)

func init() {
	nameX = types.ExecName(types.TokenX)
	// init executor type
	types.RegistorExecutor(types.TokenX, NewType())

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

func NewType() *TokenType {
	c := &TokenType{}
	c.SetChild(c)
	return c
}

func (t *TokenType) GetPayload() types.Message {
	return &TokenAction{}
}

func (t *TokenType) GetTypeMap() map[string]int32 {
	return actionName
}

func (c *TokenType) RPC_Default_Process(action string, msg interface{}) (*types.Transaction, error) {
	var create *types.CreateTx
	if _, ok := msg.(*types.CreateTx); !ok {
		return nil, types.ErrInvalidParam
	}
	create = msg.(*types.CreateTx)
	if !create.IsToken {
		return nil, types.ErrNotSupport
	}
	tx, err := c.AssertCreate(create)
	if err != nil {
		return nil, err
	}
	//to地址的问题,如果是主链交易，to地址就是直接是设置to
	if !types.IsPara() {
		tx.To = create.To
	}
	return tx, err
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
	var logTmp ReceiptToken
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
	var logTmp ReceiptToken
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
	var logTmp ReceiptToken
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
	var req ReqTokens
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
	var req ReqAddressToken
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
	var req ReqAccountTokenAssets
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
	var req ReqTokenTx
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
