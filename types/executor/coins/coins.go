package coins

import (
	"encoding/json"
	"math/rand"
	"time"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/types"
)

var name string
var tlog = log.New("module", name)

func Init() {
	name = types.ExecName("coins")
	// init executor type
	types.RegistorExecutor(name, &CoinsType{})

	// init log
	types.RegistorLog(types.TyLogDeposit, &CoinsDepositLog{})
	types.RegistorLog(types.TyLogTransfer, &CoinsTransferLog{})
	types.RegistorLog(types.TyLogGenesis, &CoinsGenesisLog{})

	types.RegistorLog(types.TyLogExecTransfer, &CoinsExecTransferLog{})
	types.RegistorLog(types.TyLogExecWithdraw, &CoinsExecWithdrawLog{})
	types.RegistorLog(types.TyLogExecDeposit, &CoinsExecDepositLog{})
	types.RegistorLog(types.TyLogExecFrozen, &CoinsExecFrozenLog{})
	types.RegistorLog(types.TyLogExecActive, &CoinsExecActiveLog{})

	types.RegistorLog(types.TyLogGenesisTransfer, &CoinsGenesisTransferLog{})
	types.RegistorLog(types.TyLogGenesisDeposit, &CoinsGenesisDepositLog{})

	// init query rpc
	types.RegistorRpcType("GetAddrReciver", &CoinsGetAddrReceiver{})
	types.RegistorRpcType("GetAddrReceiver", &CoinsGetAddrReceiver{})
	types.RegistorRpcType("GetTxsByAddr", &CoinsGetTxsByAddr{})
}

type CoinsType struct {
	types.ExecTypeBase
}

func (coins CoinsType) GetRealToAddr(tx *types.Transaction) string {
	if string(tx.Execer) == "coins" {
		return tx.To
	}
	var action types.CoinsAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return tx.To
	}
	if action.Ty == types.CoinsActionTransfer && action.GetTransfer() != nil {
		return action.GetTransfer().GetTo()
	} else if action.Ty == types.CoinsActionTransferToExec && action.GetTransferToExec() != nil {
		return action.GetTransferToExec().GetTo()
	} else if action.Ty == types.CoinsActionWithdraw && action.GetWithdraw() != nil {
		return action.GetWithdraw().GetTo()
	}
	return tx.To
}

func (coins CoinsType) ActionName(tx *types.Transaction) string {
	var action types.CoinsAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return "unknown-err"
	}
	if action.Ty == types.CoinsActionTransfer && action.GetTransfer() != nil {
		return "transfer"
	} else if action.Ty == types.CoinsActionWithdraw && action.GetWithdraw() != nil {
		return "withdraw"
	} else if action.Ty == types.CoinsActionGenesis && action.GetGenesis() != nil {
		return "genesis"
	} else if action.Ty == types.CoinsActionTransferToExec && action.GetTransferToExec() != nil {
		return "sendToExec"
	}
	return "unknown"
}

func (coins CoinsType) DecodePayload(tx *types.Transaction) (interface{}, error) {
	var action types.CoinsAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return nil, err
	}
	return &action, nil
}

func (t CoinsType) Amount(tx *types.Transaction) (int64, error) {
	var action types.CoinsAction
	err := types.Decode(tx.GetPayload(), &action)
	if err != nil {
		return 0, types.ErrDecode
	}
	if action.Ty == types.CoinsActionTransfer && action.GetTransfer() != nil {
		transfer := action.GetTransfer()
		return transfer.Amount, nil
	} else if action.Ty == types.CoinsActionGenesis && action.GetGenesis() != nil {
		gen := action.GetGenesis()
		return gen.Amount, nil
	} else if action.Ty == types.CoinsActionWithdraw && action.GetWithdraw() != nil {
		transfer := action.GetWithdraw()
		return transfer.Amount, nil
	} else if action.Ty == types.CoinsActionTransferToExec && action.GetTransferToExec() != nil {
		transfer := action.GetTransferToExec()
		return transfer.Amount, nil
	}
	return 0, nil
}

