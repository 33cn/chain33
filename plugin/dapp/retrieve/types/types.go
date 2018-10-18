package types

import (
	"encoding/json"

	"gitlab.33.cn/chain33/chain33/types"
)

func init() {
	name := RetrieveX
	// init executor type
	types.RegistorExecutor(name, NewType())
}

type RetrieveType struct {
	types.ExecTypeBase
}

func NewType() *RetrieveType {
	c := &RetrieveType{}
	c.SetChild(c)
	return c
}

func (at *RetrieveType) GetPayload() types.Message {
	return &RetrieveAction{}
}

func (at *RetrieveType) GetName() string {
	return RetrieveX
}

func (at *RetrieveType) GetLogMap() map[int64]*types.LogInfo {
	return nil
}

func (at *RetrieveType) GetTypeMap() map[string]int32 {
	return actionName
}

func (r RetrieveType) ActionName(tx *types.Transaction) string {
	var action RetrieveAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return "unknown-err"
	}
	if action.Ty == RetrieveActionPrepare && action.GetPrepare() != nil {
		return "prepare"
	} else if action.Ty == RetrieveActionPerform && action.GetPerform() != nil {
		return "perform"
	} else if action.Ty == RetrieveActionBackup && action.GetBackup() != nil {
		return "backup"
	} else if action.Ty == RetrieveActionCancel && action.GetCancel() != nil {
		return "cancel"
	}
	return "unknown"
}

func (retrieve RetrieveType) DecodePayload(tx *types.Transaction) (interface{}, error) {
	var action RetrieveAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return nil, err
	}
	return &action, nil
}

func (r RetrieveType) Amount(tx *types.Transaction) (int64, error) {
	return 0, nil
}

func (retrieve RetrieveType) CreateTx(action string, message json.RawMessage) (*types.Transaction, error) {
	return nil, nil
}

type CoinsDepositLog struct {
}

func (l CoinsDepositLog) Name() string {
	return "LogDeposit"
}

func (l CoinsDepositLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptAccountTransfer
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type RetrieveGetInfo struct {
}

func (t *RetrieveGetInfo) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req ReqRetrieveInfo
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *RetrieveGetInfo) ProtoToJson(reply *types.Message) (interface{}, error) {
	return reply, nil
}
