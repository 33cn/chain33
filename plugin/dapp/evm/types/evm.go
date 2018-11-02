package types

import (
	"encoding/json"
	"strings"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	elog = log.New("module", "exectype.evm")

	actionName = map[string]int32{
		"EvmCreate": EvmCreateAction,
		"EvmCall":   EvmCallAction,
	}
)

func init() {
	types.AllowUserExec = append(types.AllowUserExec, ExecerEvm)
	// init executor type
	types.RegistorExecutor(ExecutorName, NewType())
	types.RegisterDappFork(ExecutorName, "ForkEVMState", 650000)
	types.RegisterDappFork(ExecutorName, "ForkEVMKVHash", 1000000)
	types.RegisterDappFork(ExecutorName, "Enable", 500000)
}

type EvmType struct {
	types.ExecTypeBase
}

func NewType() *EvmType {
	c := &EvmType{}
	c.SetChild(c)
	return c
}

func (evm *EvmType) GetPayload() types.Message {
	return &EVMContractAction{}
}

func (evm EvmType) ActionName(tx *types.Transaction) string {
	// 这个需要通过合约交易目标地址来判断Action
	// 如果目标地址为空，或为evm的固定合约地址，则为创建合约，否则为调用合约
	if strings.EqualFold(tx.To, address.ExecAddress(types.ExecName(ExecutorName))) {
		return "createEvmContract"
	} else {
		return "callEvmContract"
	}
	return "unknown"
}

func (evm *EvmType) GetTypeMap() map[string]int32 {
	return actionName
}

func (evm EvmType) GetRealToAddr(tx *types.Transaction) string {
	if string(tx.Execer) == ExecutorName {
		return tx.To
	}
	var action EVMContractAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return tx.To
	}
	return tx.To
}

func (evm EvmType) Amount(tx *types.Transaction) (int64, error) {
	return 0, nil
}

func (evm EvmType) CreateTx(action string, message json.RawMessage) (*types.Transaction, error) {
	elog.Debug("evm.CreateTx", "action", action)
	var tx *types.Transaction
	if action == "CreateCall" {
		var param CreateCallTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			elog.Error("CreateTx", "Error", err)
			return nil, types.ErrInvalidParam
		}
		return CreateRawEvmCreateCallTx(&param)
	} else {
		return nil, types.ErrNotSupport
	}

	return tx, nil
}

func (evm *EvmType) GetLogMap() map[int64]*types.LogInfo {
	return logInfo
}

type EvmCheckAddrExists struct {
}

func (t *EvmCheckAddrExists) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req CheckEVMAddrReq
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *EvmCheckAddrExists) ProtoToJson(reply *types.Message) (interface{}, error) {
	return reply, nil
}

type EvmEstimateGas struct {
}

func (t *EvmEstimateGas) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req EstimateEVMGasReq
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *EvmEstimateGas) ProtoToJson(reply *types.Message) (interface{}, error) {
	return reply, nil
}

type EvmDebug struct {
}

func (t *EvmDebug) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req EvmDebugReq
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *EvmDebug) ProtoToJson(reply *types.Message) (interface{}, error) {
	return reply, nil
}

//different with other exector, the name is defined by parm
func CreateRawEvmCreateCallTx(parm *CreateCallTx) (*types.Transaction, error) {
	if parm == nil {
		elog.Error("CreateRawEvmCreateCallTx", "parm", parm)
		return nil, types.ErrInvalidParam
	}

	bCode, err := common.FromHex(parm.Code)
	if err != nil {
		elog.Error("CreateRawEvmCreateCallTx", "parm.Code", parm.Code)
		return nil, err
	}

	action := &EVMContractAction{
		Amount:   parm.Amount,
		Code:     bCode,
		GasLimit: parm.GasLimit,
		GasPrice: parm.GasPrice,
		Note:     parm.Note,
		Alias:    parm.Alias,
	}
	tx := &types.Transaction{}
	if parm.IsCreate {
		tx = &types.Transaction{
			Execer:  []byte(types.ExecName(ExecutorName)),
			Payload: types.Encode(action),
			To:      address.ExecAddress(types.ExecName(ExecutorName)),
		}
	} else {
		tx = &types.Transaction{
			Execer:  []byte(types.ExecName(parm.Name)),
			Payload: types.Encode(action),
			To:      address.ExecAddress(types.ExecName(parm.Name)),
		}
	}
	tx, err = types.FormatTx(string(tx.Execer), tx)
	if err != nil {
		return nil, err
	}
	if tx.Fee < parm.Fee {
		tx.Fee += parm.Fee
	}
	return tx, nil
}
