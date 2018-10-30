package executor

import (
	log "github.com/inconshreveable/log15"
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
)

var clog = log.New("module", "execs.norm")
var driverName = "norm"

func init() {
	ety := types.LoadExecutorType(driverName)
	ety.InitFuncList(types.ListMethod(&Norm{}))
}

func Init(name string, sub []byte) {
	clog.Debug("register norm execer")
	drivers.Register(GetName(), newNorm, types.GetDappFork(driverName, "Enable"))
}

func GetName() string {
	return newNorm().GetName()
}

type Norm struct {
	drivers.DriverBase
}

func newNorm() drivers.Driver {
	n := &Norm{}
	n.SetChild(n)
	n.SetIsFree(true)
	n.SetExecutorType(types.LoadExecutorType(driverName))
	return n
}

func (n *Norm) GetDriverName() string {
	return driverName
}

func (n *Norm) CheckTx(tx *types.Transaction, index int) error {
	return nil
}

func Key(str string) (key []byte) {
	key = append(key, []byte("mavl-norm-")...)
	key = append(key, str...)
	return key
}
