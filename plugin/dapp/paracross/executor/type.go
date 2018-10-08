package executor

import (
	"encoding/json"

	log "github.com/inconshreveable/log15"
	pt "gitlab.33.cn/chain33/chain33/plugin/dapp/paracross/types"
	"gitlab.33.cn/chain33/chain33/types"
)

var nameX string

var glog = log.New("module", types.ParaX)

func InitType() {
	nameX = types.ExecName(types.ParaX)
	// init executor type
	types.RegistorExecutor(types.ParaX, NewType())

	// init log
	types.RegistorLog(pt.TyLogParacrossCommit, &ParacrossCommitLog{})
	types.RegistorLog(pt.TyLogParacrossCommitDone, &ParacrossDoneLog{})
	types.RegistorLog(pt.TyLogParacrossCommitRecord, &ParacrossCommitRecordLog{})
	types.RegistorLog(pt.TyLogParaAssetWithdraw, &ParacrossAssetWithdrawLog{})
	types.RegistorLog(pt.TyLogParaAssetTransfer, &ParacrossAssetTransferLog{})
	types.RegistorLog(pt.TyLogParaAssetDeposit, &ParacrossAssetDepositLog{})
	types.RegistorLog(pt.TyLogParacrossMiner, &ParacrossMinerLog{})

	// init query rpc
	types.RegisterRPCQueryHandle("ParacrossGetTitle", &ParacrossGetTitle{})
	types.RegisterRPCQueryHandle("ParacrossListTitles", &ParacrossListTitles{})
	types.RegisterRPCQueryHandle("ParacrossGetTitleHeight", &ParacrossGetTitleHeight{})
	types.RegisterRPCQueryHandle("ParacrossGetAssetTxResult", &ParacrossGetAssetTxResult{})
}

func GetExecName() string {
	return nameX
}

type ParacrossType struct {
	types.ExecTypeBase
}

func NewType() *ParacrossType {
	c := &ParacrossType{}
	c.SetChild(c)
	return c
}

func (b *ParacrossType) GetPayload() types.Message {
	return &pt.ParacrossAction{}
}

func (m ParacrossType) ActionName(tx *types.Transaction) string {
	var g pt.ParacrossAction
	err := types.Decode(tx.Payload, &g)
	if err != nil {
		return "unkown-paracross-action-err"
	}
	if g.Ty == pt.ParacrossActionCommit && g.GetCommit() != nil {
		return pt.ParacrossActionCommitStr
	} else if g.Ty == pt.ParacrossActionAssetTransfer && g.GetAssetTransfer() != nil {
		return pt.ParacrossActionTransferStr
	} else if g.Ty == pt.ParacrossActionAssetWithdraw && g.GetAssetWithdraw() != nil {
		return pt.ParacrossActionWithdrawStr
	} else if g.Ty == pt.ParacrossActionMiner && g.GetMiner() != nil {
		return types.MinerAction
	}
	return "unkown"
}

func (m ParacrossType) DecodePayload(tx *types.Transaction) (interface{}, error) {
	var action pt.ParacrossAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return nil, err
	}
	return &action, nil
}

func (m ParacrossType) Amount(tx *types.Transaction) (int64, error) {
	return 0, nil
}

func (m ParacrossType) CreateTx(action string, message json.RawMessage) (*types.Transaction, error) {
	if action == "ParacrossCommit" {
		var param pt.ParacrossCommitTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			glog.Error("CreateTx", "Error", err)
			return nil, types.ErrInputPara
		}

		return pt.CreateRawParacrossCommitTx(&param)
	} else if action == "ParacrossTransfer" || action == "ParacrossWithdraw" {
		var param types.CreateTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			glog.Error("CreateTx", "Error", err)
			return nil, types.ErrInputPara
		}
		return pt.CreateRawAssetTransferTx(&param)

	}

	return nil, types.ErrNotSupport
}

type ParacrossCommitLog struct {
}

func (l ParacrossCommitLog) Name() string {
	return "LogParacrossCommit"
}

func (l ParacrossCommitLog) Decode(msg []byte) (interface{}, error) {
	var logTmp pt.ReceiptParacrossCommit
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
	var logTmp pt.ReceiptParacrossDone
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
	var logTmp pt.ReceiptParacrossRecord
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
	var logTmp pt.ReceiptParacrossMiner
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
	var req types.ReqStr
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
	var req pt.ReqParacrossTitleHeight
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
