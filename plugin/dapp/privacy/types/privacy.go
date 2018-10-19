package types

import (
	"encoding/json"
	"reflect"

	"gitlab.33.cn/chain33/chain33/types"
)

var PrivacyX = types.PrivacyX

const (
	InvalidAction = 0
	//action type for privacy
	ActionPublic2Privacy = iota + 100
	ActionPrivacy2Privacy
	ActionPrivacy2Public

	// log for privacy
	TyLogPrivacyFee    = 500
	TyLogPrivacyInput  = 501
	TyLogPrivacyOutput = 502
)

// RescanUtxoFlag
const (
	UtxoFlagNoScan  int32 = 0
	UtxoFlagScaning int32 = 1
	UtxoFlagScanEnd int32 = 2
)

var RescanFlagMapint2string = map[int32]string{
	UtxoFlagNoScan:  "UtxoFlagNoScan",
	UtxoFlagScaning: "UtxoFlagScaning",
	UtxoFlagScanEnd: "UtxoFlagScanEnd",
}

func init() {
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
	return &PrivacyAction{}
}

func (at *PrivacyType) GetName() string {
	return PrivacyX
}

func (at *PrivacyType) GetLogMap() map[int64]*types.LogInfo {
	return map[int64]*types.LogInfo{
		TyLogPrivacyFee:    {reflect.TypeOf(types.ReceiptExecAccountTransfer{}), "LogPrivacyFee"},
		TyLogPrivacyInput:  {reflect.TypeOf(PrivacyInput{}), "LogPrivacyInput"},
		TyLogPrivacyOutput: {reflect.TypeOf(PrivacyOutput{}), "LogPrivacyOutput"},
	}
}

func (c *PrivacyType) GetTypeMap() map[string]int32 {
	return map[string]int32{
		"Public2Privacy":  ActionPublic2Privacy,
		"Privacy2Privacy": ActionPrivacy2Privacy,
		"Privacy2Public":  ActionPrivacy2Public,
	}
}

func (coins PrivacyType) ActionName(tx *types.Transaction) string {
	var action PrivacyAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return "unknow-privacy-err"
	}
	return action.GetActionName()
}

func (privacy PrivacyType) DecodePayload(tx *types.Transaction) (interface{}, error) {
	action := &PrivacyAction{}
	err := types.Decode(tx.Payload, action)
	if err != nil {
		return nil, err
	}
	return action, nil
}

func (t PrivacyType) Amount(tx *types.Transaction) (int64, error) {
	var action PrivacyAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return 0, types.ErrDecode
	}
	if action.Ty == ActionPublic2Privacy && action.GetPublic2Privacy() != nil {
		return action.GetPublic2Privacy().GetAmount(), nil
	} else if action.Ty == ActionPrivacy2Privacy && action.GetPrivacy2Privacy() != nil {
		return action.GetPrivacy2Privacy().GetAmount(), nil
	} else if action.Ty == ActionPrivacy2Public && action.GetPrivacy2Public() != nil {
		return action.GetPrivacy2Public().GetAmount(), nil
	}
	return 0, nil
}

// TODO 暂时不修改实现， 先完成结构的重构
func (t PrivacyType) CreateTx(action string, message json.RawMessage) (*types.Transaction, error) {
	var tx *types.Transaction
	return tx, nil
}

func (action *PrivacyAction) GetInput() *PrivacyInput {
	if action.GetTy() == ActionPrivacy2Privacy && action.GetPrivacy2Privacy() != nil {
		return action.GetPrivacy2Privacy().GetInput()

	} else if action.GetTy() == ActionPrivacy2Public && action.GetPrivacy2Public() != nil {
		return action.GetPrivacy2Public().GetInput()
	}
	return nil
}

func (action *PrivacyAction) GetOutput() *PrivacyOutput {
	if action.GetTy() == ActionPublic2Privacy && action.GetPublic2Privacy() != nil {
		return action.GetPublic2Privacy().GetOutput()
	} else if action.GetTy() == ActionPrivacy2Privacy && action.GetPrivacy2Privacy() != nil {
		return action.GetPrivacy2Privacy().GetOutput()
	} else if action.GetTy() == ActionPrivacy2Public && action.GetPrivacy2Public() != nil {
		return action.GetPrivacy2Public().GetOutput()
	}
	return nil
}

func (action *PrivacyAction) GetActionName() string {
	if action.Ty == ActionPrivacy2Privacy && action.GetPrivacy2Privacy() != nil {
		return "Privacy2Privacy"
	} else if action.Ty == ActionPublic2Privacy && action.GetPublic2Privacy() != nil {
		return "Public2Privacy"
	} else if action.Ty == ActionPrivacy2Public && action.GetPrivacy2Public() != nil {
		return "Privacy2Public"
	}
	return "unknow-privacy"
}

func (action *PrivacyAction) GetTokenName() string {
	if action.GetTy() == ActionPublic2Privacy && action.GetPublic2Privacy() != nil {
		return action.GetPublic2Privacy().GetTokenname()
	} else if action.GetTy() == ActionPrivacy2Privacy && action.GetPrivacy2Privacy() != nil {
		return action.GetPrivacy2Privacy().GetTokenname()
	} else if action.GetTy() == ActionPrivacy2Public && action.GetPrivacy2Public() != nil {
		return action.GetPrivacy2Public().GetTokenname()
	}
	return ""
}
