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

func (c *Manage) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := c.DriverBase.ExecLocal(tx, receipt, index)
	if err != nil {
		return nil, err
	}
	if receipt.GetTy() != types.ExecOk {
		return set, nil
	}
	//执行成功
	var manageAction pty.ManageAction
	err = types.Decode(tx.Payload, &manageAction)
	if err != nil {
		return nil, err
	}
	clog.Info("manage.ExecLocal", "ty", manageAction.Ty)

	for i := 0; i < len(receipt.Logs); i++ {
		item := receipt.Logs[i]
		if item.Ty == pty.ManageActionModifyConfig {
			var receipt types.ReceiptConfig
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			key := receipt.Current.Key
			set.KV = append(set.KV, &types.KeyValue{Key: []byte(key), Value: types.Encode(receipt.Current)})
			clog.Debug("ExecLocal to savelogs", "config ", key, "value", receipt.Current)
		}
	}
	return set, nil
}

func (c *Manage) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := c.DriverBase.ExecDelLocal(tx, receipt, index)
	if err != nil {
		return nil, err
	}
	if receipt.GetTy() != types.ExecOk {
		return set, nil
	}
	//执行成功
	var manageAction pty.ManageAction
	err = types.Decode(tx.Payload, &manageAction)
	if err != nil {
		return nil, err
	}
	clog.Info("manage.ExecDelLocal", "ty", manageAction.Ty)

	for i := 0; i < len(receipt.Logs); i++ {
		item := receipt.Logs[i]
		if item.Ty == pty.ManageActionModifyConfig {
			var receipt types.ReceiptConfig
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			key := receipt.Current.Key
			set.KV = append(set.KV, &types.KeyValue{Key: []byte(key), Value: types.Encode(receipt.Prev)})
			clog.Debug("ExecDelLocal to savelogs", "config ", key, "value", receipt.Prev)
		}
	}
	return set, nil
}

func IsSuperManager(addr string) bool {
	for _, m := range types.SuperManager {
		if addr == m {
			return true
		}
	}
	return false
}
