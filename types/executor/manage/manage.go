package manage

import (
	"encoding/json"

	//log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/types"
)

var nameX string

//var tlog = log.New("module", name)

func Init() {
	nameX = types.ExecName(types.ManageX)
	// init executor type
	types.RegistorExecutor(types.ManageX, NewType())

	// init log
	types.RegistorLog(types.TyLogModifyConfig, &ModifyConfigLog{})

	// init query rpc
	types.RegisterRPCQueryHandle("GetConfigItem", &MagageGetConfigItem{})
}

type ManageType struct {
	types.ExecTypeBase
}

func NewType() *ManageType {
	c := &ManageType{}
	c.SetChild(c)
	return c
}

func (at *ManageType) GetPayload() types.Message {
	return &types.ManageAction{}
}

func (m ManageType) ActionName(tx *types.Transaction) string {
	return "config"
}

func (manage ManageType) DecodePayload(tx *types.Transaction) (interface{}, error) {
	var action types.ManageAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return nil, err
	}
	return &action, nil
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

func (t *MagageGetConfigItem) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req types.ReqString
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *MagageGetConfigItem) ProtoToJson(reply *types.Message) (interface{}, error) {
	return reply, nil
}
