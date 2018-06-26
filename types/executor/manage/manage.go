package manage

import (
	"encoding/json"

	//log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/types"
)

const name = "manage"

//var tlog = log.New("module", name)

func init() {
	// init executor type
	types.RegistorExecutor(name, &ManageType{})

	// init log
	types.RegistorLog(types.TyLogModifyConfig, &ModifyConfigLog{})

	// init query rpc
	//types.RegistorRpcType("q2", &CoinsGetTxsByAddr{})
}

type ManageType struct {
}

func (m ManageType) ActionName(tx *types.Transaction) string {
	return "config"
}

func (m ManageType) Amount(tx *types.Transaction) (int64, error) {
	return 0, nil
}

// TODO 暂时不修改实现， 先完成结构的重构
func (m ManageType) CreateTx(action string, message json.RawMessage) (*types.Transaction, error) {
	var tx *types.Transaction
	return tx, nil
}

type ModifyConfigLog struct {
}

func (l ModifyConfigLog) Name() string {
	return "LogModifyConfig"
}

func (l ModifyConfigLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptConfig
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
