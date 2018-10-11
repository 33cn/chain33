package types

import (
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/types"
)

var nameX string

var (
	vlog       = log.New("module", "exectype.valnode")
	actionName = map[string]int32{
		"Node":      ValNodeActionUpdate,
		"BlockInfo": ValNodeActionBlockInfo,
	}
)

func init() {
	nameX = types.ExecName("valnode")
	// init executor type
	types.RegistorExecutor("valnode", NewType())
}

type ValNodeType struct {
	types.ExecTypeBase
}

func NewType() *ValNodeType {
	c := &ValNodeType{}
	c.SetChild(c)
	return c
}

func (valType *ValNodeType) GetPayload() types.Message {
	return &ValNodeAction{}
}

func (valType *ValNodeType) ActionName(tx *types.Transaction) string {
	var action ValNodeAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return "unknow"
	}
	if action.Ty == ValNodeActionUpdate && action.GetNode() != nil {
		return "update"
	}
	if action.Ty == ValNodeActionBlockInfo && action.GetBlockInfo() != nil {
		return "blockInfo"
	}
	return "unknow"
}

func (valType *ValNodeType) GetTypeMap() map[string]int32 {
	return actionName
}
