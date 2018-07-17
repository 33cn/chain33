package hashlock

import (
	"encoding/json"

	//log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/types"
)

var name string

//var tlog = log.New("module", name)

func Init() {
	name = types.ExecName("hashlock")
	// init executor type
	types.RegistorExecutor(name, &HashlockType{})

	// init log
	//types.RegistorLog(types.TyLogDeposit, &CoinsDepositLog{})

	// init query rpc
	//types.RegistorRpcType("q2", &CoinsGetTxsByAddr{})
}

type HashlockType struct {
	types.ExecTypeBase
}

func (hashlock HashlockType) ActionName(tx *types.Transaction) string {
	var action types.HashlockAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return "unknow-err"
	}
	if action.Ty == types.HashlockActionLock && action.GetHlock() != nil {
		return "lock"
	} else if action.Ty == types.HashlockActionUnlock && action.GetHunlock() != nil {
		return "unlock"
	} else if action.Ty == types.HashlockActionSend && action.GetHsend() != nil {
		return "send"
	}
	return "unknow"
}

func (t HashlockType) Amount(tx *types.Transaction) (int64, error) {
	return 0, nil
}

// TODO 暂时不修改实现， 先完成结构的重构
func (hashlock HashlockType) CreateTx(action string, message json.RawMessage) (*types.Transaction, error) {
	var tx *types.Transaction
	return tx, nil
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
