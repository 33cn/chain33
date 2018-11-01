package types

import (
	"encoding/json"
	"reflect"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	ParaX = "paracross"
	glog  = log.New("module", ParaX)

	/*
		logInfo = map[int64]*types.LogInfo{
			TyLogParacrossCommit:  {reflect.TypeOf(ReceiptParacrossCommit{}), "LogParacrossCommit"},
			TyLogParacrossCommitDone:  {reflect.TypeOf(ReceiptParacrossDone{}), "LogParacrossDone"},
			TyLogParacrossCommitRecord: {reflect.TypeOf(ReceiptParacrossRecord{}), "LogParacrossCommitRecord"},
			TyLogParaAssetTransfer: {reflect.TypeOf(types.ReceiptAccountTransfer{}), "LogParacrossAssetTransfer"},
			TyLogParaAssetWithdraw:   {reflect.TypeOf(types.ReceiptAccountTransfer{}), "LogParacrossAssetWithdraw"},
			TyLogParacrossMiner:  {reflect.TypeOf(ReceiptParacrossMiner{}), "LogParacrossMiner"},
			TyLogParaAssetDeposit: {reflect.TypeOf(types.ReceiptAccountTransfer{}), "LogParacrossAssetDeposit"},
		}*/

	// init query rpc
	/* TODO-TODO
	types.RegisterRPCQueryHandle("ParacrossGetTitle", &ParacrossGetTitle{})
	types.RegisterRPCQueryHandle("ParacrossListTitles", &ParacrossListTitles{})
	types.RegisterRPCQueryHandle("ParacrossGetTitleHeight", &ParacrossGetTitleHeight{})
	types.RegisterRPCQueryHandle("ParacrossGetAssetTxResult", &ParacrossGetAssetTxResult{})
	*/
)

func init() {
	// init executor type
	types.AllowUserExec = append(types.AllowUserExec, []byte(ParaX))
	types.RegistorExecutor(ParaX, NewType())
	types.RegisterDappFork(ParaX, "Enable", 0)
}

func GetExecName() string {
	return types.ExecName(ParaX)
}

type ParacrossType struct {
	types.ExecTypeBase
}

func NewType() *ParacrossType {
	c := &ParacrossType{}
	c.SetChild(c)
	return c
}

func (at *ParacrossType) GetLogMap() map[int64]*types.LogInfo {
	return map[int64]*types.LogInfo{
		TyLogParacrossCommit:       {reflect.TypeOf(ReceiptParacrossCommit{}), "LogParacrossCommit"},
		TyLogParacrossCommitDone:   {reflect.TypeOf(ReceiptParacrossDone{}), "LogParacrossCommitDone"},
		TyLogParacrossCommitRecord: {reflect.TypeOf(ReceiptParacrossRecord{}), "LogParacrossCommitRecord"},
		TyLogParaAssetWithdraw:     {reflect.TypeOf(types.ReceiptAccountTransfer{}), "LogParaAssetWithdraw"},
		TyLogParaAssetTransfer:     {reflect.TypeOf(types.ReceiptAccountTransfer{}), "LogParaAssetTransfer"},
		TyLogParaAssetDeposit:      {reflect.TypeOf(types.ReceiptAccountTransfer{}), "LogParaAssetDeposit"},
		TyLogParacrossMiner:        {reflect.TypeOf(ReceiptParacrossMiner{}), "LogParacrossMiner"},
	}
}

func (t *ParacrossType) GetTypeMap() map[string]int32 {
	return map[string]int32{
		"Commit":         ParacrossActionCommit,
		"Miner":          ParacrossActionMiner,
		"AssetTransfer":  ParacrossActionAssetTransfer,
		"AssetWithdraw":  ParacrossActionAssetWithdraw,
		"Transfer":       ParacrossActionTransfer,
		"Withdraw":       ParacrossActionWithdraw,
		"TransferToExec": ParacrossActionTransferToExec,
	}
}

