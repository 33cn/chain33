package types

import (
	"gitlab.33.cn/chain33/chain33/types"
)

var nameX string

var (
	actionName = map[string]int32{
		"Nput": NormActionPut,
	}
)

func init() {
	nameX = types.ExecName("norm")
	// init executor type
	types.RegistorExecutor("norm", NewType())
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

func (norm *NormType) ActionName(tx *types.Transaction) string {
	var action NormAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return "unknow"
	}
	if action.Ty == NormActionPut && action.GetNput() != nil {
		return "put"
	}
	return "unknow"
}

func (norm *NormType) GetTypeMap() map[string]int32 {
	return actionName
}
