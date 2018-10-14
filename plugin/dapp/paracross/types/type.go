package types

import (
	"encoding/json"
	"reflect"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/types"
)

var glog = log.New("module", types.ParaX)

func init() {
	// init executor type
	types.RegistorExecutor(types.ParaX, NewType())
}

func GetExecName() string {
	return types.ExecName(types.ParaX)
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

func (b *ParacrossType) GetPayload() types.Message {
	return &ParacrossAction{}
}

func (m ParacrossType) DecodePayload(tx *types.Transaction) (interface{}, error) {
	var action ParacrossAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return nil, err
	}
	return &action, nil
}

func (m ParacrossType) CreateTx(action string, message json.RawMessage) (*types.Transaction, error) {
	if action == "ParacrossCommit" {
		var param ParacrossCommitTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			glog.Error("CreateTx", "Error", err)
			return nil, types.ErrInputPara
		}

		return CreateRawParacrossCommitTx(&param)
	} else if action == "ParacrossTransfer" || action == "ParacrossWithdraw" {
		var param types.CreateTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			glog.Error("CreateTx", "Error", err)
			return nil, types.ErrInputPara
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
