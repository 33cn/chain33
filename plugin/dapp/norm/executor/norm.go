package executor

import (
	"reflect"

	log "github.com/inconshreveable/log15"
	pty "gitlab.33.cn/chain33/chain33/plugin/dapp/norm/types"
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
)

var clog = log.New("module", "execs.norm")

//初始化过程比较重量级，有很多reflact, 所以弄成全局的
var executorFunList = make(map[string]reflect.Method)
var executorType = pty.NewType()

func init() {
	actionFunList := executorType.GetFuncMap()
	executorFunList = types.ListMethod(&Norm{})
	for k, v := range actionFunList {
		executorFunList[k] = v
	}
}

func Init(name string) {
	clog.Debug("register norm execer")
	drivers.Register(GetName(), newNorm, 0)
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
	n.SetExecutorType(executorType)
	return n
}

func (n *Norm) GetDriverName() string {
	return "norm"
}

//获取运行状态名
func (n *Norm) GetActionName(tx *types.Transaction) string {
	return tx.ActionName()
}

func (n *Norm) GetFuncMap() map[string]reflect.Method {
	return executorFunList
}

func (n *Norm) CheckTx(tx *types.Transaction, index int) error {
	return nil
}

func Key(str string) (key []byte) {
	key = append(key, []byte("mavl-norm-")...)
	key = append(key, str...)
	return key
}