func (b *ParacrossType) GetPayload() types.Message {
	return &ParacrossAction{}
}

func (m ParacrossType) CreateTx(action string, message json.RawMessage) (*types.Transaction, error) {
	if action == "ParacrossCommit" {
		var param ParacrossCommitTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			glog.Error("CreateTx", "Error", err)
			return nil, types.ErrInvalidParam
		}

		return CreateRawParacrossCommitTx(&param)
	} else if action == "ParacrossAssetTransfer" || action == "ParacrossAssetWithdraw" {
		var param types.CreateTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			glog.Error("CreateTx", "Error", err)
			return nil, types.ErrInvalidParam
		}
		return CreateRawAssetTransferTx(&param)

	} else if action == "ParacrossTransfer" || action == "ParacrossWithdraw" || action == "ParacrossTransferToExec" {
		var param types.CreateTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			glog.Error("CreateTx", "Error", err)
			return nil, types.ErrInvalidParam
		}
		return CreateRawTransferTx(&param)

	}

	return nil, types.ErrNotSupport
}

type ParacrossCommitLog struct {
}

func (l ParacrossCommitLog) Name() string {
	return "LogParacrossCommit"
}

func (l ParacrossCommitLog) Decode(msg []byte) (interface{}, error) {
	var logTmp ReceiptParacrossCommit
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type ParacrossDoneLog struct {
}

func (l ParacrossDoneLog) Name() string {
	return "LogParacrossDone"
}

func (l ParacrossDoneLog) Decode(msg []byte) (interface{}, error) {
	var logTmp ReceiptParacrossDone
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type ParacrossCommitRecordLog struct {
}

func (l ParacrossCommitRecordLog) Name() string {
	return "LogParacrossCommitRecord"
}

func (l ParacrossCommitRecordLog) Decode(msg []byte) (interface{}, error) {
	var logTmp ReceiptParacrossRecord
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type ParacrossAssetWithdrawLog struct {
}

func (l ParacrossAssetWithdrawLog) Name() string {
	return "LogParacrossAssetWithdraw"
}

func (l ParacrossAssetWithdrawLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptAccountTransfer
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type ParacrossAssetTransferLog struct {
}

func (l ParacrossAssetTransferLog) Name() string {
	return "LogParacrossAssetTransfer"
}

func (l ParacrossAssetTransferLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptAccountTransfer
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type ParacrossMinerLog struct {
}

func (l ParacrossMinerLog) Name() string {
	return "LogParaMiner"
}

func (l ParacrossMinerLog) Decode(msg []byte) (interface{}, error) {
	var logTmp ReceiptParacrossMiner
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type ParacrossAssetDepositLog struct {
}

func (l ParacrossAssetDepositLog) Name() string {
	return "LogParacrossAssetDeposit"
}

func (l ParacrossAssetDepositLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptAccountTransfer
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, nil
}

type ParacrossGetTitle struct {
}

func (t *ParacrossGetTitle) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req types.ReqString
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *ParacrossGetTitle) ProtoToJson(reply *types.Message) (interface{}, error) {
	return reply, nil
}

type ParacrossListTitles struct {
}

func (t *ParacrossListTitles) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req types.ReqNil
	return types.Encode(&req), nil
}

func (t *ParacrossListTitles) ProtoToJson(reply *types.Message) (interface{}, error) {
	return reply, nil
}

type ParacrossGetTitleHeight struct {
}

func (t *ParacrossGetTitleHeight) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req ReqParacrossTitleHeight
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *ParacrossGetTitleHeight) ProtoToJson(reply *types.Message) (interface{}, error) {
	return reply, nil
}

type ParacrossGetAssetTxResult struct {
}

func (t *ParacrossGetAssetTxResult) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req types.ReqHash
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *ParacrossGetAssetTxResult) ProtoToJson(reply *types.Message) (interface{}, error) {
	return reply, nil
}
