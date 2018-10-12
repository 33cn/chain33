package executor

import (
	"fmt"
	"reflect"

	log "github.com/inconshreveable/log15"
	pty "gitlab.33.cn/chain33/chain33/plugin/dapp/valnode/types"
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
)

var clog = log.New("module", "execs.valnode")

//初始化过程比较重量级，有很多reflact, 所以弄成全局的
var executorFunList = make(map[string]reflect.Method)
var executorType = pty.NewType()

func init() {
	actionFunList := executorType.GetFuncMap()
	executorFunList = types.ListMethod(&ValNode{})
	for k, v := range actionFunList {
		executorFunList[k] = v
	}
}

func Init(name string) {
	clog.Debug("register norm execer")
	drivers.Register(GetName(), newValNode, 0)
}

func GetName() string {
	return newValNode().GetName()
}

type ValNode struct {
	drivers.DriverBase
}

func newValNode() drivers.Driver {
	n := &ValNode{}
	n.SetChild(n)
	n.SetIsFree(true)
	n.SetExecutorType(executorType)
	return n
}

func (val *ValNode) GetDriverName() string {
	return "valnode"
}

func (val *ValNode) GetActionName(tx *types.Transaction) string {
	return tx.ActionName()
}

func (n *ValNode) GetFuncMap() map[string]reflect.Method {
	return executorFunList
}

func (n *ValNode) CheckTx(tx *types.Transaction, index int) error {
	return nil
}

func CalcValNodeUpdateHeightIndexKey(height int64, index int) []byte {
	return []byte(fmt.Sprintf("ValNodeUpdate:%18d:%18d", height, int64(index)))
}

func CalcValNodeUpdateHeightKey(height int64) []byte {
	return []byte(fmt.Sprintf("ValNodeUpdate:%18d:", height))
}

func CalcValNodeBlockInfoHeightKey(height int64) []byte {
	return []byte(fmt.Sprintf("ValNodeBlockInfo:%18d:", height))
}
