package relay

import (
	"encoding/json"

	//log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/types"
)

const name = "relay"

//var tlog = log.New("module", name)

func init() {
	// init executor type
	types.RegistorExecutor(name, &RelayType{})

	// init log
	types.RegistorLog(types.TyLogCallContract, &EvmCallContractLog{})
	types.RegistorLog(types.TyLogContractData, &EvmContractDataLog{})
	types.RegistorLog(types.TyLogContractState, &EvmContractStateLog{})

	// init query rpc
	types.RegistorRpcType("CheckAddrExists", &EvmCheckAddrExists{})
	types.RegistorRpcType("EstimateGas", &EvmEstimateGas{})
	types.RegistorRpcType("EvmDebug", &EvmDebug{})
}

type RelayType struct {
}

func (r RelayType) ActionName(tx *types.Transaction) string {
	var relay types.RelayAction
	err := types.Decode(tx.Payload, &relay)
	if err != nil {
		return "unkown-relay-action-err"
	}
	if relay.Ty == types.RelayActionCreate && relay.GetCreate() != nil {
		return "relayCreateTx"
	}
	if relay.Ty == types.RelayActionRevoke && relay.GetRevoke() != nil {
		return "relayRevokeTx"
	}
	if relay.Ty == types.RelayActionAccept && relay.GetAccept() != nil {
		return "relayAcceptTx"
	}
	if relay.Ty == types.RelayActionConfirmTx && relay.GetConfirmTx() != nil {
		return "relayConfirmTx"
	}
	if relay.Ty == types.RelayActionVerifyTx && relay.GetVerify() != nil {
		return "relayVerifyTx"
	}
	if relay.Ty == types.RelayActionRcvBTCHeaders && relay.GetBtcHeaders() != nil {
		return "relay-receive-btc-heads"
	}
	return "unknow"
}

func (r RelayType) Amount(tx *types.Transaction) (int64, error) {
	var relay types.RelayAction
	err := types.Decode(tx.GetPayload(), &relay)
	if err != nil {
		return 0, types.ErrDecode
	}
	if types.RelayActionCreate == relay.Ty && relay.GetCreate() != nil {
		return int64(relay.GetCreate().BtyAmount), nil
	}
	return 0, nil
}

// TODO 暂时不修改实现， 先完成结构的重构
func (evm RelayType) CreateTx(action string, message json.RawMessage) (*types.Transaction, error) {
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
