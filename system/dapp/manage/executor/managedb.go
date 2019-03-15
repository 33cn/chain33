// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	dbm "github.com/33cn/chain33/common/db"
	pty "github.com/33cn/chain33/system/dapp/manage/types"
	"github.com/33cn/chain33/types"
)

// Action attribute
type Action struct {
	db       dbm.KV
	fromaddr string
	height   int64
}

// NewAction new a action object
func NewAction(m *Manage, tx *types.Transaction) *Action {
	return &Action{db: m.GetStateDB(), fromaddr: tx.From(), height: m.GetHeight()}

}

func (m *Action) modifyConfig(modify *types.ModifyConfig) (*types.Receipt, error) {

	//if modify.Key == "Manager-managers" && !wallet.IsSuperManager(modify.GetAddr()) {
	//	return nil, types.ErrNoPrivilege
	//}
	//if !wallet.IsSuperManager(m.fromaddr) && ! IsManager(m.fromaddr) {
	//	return nil, types.ErrNoPrivilege
	//}

	if !IsSuperManager(m.fromaddr) {
		return nil, pty.ErrNoPrivilege
	}
	if len(modify.Key) == 0 {
		return nil, pty.ErrBadConfigKey
	}
	if modify.Op != "add" && modify.Op != "delete" {
		return nil, pty.ErrBadConfigOp
	}

	var item types.ConfigItem
	value, err := m.db.Get([]byte(types.ManageKey(modify.Key)))
	if err != nil {
		value = nil
	}
	if value == nil {
		value, err = m.db.Get([]byte(types.ConfigKey(modify.Key)))
		if err != nil {
			value = nil
		}
	}
	if value != nil {
		err = types.Decode(value, &item)
		if err != nil {
			clog.Error("modifyConfig", "decode db key", modify.Key)
			return nil, err // types.ErrBadConfigValue
		}
	} else { // if config item not exist, create a new empty
		item.Key = modify.Key
		item.Addr = modify.Addr
		item.Ty = pty.ConfigItemArrayConfig
		emptyValue := &types.ArrayConfig{Value: make([]string, 0)}
		arr := types.ConfigItem_Arr{Arr: emptyValue}
		item.Value = &arr
	}
	copyValue := *item.GetArr()
	copyItem := item
	copyItem.Value = &types.ConfigItem_Arr{Arr: &copyValue}

	switch modify.Op {
	case "add":
		item.GetArr().Value = append(item.GetArr().Value, modify.Value)
		item.Addr = modify.Addr
		clog.Info("modifyConfig", "add key", modify.Key, "from", copyItem.GetArr().Value, "to", item.GetArr().Value)

	case "delete":
		item.Addr = modify.Addr
		item.GetArr().Value = make([]string, 0)
		for _, value := range copyItem.GetArr().Value {
			clog.Info("modifyConfig", "key delete", modify.Key, "current", value)
			if value != modify.Value {
				item.GetArr().Value = append(item.GetArr().Value, value)
			}
		}
		clog.Info("modifyConfig", "delete key", modify.Key, "from", copyItem.GetArr().Value, "to", item.GetArr().Value)
		/*
			case "assign":
				if item.Ty == types.ConfigItemIntConfig {
					intvalue, err := strconv.Atoi(modify.Value)
					if err != nil {
						clog.Error("modifyConfig", "key", modify.Key, "strconv.Atoi", err)
					}
					item.GetInt().Value = int32(intvalue)
					clog.Info("modifyConfig", "key", modify.Key, "from", copyItem.GetInt().Value, "to", item.GetInt().Value)
				} else {
					item.GetStr().Value = modify.Value
					clog.Info("modifyConfig", "key", modify.Key, "from", copyItem.GetStr().Value, "to", item.GetStr().Value)
				}*/
	}

	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	key := types.ManaeKeyWithHeigh(modify.Key, m.height)
	valueSave := types.Encode(&item)
	err = m.db.Set([]byte(key), valueSave)
	if err != nil {
		return nil, err
	}
	kv = append(kv, &types.KeyValue{Key: []byte(key), Value: valueSave})
	log := types.ReceiptConfig{Prev: &copyItem, Current: &item}
	logs = append(logs, &types.ReceiptLog{Ty: pty.TyLogModifyConfig, Log: types.Encode(&log)})
	receipt := &types.Receipt{Ty: types.ExecOk, KV: kv, Logs: logs}
	return receipt, nil
}
