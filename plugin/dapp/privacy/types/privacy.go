package types

import (
	"encoding/json"
	"reflect"

	"gitlab.33.cn/chain33/chain33/types"
)

var nameX string

//var tlog = log.New("module", name)

// log for privacy
const (

//TyLogPrivacyFee    = 500
//TyLogPrivacyInput  = 501
//TyLogPrivacyOutput = 502

)

var (
	actionName = map[string]int32{
		"Public2Privacy":  types.ActionPublic2Privacy,
		"Privacy2Privacy": types.ActionPrivacy2Privacy,
		"Privacy2Public":  types.ActionPrivacy2Public,
	}

	logInfo = map[int64]*types.LogInfo{
		types.TyLogPrivacyFee:    {reflect.TypeOf(types.ReceiptExecAccountTransfer{}), "LogPrivacyFee"},
		types.TyLogPrivacyInput:  {reflect.TypeOf(types.PrivacyInput{}), "LogPrivacyInput"},
		types.TyLogPrivacyOutput: {reflect.TypeOf(types.PrivacyOutput{}), "LogPrivacyOutput"},
	}
)

func init() {
	nameX = types.ExecName("privacy")
	// init executor type
	types.RegistorExecutor(types.PrivacyX, NewType())
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
	return logInfo
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
