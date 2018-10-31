package executor

import (
	"fmt"

	log "github.com/inconshreveable/log15"
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
)

var clog = log.New("module", "execs.valnode")
var driverName = "valnode"

func init() {
	ety := types.LoadExecutorType(driverName)
	ety.InitFuncList(types.ListMethod(&ValNode{}))
}

func Init(name string, sub []byte) {
	clog.Debug("register valnode execer")
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
	n.SetExecutorType(types.LoadExecutorType(driverName))
	return n
}

func (val *ValNode) GetDriverName() string {
	return driverName
}

func (val *ValNode) CheckTx(tx *types.Transaction, index int) error {
	return nil
}

func CalcValNodeUpdateHeightIndexKey(height int64, index int) []byte {
	return []byte(fmt.Sprintf("LODB-valnode-Update:%18d:%18d", height, int64(index)))
}

func CalcValNodeUpdateHeightKey(height int64) []byte {
	return []byte(fmt.Sprintf("LODB-valnode-Update:%18d:", height))
}

func CalcValNodeBlockInfoHeightKey(height int64) []byte {
	return []byte(fmt.Sprintf("LODB-valnode-BlockInfo:%18d:", height))
}
