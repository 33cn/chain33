package types

import (
	"encoding/json"

	"gitlab.33.cn/chain33/chain33/types"
)

var nameX string

//var tlog = log.New("module", name)
var (
	actionName = map[string]int32{
		"Public2Privacy":  types.ActionPublic2Privacy,
		"Privacy2Privacy": types.ActionPrivacy2Privacy,
		"Privacy2Public":  types.ActionPrivacy2Public,
	}
)

func Init() {
	nameX = types.ExecName("privacy")
	// init executor type
	types.RegistorExecutor("privacy", NewType())

	// init log
	types.RegistorLog(types.TyLogPrivacyFee, &PrivacyFeeLog{})
	types.RegistorLog(types.TyLogPrivacyInput, &PrivacyInputLog{})
	types.RegistorLog(types.TyLogPrivacyOutput, &PrivacyOutputLog{})

	// init query rpc
	types.RegisterRPCQueryHandle("ShowAmountsOfUTXO", &PrivacyShowAmountsOfUTXO{})
	types.RegisterRPCQueryHandle("ShowUTXOs4SpecifiedAmount", &PrivacyShowUTXOs4SpecifiedAmount{})
}

type PrivacyType struct {
	types.ExecTypeBase
}

func NewType() *PrivacyType {
	c := &PrivacyType{}
	c.SetChild(c)
	return c
}

func (at *PrivacyType) GetPayload() types.Message {
	return &types.PrivacyAction{}
}

func (at *PrivacyType) GetName() string {
	return "privacy"
}

func (at *PrivacyType) GetLogMap() map[int64]*types.LogInfo {
	return nil
}

func (c *PrivacyType) GetTypeMap() map[string]int32 {
	return actionName
}

func (coins PrivacyType) ActionName(tx *types.Transaction) string {
	var action types.PrivacyAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return "unknow-privacy-err"
	}
	return action.GetActionName()
}

func (privacy PrivacyType) DecodePayload(tx *types.Transaction) (interface{}, error) {
	action := &types.PrivacyAction{}
	err := types.Decode(tx.Payload, action)
	if err != nil {
		return nil, err
	}
	return action, nil
}

func (t PrivacyType) Amount(tx *types.Transaction) (int64, error) {
	var action types.PrivacyAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return 0, types.ErrDecode
	}
	if action.Ty == types.ActionPublic2Privacy && action.GetPublic2Privacy() != nil {
		return action.GetPublic2Privacy().GetAmount(), nil
	} else if action.Ty == types.ActionPrivacy2Privacy && action.GetPrivacy2Privacy() != nil {
		return action.GetPrivacy2Privacy().GetAmount(), nil
	} else if action.Ty == types.ActionPrivacy2Public && action.GetPrivacy2Public() != nil {
		return action.GetPrivacy2Public().GetAmount(), nil
	}
	return 0, nil
}

// TODO 暂时不修改实现， 先完成结构的重构
func (t PrivacyType) CreateTx(action string, message json.RawMessage) (*types.Transaction, error) {
	var tx *types.Transaction
	return tx, nil
}

type PrivacyFeeLog struct {
}

func (l PrivacyFeeLog) Name() string {
	return "LogPrivacyFee"
}

func (l PrivacyFeeLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptExecAccountTransfer
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type PrivacyInputLog struct {
}

func (l PrivacyInputLog) Name() string {
	return "TyLogPrivacyInput"
}

func (l PrivacyInputLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.PrivacyInput
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type PrivacyOutputLog struct {
}

func (l PrivacyOutputLog) Name() string {
	return "LogPrivacyOutput"
}

func (l PrivacyOutputLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptPrivacyOutput
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, nil
}

// query
type PrivacyShowAmountsOfUTXO struct {
}

func (t *PrivacyShowAmountsOfUTXO) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req types.ReqPrivacyToken
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *PrivacyShowAmountsOfUTXO) ProtoToJson(reply *types.Message) (interface{}, error) {
	return reply, nil
}

type PrivacyShowUTXOs4SpecifiedAmount struct {
}

func (t *PrivacyShowUTXOs4SpecifiedAmount) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req types.ReqPrivacyToken
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *PrivacyShowUTXOs4SpecifiedAmount) ProtoToJson(reply *types.Message) (interface{}, error) {
	return reply, nil
}
