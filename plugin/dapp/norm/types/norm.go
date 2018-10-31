package types

import (
	"gitlab.33.cn/chain33/chain33/types"
)

var NormX = "norm"

func init() {
	types.AllowUserExec = append(types.AllowUserExec, []byte(NormX))
	types.RegistorExecutor(NormX, NewType())
	types.RegisterDappFork(NormX, "Enable", 0)
}

type NormType struct {
	types.ExecTypeBase
}

func NewType() *NormType {
	c := &NormType{}
	c.SetChild(c)
	return c
}

func (norm *NormType) GetPayload() types.Message {
	return &NormAction{}
}

func (norm *NormType) GetTypeMap() map[string]int32 {
	return map[string]int32{
		"Nput": NormActionPut,
	}
}

func (at *NormType) GetLogMap() map[int64]*types.LogInfo {
	return map[int64]*types.LogInfo{}
}
