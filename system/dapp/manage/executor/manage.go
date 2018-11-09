package executor

/*
manage 负责管理配置
 1. 添加管理
 1. 添加运营人员
 1. （未来）修改某些配置项
*/

import (
	log "gitlab.33.cn/chain33/chain33/common/log/log15"
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	clog       = log.New("module", "execs.manage")
	driverName = "manage"
	conf       = types.ConfSub(driverName)
)

func init() {
	ety := types.LoadExecutorType(driverName)
	ety.InitFuncList(types.ListMethod(&Manage{}))
}

func Init(name string, sub []byte) {
	drivers.Register(GetName(), newManage, types.GetDappFork(driverName, "Enable"))
}

func GetName() string {
	return newManage().GetName()
}

type Manage struct {
	drivers.DriverBase
}

func newManage() drivers.Driver {
	c := &Manage{}
	c.SetChild(c)
	c.SetExecutorType(types.LoadExecutorType(driverName))
	return c
}

func (c *Manage) GetDriverName() string {
	return driverName
}

func (c *Manage) CheckTx(tx *types.Transaction, index int) error {
	return nil
}

func IsSuperManager(addr string) bool {
	for _, m := range conf.GStrList("superManager") {
		if addr == m {
			return true
		}
	}
	return false
}
