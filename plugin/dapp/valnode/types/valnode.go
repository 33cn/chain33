package types

import (
	"gitlab.33.cn/chain33/chain33/types"
)

var ValNodeX = "valnode"

func init() {
	types.AllowUserExec = append(types.AllowUserExec, []byte(ValNodeX))
	types.RegistorExecutor(ValNodeX, NewType())
	types.RegisterDappFork(ValNodeX, "Enable", 0)
}

// exec
type ValNodeType struct {
	types.ExecTypeBase
}

func NewType() *ValNodeType {
	c := &ValNodeType{}
	c.SetChild(c)
	return c
}

func (t *ValNodeType) GetPayload() types.Message {
	return &ValNodeAction{}
}

func (t *ValNodeType) GetTypeMap() map[string]int32 {
	return map[string]int32{
		"Node":      ValNodeActionUpdate,
		"BlockInfo": ValNodeActionBlockInfo,
	}
}

func (t *ValNodeType) GetLogMap() map[int64]*types.LogInfo {
	return map[int64]*types.LogInfo{}
}
