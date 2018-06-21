package coins

import (
	"gitlab.33.cn/chain33/chain33/types"
	"encoding/json"
)

func init() {
	// init executor type
	types.RegistorExecutor("coins", &CoinsType{})

	// init log
	types.RegistorLog(types.TyLogDeposit, &CoinsDepositLog{})
	types.RegistorLog(types.TyLogTransfer, &CoinsTransferLog{})
	types.RegistorLog(types.TyLogGenesis, &CoinsGenesisLog{})

	// init query rpc
	types.RegistorRpcType("GetAddrReciver", &CoinsGetAddrReciver{})
	types.RegistorRpcType("GetTxsByAddr", &CoinsGetTxsByAddr{})
}

type CoinsType struct {
}

func (coins CoinsType) ActionName(tx *types.Transaction) string {
	var action types.CoinsAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return "unknow-err"
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
	return "unknow"
}

type CoinsDepositLog struct {
}

func (l CoinsDepositLog) Name() string {
	return "LogDeposit"
}

func (l CoinsDepositLog) Decode(msg []byte) (interface{}, error){
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

func (l CoinsGenesisLog) Decode(msg []byte) (interface{}, error){
	return nil, nil
}

type CoinsTransferLog struct {
}

func (l CoinsTransferLog) Name() string {
	return "LogGenesis"
}

func (l CoinsTransferLog) Decode(msg []byte) (interface{}, error){
	var logTmp types.ReceiptAccountTransfer
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, nil
}


type CoinsGetAddrReciver struct {
}

func (t *CoinsGetAddrReciver) Input(message json.RawMessage) ([]byte, error) {
	var req types.ReqAddr
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *CoinsGetAddrReciver) Output(reply interface{}) (interface{}, error) {
	return reply, nil
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


