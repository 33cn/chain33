package manage

/*
manage 负责管理配置
 1. 添加管理
 1. 添加运营人员
 1. （未来）修改某些配置项
*/

import (
	"fmt"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/types"
)

var clog = log.New("module", "execs.manage")

func init() {
	n := newManage()
	drivers.Register(n.GetName(), n, types.ForkV4_add_manage)
}

type Manage struct {
	drivers.DriverBase
}

func newManage() *Manage {
	c := &Manage{}
	c.SetChild(c)
	return c
}

func (c *Manage) GetName() string {
	return "manage"
}

func (c *Manage) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	clog.Info("manage.Exec", "start index", index)
	//_, err := c.DriverBase.Exec(tx, index)
	//if err != nil {
	//	return nil, err
	//}
	//clog.Info("manage.Exec", "start index 2", index)
	var manageAction types.ManageAction
	err := types.Decode(tx.Payload, &manageAction)
	if err != nil {
		return nil, err
	}
	clog.Info("manage.Exec", "ty", manageAction.Ty)
	if manageAction.Ty == types.ManageActionModifyConfig {
		if manageAction.GetModify() == nil {
			return nil, types.ErrInputPara
		}
		action := NewAction(c, tx)
		return action.modifyConfig(manageAction.GetModify())
	}

	return nil, types.ErrActionNotSupport
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
	var manageAction types.ManageAction
	err = types.Decode(tx.Payload, &manageAction)
	if err != nil {
		return nil, err
	}
	clog.Info("manage.ExecLocal", "ty", manageAction.Ty)

	for i := 0; i < len(receipt.Logs); i++ {
		item := receipt.Logs[i]
		if item.Ty == types.ManageActionModifyConfig {
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
	var manageAction types.ManageAction
	err = types.Decode(tx.Payload, &manageAction)
	if err != nil {
		return nil, err
	}
	clog.Info("manage.ExecDelLocal", "ty", manageAction.Ty)

	for i := 0; i < len(receipt.Logs); i++ {
		item := receipt.Logs[i]
		if item.Ty == types.ManageActionModifyConfig {
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

func (c *Manage) Query(funcName string, params []byte) (types.Message, error) {
	if funcName == "GetConfigItem" {
		var in types.ReqString
		err := types.Decode(params, &in)
		if err != nil {
			return nil, err
		}

		// Load config from store
		value, err := c.GetDB().Get([]byte(types.ConfigKey(in.Data)))
		if err != nil {
			clog.Info("modifyConfig", "get db key", "not found")
			value = nil
		}

		var reply types.ReplyConfig
		reply.Key = in.Data

		var item types.ConfigItem
		if value != nil {
			err = types.Decode(value, &item)
			if err != nil {
				clog.Error("modifyConfig", "get db key", in.Data)
				return nil, err // types.ErrBadConfigValue
			}
			reply.Value = fmt.Sprint(item.GetArr().Value)
		} else { // if config item not exist
			reply.Value = ""
		}
		clog.Info("manage  Query", "key ", in.Data)

		return &reply, nil
	}
	return nil, types.ErrActionNotSupport
}

func IsSuperManager(addr string) bool {
	for _, m := range types.SuperManager {
		if addr == m {
			return true
		}
	}
	return false
}
