package evm

import (
	"encoding/json"
	"math/rand"
	"strings"
	"time"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/types"
)

var name string

var elog = log.New("module", name)

func Init() {
	// init executor type
	name = types.ExecName("evm")
	types.RegistorExecutor(name, &EvmType{})

	// init log
	types.RegistorLog(types.TyLogCallContract, &EvmCallContractLog{})
	types.RegistorLog(types.TyLogContractData, &EvmContractDataLog{})
	types.RegistorLog(types.TyLogContractState, &EvmContractStateLog{})
	types.RegistorLog(types.TyLogEVMStateChangeItem, &EvmStateChangeItemLog{})

	// init query rpc
	types.RegistorRpcType("CheckAddrExists", &EvmCheckAddrExists{})
	types.RegistorRpcType("EstimateGas", &EvmEstimateGas{})
	types.RegistorRpcType("EvmDebug", &EvmDebug{})
}

type EvmType struct {
	types.ExecTypeBase
}

func (evm EvmType) ActionName(tx *types.Transaction) string {
	// 这个需要通过合约交易目标地址来判断Action
	// 如果目标地址为空，或为evm的固定合约地址，则为创建合约，否则为调用合约
	if strings.EqualFold(tx.To, address.ExecAddress(types.ExecName(types.EvmX))) {
		return "createEvmContract"
	} else {
		return "callEvmContract"
	}
	return "unknow"
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
			return nil, types.ErrInputPara
		}
		return CreateRawEvmCreateCallTx(&param)
	} else {
		return nil, types.ErrNotSupport
	}

	return tx, nil
}

type EvmCallContractLog struct {
}

func (l EvmCallContractLog) Name() string {
	return "LogCallContract"
}

func (l EvmCallContractLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptEVMContract
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type EvmContractDataLog struct {
}

func (l EvmContractDataLog) Name() string {
	return "LogContractData"
}

func (l EvmContractDataLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.EVMContractData
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type EvmStateChangeItemLog struct {
}

func (l EvmStateChangeItemLog) Name() string {
	return "LogEVMStateChangeItem"
}

func (l EvmStateChangeItemLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.EVMStateChangeItem
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type EvmContractStateLog struct {
}

func (l EvmContractStateLog) Name() string {
	return "LogContractState"
}

func (l EvmContractStateLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.EVMContractState
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type EvmCheckAddrExists struct {
}

func (t *EvmCheckAddrExists) Input(message json.RawMessage) ([]byte, error) {
	var req types.CheckEVMAddrReq
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *EvmCheckAddrExists) Output(reply interface{}) (interface{}, error) {
	return reply, nil
}

type EvmEstimateGas struct {
}

func (t *EvmEstimateGas) Input(message json.RawMessage) ([]byte, error) {
	var req types.EstimateEVMGasReq
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *EvmEstimateGas) Output(reply interface{}) (interface{}, error) {
	return reply, nil
}

type EvmDebug struct {
}

func (t *EvmDebug) Input(message json.RawMessage) ([]byte, error) {
	var req types.EvmDebugReq
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *EvmDebug) Output(reply interface{}) (interface{}, error) {
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

	action := &types.EVMContractAction{
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
			Execer:  []byte(types.ExecName(types.EvmX)),
			Payload: types.Encode(action),
			//Fee:     parm.Fee,
			Nonce: rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
			To:    address.ExecAddress(types.ExecName(types.EvmX)),
		}
	} else {
		tx = &types.Transaction{
			Execer:  []byte(types.ExecName(parm.Name)),
			Payload: types.Encode(action),
			//Fee:     parm.Fee,
			Nonce: rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
			To:    address.ExecAddress(types.ExecName(parm.Name)),
		}
	}

	//according to cli/commands/evm.go/createEvmTx
	tx.Fee, _ = tx.GetRealFee(types.MinFee)
	if tx.Fee < parm.Fee {
		tx.Fee += parm.Fee
	}

	return tx, nil
}
