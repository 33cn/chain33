package evm

import (
	"encoding/json"
	"strings"

	//log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/types"
)

const name = "evm"

//var tlog = log.New("module", name)

func init() {
	// init executor type
	types.RegistorExecutor(name, &EvmType{})

	// init log
	types.RegistorLog(types.TyLogCallContract, &EvmCallContractLog{})
	types.RegistorLog(types.TyLogContractData, &EvmContractDataLog{})
	types.RegistorLog(types.TyLogContractState, &EvmContractStateLog{})

	// init query rpc
	//types.RegistorRpcType("q2", &CoinsGetTxsByAddr{})
}

type EvmType struct {
}

func (evm EvmType) ActionName(tx *types.Transaction) string {
	// 这个需要通过合约交易目标地址来判断Action
	// 如果目标地址为空，或为evm的固定合约地址，则为创建合约，否则为调用合约
	if strings.EqualFold(tx.To, "19tjS51kjwrCoSQS13U3owe7gYBLfSfoFm") {
		return "createEvmContract"
	} else {
		return "callEvmContract"
	}
	return "unknow"
}

func (evm EvmType) Amount(tx *types.Transaction) (int64, error) {
	return 0, nil
}

// TODO 暂时不修改实现， 先完成结构的重构
func (evm EvmType) NewTx(action string, message json.RawMessage) (*types.Transaction, error) {
	var tx *types.Transaction
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

type CoinsGetTxsByAddr struct {
}

func (t *CoinsGetTxsByAddr) Input(message json.RawMessage) ([]byte, error) {
	var req types.ReqAddr
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *CoinsGetTxsByAddr) Output(reply interface{}) (interface{}, error) {
	return reply, nil
}
