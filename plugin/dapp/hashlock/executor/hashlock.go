package executor

import (
	"reflect"

	log "github.com/inconshreveable/log15"
	pty "gitlab.33.cn/chain33/chain33/plugin/dapp/hashlock/types"
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
)

var clog = log.New("module", "execs.hashlock")

const minLockTime = 60

//初始化过程比较重量级，有很多reflact, 所以弄成全局的
var executorFunList = make(map[string]reflect.Method)
var executorType = pty.NewType()

func init() {
	actionFunList := executorType.GetFuncMap()
	executorFunList = types.ListMethod(&Hashlock{})
	for k, v := range actionFunList {
		executorFunList[k] = v
	}
}

func Init(name string) {
	drivers.Register(GetName(), newHashlock, 0)
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
	h.SetExecutorType(executorType)
	return h
}

func (h *Hashlock) GetDriverName() string {
	return "hashlock"
}

//获取运行状态名
func (h *Hashlock) GetActionName(tx *types.Transaction) string {
	return tx.ActionName()
}

func (h *Hashlock) GetFuncMap() map[string]reflect.Method {
	return executorFunList
}

func (h *Hashlock) CheckTx(tx *types.Transaction, index int) error {
	return nil
}
