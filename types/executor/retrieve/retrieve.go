package retrieve

import (
	"encoding/json"

	//log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/types"
)

const name = "retrieve"

//var tlog = log.New("module", name)

func Init() {
	// init executor type
	types.RegistorExecutor(name, &RetrieveType{})

	// init log
	//types.RegistorLog(types.TyLogDeposit, &CoinsDepositLog{})

	// init query rpc
	types.RegistorRpcType("GetRetrieveInfo", &RetrieveGetInfo{})
}

type RetrieveType struct {
	types.ExecTypeBase
}

func (r RetrieveType) ActionName(tx *types.Transaction) string {
	var action types.RetrieveAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return "unknow-err"
	}
	if action.Ty == types.RetrievePre && action.GetPreRet() != nil {
		return "prepare"
	} else if action.Ty == types.RetrievePerf && action.GetPerfRet() != nil {
		return "perform"
	} else if action.Ty == types.RetrieveBackup && action.GetBackup() != nil {
		return "backup"
	} else if action.Ty == types.RetrieveCancel && action.GetCancel() != nil {
		return "cancel"
	}
	return "unknow"
}

func (r RetrieveType) Amount(tx *types.Transaction) (int64, error) {
	return 0, nil
}

// TODO 暂时不修改实现， 先完成结构的重构
func (ticket RetrieveType) CreateTx(action string, message json.RawMessage) (*types.Transaction, error) {
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

type RetrieveGetInfo struct {
}

func (t *RetrieveGetInfo) Input(message json.RawMessage) ([]byte, error) {
	var req types.ReqRetrieveInfo
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *RetrieveGetInfo) Output(reply interface{}) (interface{}, error) {
	return reply, nil
}
