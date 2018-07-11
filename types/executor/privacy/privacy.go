package privacy

import (
	"encoding/json"

	//log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/types"
)

const name = "privacy"

//var tlog = log.New("module", name)

func Init() {
	// init executor type
	types.RegistorExecutor(name, &PrivacyType{})

	// init log
	types.RegistorLog(types.TyLogPrivacyFee, &PrivacyFeeLog{})
	types.RegistorLog(types.TyLogPrivacyInput, &PrivacyInputLog{})
	types.RegistorLog(types.TyLogPrivacyOutput, &PrivacyOutputLog{})

	// init query rpc
	types.RegistorRpcType("ShowAmountsOfUTXO", &PrivacyShowAmountsOfUTXO{})
	types.RegistorRpcType("ShowUTXOs4SpecifiedAmount", &PrivacyShowUTXOs4SpecifiedAmount{})
}

type PrivacyType struct {
	types.ExecTypeBase
}

func (coins PrivacyType) ActionName(tx *types.Transaction) string {
	var action types.PrivacyAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return "unknow-privacy-err"
	}
	return action.GetActionName()
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

func (t *PrivacyShowAmountsOfUTXO) Input(message json.RawMessage) ([]byte, error) {
	var req types.ReqPrivacyToken
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *PrivacyShowAmountsOfUTXO) Output(reply interface{}) (interface{}, error) {
	return reply, nil
}

type PrivacyShowUTXOs4SpecifiedAmount struct {
}

func (t *PrivacyShowUTXOs4SpecifiedAmount) Input(message json.RawMessage) ([]byte, error) {
	var req types.ReqPrivacyToken
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *PrivacyShowUTXOs4SpecifiedAmount) Output(reply interface{}) (interface{}, error) {
	return reply, nil
}
