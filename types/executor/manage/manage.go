package manage

import (
	"encoding/json"

	//log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/types"
)

var name string

//var tlog = log.New("module", name)

func Init() {
	name = types.ExecName("manage")
	// init executor type
	types.RegistorExecutor(name, &ManageType{})

	// init log
	types.RegistorLog(types.TyLogModifyConfig, &ModifyConfigLog{})

	// init query rpc
	types.RegistorRpcType("GetConfigItem", &MagageGetConfigItem{})
}

type ManageType struct {
	types.ExecTypeBase
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

type MagageGetConfigItem struct {
}

func (t *MagageGetConfigItem) Input(message json.RawMessage) ([]byte, error) {
	var req types.ReqString
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *MagageGetConfigItem) Output(reply interface{}) (interface{}, error) {
	return reply, nil
}
