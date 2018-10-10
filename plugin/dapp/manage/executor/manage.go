package executor

/*
manage 负责管理配置
 1. 添加管理
 1. 添加运营人员
 1. （未来）修改某些配置项
*/

import (
	"reflect"

	log "github.com/inconshreveable/log15"
	pty "gitlab.33.cn/chain33/chain33/plugin/dapp/manage/types"
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	clog = log.New("module", "execs.manage")

	//初始化过程比较重量级，有很多reflact, 所以弄成全局的
	executorFunList = make(map[string]reflect.Method)
	executorType    = pty.NewType()
)

func init() {
	actionFunList := executorType.GetFuncMap()
	executorFunList = types.ListMethod(&Manage{})
	for k, v := range actionFunList {
		executorFunList[k] = v
	}
}

func Init(name string) {
	drivers.Register(GetName(), newManage, types.ForkV4AddManage)
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
	c.SetExecutorType(executorType)
	return c
}

func (c *Manage) GetDriverName() string {
	return "manage"
}

func (c *Manage) CheckTx(tx *types.Transaction, index int) error {
	return nil
}

func (c *Manage) GetFuncMap() map[string]reflect.Method {
	return executorFunList
}

func IsSuperManager(addr string) bool {
	for _, m := range types.SuperManager {
		if addr == m {
			return true
		}
	}
	return false
}
