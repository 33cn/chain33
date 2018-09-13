package privacy

import (
	"encoding/json"

	//log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/types"
)

var nameX string

//var tlog = log.New("module", name)

func Init() {
	nameX = types.ExecName("privacy")
	// init executor type
	types.RegistorExecutor("privacy", &PrivacyType{})

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

func (coins PrivacyType) ActionName(tx *types.Transaction) string {
	var action types.PrivacyAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return "unknow-privacy-err"
	}
	return action.GetActionName()
}

func (privacy PrivacyType) DecodePayload(tx *types.Transaction) (interface{}, error) {
	fromAction := &types.PrivacyAction{}
	err := types.Decode(tx.Payload, fromAction)
	if err != nil {
		return nil, err
	}
	retAction := &types.PrivacyAction4Print{}
	retAction.Ty = fromAction.Ty
	if fromAction.GetPublic2Privacy() != nil {
		fromValue := fromAction.GetPublic2Privacy()
		value := &types.Public2Privacy4Print{}
		value.Tokenname = fromValue.Tokenname
		value.Amount = fromValue.Amount
		value.Note = fromValue.Note
		value.Output = convertToPrivacyOutput4Print(fromValue.Output)
		retAction.Value = &types.PrivacyAction4Print_Public2Privacy{Public2Privacy: value}
	} else if fromAction.GetPrivacy2Privacy() != nil {
		fromValue := fromAction.GetPrivacy2Privacy()
		value := &types.Privacy2Privacy4Print{}
		value.Tokenname = fromValue.Tokenname
		value.Amount = fromValue.Amount
		value.Note = fromValue.Note
		value.Input = convertToPrivacyInput4Print(fromValue.Input)
		value.Output = convertToPrivacyOutput4Print(fromValue.Output)
		retAction.Value = &types.PrivacyAction4Print_Privacy2Privacy{Privacy2Privacy: value}
	} else if fromAction.GetPrivacy2Public() != nil {
		fromValue := fromAction.GetPrivacy2Public()
		value := &types.Privacy2Public4Print{}
		value.Tokenname = fromValue.Tokenname
		value.Amount = fromValue.Amount
		value.Note = fromValue.Note
		value.Input = convertToPrivacyInput4Print(fromValue.Input)
		value.Output = convertToPrivacyOutput4Print(fromValue.Output)
		retAction.Value = &types.PrivacyAction4Print_Privacy2Public{Privacy2Public: value}
	}
	return retAction, nil
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

func convertToPrivacyInput4Print(privacyInput *types.PrivacyInput) *types.PrivacyInput4Print {
	input4print := &types.PrivacyInput4Print{}
	for _, fromKeyInput := range privacyInput.Keyinput {
		keyinput := &types.KeyInput4Print{
			Amount:   fromKeyInput.Amount,
			KeyImage: common.Bytes2Hex(fromKeyInput.KeyImage),
		}
		for _, fromUTXOGl := range fromKeyInput.UtxoGlobalIndex {
			utxogl := &types.UTXOGlobalIndex4Print{
				Txhash:   common.Bytes2Hex(fromUTXOGl.Txhash),
				Outindex: fromUTXOGl.Outindex,
			}
			keyinput.UtxoGlobalIndex = append(keyinput.UtxoGlobalIndex, utxogl)
		}

		input4print.Keyinput = append(input4print.Keyinput, keyinput)
	}
	return input4print
}

func convertToPrivacyOutput4Print(privacyOutput *types.PrivacyOutput) *types.PrivacyOutput4Print {
	output4print := &types.PrivacyOutput4Print{
		RpubKeytx: common.Bytes2Hex(privacyOutput.RpubKeytx),
	}
	for _, fromoutput := range privacyOutput.Keyoutput {
		output := &types.KeyOutput4Print{
			Amount:        fromoutput.Amount,
			Onetimepubkey: common.Bytes2Hex(fromoutput.Onetimepubkey),
		}
		output4print.Keyoutput = append(output4print.Keyoutput, output)
	}
	return output4print
}