// TODO 暂时不修改实现， 先完成结构的重构
func (coins CoinsType) CreateTx(action string, message json.RawMessage) (*types.Transaction, error) {
	var param types.CreateTx
	err := json.Unmarshal(message, &param)
	if err != nil {
		tlog.Error("CreateTx", "Error", err)
		return nil, types.ErrInputPara
	}
	if param.ExecName != "" && !types.IsAllowExecName(param.ExecName) {
		tlog.Error("CreateTx", "Error", types.ErrExecNameNotMatch)
		return nil, types.ErrExecNameNotMatch
	}

	//to地址要么是普通用户地址，要么就是执行器地址，不能为空
	if param.To == "" {
		return nil, types.ErrAddrNotExist
	}

	var tx *types.Transaction
	if param.Amount < 0 {
		return nil, types.ErrAmount
	}
	if param.IsToken {
		return nil, types.ErrNotSupport
	} else {
		tx = CreateCoinsTransfer(&param)
	}

	tx.Fee, err = tx.GetRealFee(types.MinFee)
	if err != nil {
		return nil, err
	}

	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	tx.Nonce = random.Int63()

	return tx, nil
}

func CreateCoinsTransfer(param *types.CreateTx) *types.Transaction {
	transfer := &types.CoinsAction{}
	to := ""
	if types.IsPara() {
		to = param.GetTo()
	}
	if !param.IsWithdraw {
		if param.ExecName != "" {
			v := &types.CoinsAction_TransferToExec{TransferToExec: &types.CoinsTransferToExec{
				Amount: param.Amount, Note: param.GetNote(), ExecName: param.GetExecName(), To: to}}
			transfer.Value = v
			transfer.Ty = types.CoinsActionTransferToExec
		} else {
			v := &types.CoinsAction_Transfer{Transfer: &types.CoinsTransfer{
				Amount: param.Amount, Note: param.GetNote(), To: to}}
			transfer.Value = v
			transfer.Ty = types.CoinsActionTransfer
		}
	} else {
		v := &types.CoinsAction_Withdraw{Withdraw: &types.CoinsWithdraw{
			Amount: param.Amount, Note: param.GetNote(), ExecName: param.GetExecName(), To: to}}
		transfer.Value = v
		transfer.Ty = types.CoinsActionWithdraw
	}
	if types.IsPara() {
		return &types.Transaction{Execer: []byte(name), Payload: types.Encode(transfer), To: address.ExecAddress(name)}
	}
	return &types.Transaction{Execer: []byte(name), Payload: types.Encode(transfer), To: param.GetTo()}
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

type CoinsGenesisLog struct {
}

func (l CoinsGenesisLog) Name() string {
	return "LogGenesis"
}

func (l CoinsGenesisLog) Decode(msg []byte) (interface{}, error) {
	return nil, nil
}

type CoinsTransferLog struct {
}

func (l CoinsTransferLog) Name() string {
	return "LogTransfer"
}

func (l CoinsTransferLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptAccountTransfer
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, nil
}

type CoinsExecTransferLog struct {
}

func (l CoinsExecTransferLog) Name() string {
	return "LogExecTransfer"
}

func (l CoinsExecTransferLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptExecAccountTransfer
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, nil
}

type CoinsExecWithdrawLog struct {
}

func (l CoinsExecWithdrawLog) Name() string {
	return "LogExecWithdraw"
}

func (l CoinsExecWithdrawLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptExecAccountTransfer
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, nil
}

type CoinsExecDepositLog struct {
}

func (l CoinsExecDepositLog) Name() string {
	return "LogExecDeposit"
}

func (l CoinsExecDepositLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptExecAccountTransfer
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, nil
}

type CoinsExecFrozenLog struct {
}

func (l CoinsExecFrozenLog) Name() string {
	return "LogExecFrozen"
}

func (l CoinsExecFrozenLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptExecAccountTransfer
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, nil
}

type CoinsExecActiveLog struct {
}

func (l CoinsExecActiveLog) Name() string {
	return "LogExecActive"
}

func (l CoinsExecActiveLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptExecAccountTransfer
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, nil
}

type CoinsGenesisTransferLog struct {
}

func (l CoinsGenesisTransferLog) Name() string {
	return "LogGenesisTransfer"
}

func (l CoinsGenesisTransferLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptAccountTransfer
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, nil
}

type CoinsGenesisDepositLog struct {
}

func (l CoinsGenesisDepositLog) Name() string {
	return "LogGenesisDeposit"
}

func (l CoinsGenesisDepositLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptExecAccountTransfer
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, nil
}

// query
type CoinsGetAddrReceiver struct {
}

func (t *CoinsGetAddrReceiver) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req types.ReqAddr
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *CoinsGetAddrReceiver) ProtoToJson(reply *types.Message) (interface{}, error) {
	return reply, nil
}

type CoinsGetTxsByAddr struct {
}

func (t *CoinsGetTxsByAddr) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req types.ReqAddr
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *CoinsGetTxsByAddr) ProtoToJson(reply *types.Message) (interface{}, error) {
	return reply, nil
}
