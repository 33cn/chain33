package executor

import (
	log "github.com/inconshreveable/log15"
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
)

var clog = log.New("module", "execs.hashlock")

const minLockTime = 60

var driverName = "hashlock"

func init() {
	ety := types.LoadExecutorType(driverName)
	ety.InitFuncList(types.ListMethod(&Hashlock{}))
}

func Init(name string, sub []byte) {
	drivers.Register(GetName(), newHashlock, types.GetDappFork(driverName, "Enable"))
}

func GetName() string {
	return newHashlock().GetName()
}

type Hashlock struct {
	drivers.DriverBase
}

func newHashlock() drivers.Driver {
	h := &Hashlock{}
	h.SetChild(h)
	h.SetExecutorType(types.LoadExecutorType(driverName))
	return h
}

func (h *Hashlock) GetDriverName() string {
	return driverName
}

func (h *Hashlock) CheckTx(tx *types.Transaction, index int) error {
	return nil
}
