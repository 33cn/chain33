package types

import (
	"encoding/json"
	"reflect"

	"gitlab.33.cn/chain33/chain33/common/address"

	//log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	ManageX    = "manage"
	actionName = map[string]int32{
		"Modify": ManageActionModifyConfig,
	}
	logmap = map[int64]*types.LogInfo{
		types.TyLogModifyConfig: {reflect.TypeOf(ModifyConfigLog{}), "LogModifyConfig"},
	}
)

//var tlog = log.New("module", name)

func init() {
	types.AllowUserExec = append(types.AllowUserExec, []byte(ManageX))
	types.RegistorExecutor(ManageX, NewType())
	types.RegisterDappFork(ManageX, "Enable", 120000)
	types.RegisterDappFork(ManageX, "ForkManageExec", 400000)
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
	return &ManageAction{}
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

func (m ManageType) GetLogMap() map[int64]*types.LogInfo {
	return logmap
}

// GetRealToAddr 重载该函数主要原因是manage的协议在实现过程中，不同高度的To地址规范不一样
func (m ManageType) GetRealToAddr(tx *types.Transaction) string {
	if len(tx.To) == 0 {
		// 如果To地址为空，则认为是早期低于types.ForkV11ManageExec高度的交易，直接返回合约地址
		return address.ExecAddress(string(tx.Execer))
	}
	return tx.To
}

func (m ManageType) GetTypeMap() map[string]int32 {
	return actionName
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
