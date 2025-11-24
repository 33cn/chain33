// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"github.com/33cn/chain33/account"
	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/common"
	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/system/dapp"
	mty "github.com/33cn/chain33/system/dapp/manage/types"
	"github.com/33cn/chain33/types"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

type action struct {
	api          client.QueueProtocolAPI
	coinsAccount *account.DB
	db           dbm.KV
	txhash       []byte
	fromaddr     string
	height       int64
	index        int32
	execaddr     string
}

func newAction(m *Manage, tx *types.Transaction, index int32) *action {
	return &action{m.GetAPI(), m.GetCoinsAccount(), m.GetStateDB(), tx.Hash(), tx.From(),
		m.GetHeight(), index, dapp.ExecAddress(string(tx.Execer))}
}

func (a *action) modifyConfig(modify *types.ModifyConfig) (*types.Receipt, error) {

	//if modify.Key == "Manager-managers" && !wallet.IsSuperManager(modify.GetAddr()) {
	//	return nil, types.ErrNoPrivilege
	//}
	//if !wallet.IsSuperManager(m.fromaddr) && ! IsManager(m.fromaddr) {
	//	return nil, types.ErrNoPrivilege
	//}

	if len(modify.Key) == 0 {
		return nil, mty.ErrBadConfigKey
	}
	if modify.Op != "add" && modify.Op != "delete" {
		return nil, mty.ErrBadConfigOp
	}

	var item types.ConfigItem
	value, err := a.db.Get([]byte(types.ManageKey(modify.Key)))
	if err != nil {
		value = nil
	}
	if value == nil {
		value, err = a.db.Get([]byte(types.ConfigKey(modify.Key)))
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
		item.Ty = mty.ConfigItemArrayConfig
		emptyValue := &types.ArrayConfig{Value: make([]string, 0)}
		arr := types.ConfigItem_Arr{Arr: emptyValue}
		item.Value = &arr
	}
	copyValue := *item.GetArr()
	copyItem := types.ConfigItem{
		Ty:   item.Ty,
		Addr: item.Addr,
		Key:  item.Key,
	}
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
	key := a.manageKeyWithHeigh(modify.Key)
	valueSave := types.Encode(&item)
	err = a.db.Set(key, valueSave)
	if err != nil {
		return nil, err
	}
	kv = append(kv, &types.KeyValue{Key: key, Value: valueSave})
	log := types.ReceiptConfig{Prev: &copyItem, Current: &item}
	logs = append(logs, &types.ReceiptLog{Ty: mty.TyLogModifyConfig, Log: types.Encode(&log)})
	receipt := &types.Receipt{Ty: types.ExecOk, KV: kv, Logs: logs}
	return receipt, nil
}

// ManaeKeyWithHeigh 超级管理员账户key
func (a *action) manageKeyWithHeigh(key string) []byte {
	if a.api.GetConfig().IsFork(a.height, "ForkExecKey") {
		return []byte(types.ManageKey(key))
	}
	return []byte(types.ConfigKey(key))
}

func (a *action) applyConfig(apply *mty.ApplyConfig) (*types.Receipt, error) {
	if apply.Config == nil {
		return nil, errors.Wrapf(types.ErrInvalidParam, "modify is nil")
	}
	if len(apply.Config.Key) <= 0 || len(apply.Config.Value) <= 0 {
		return nil, errors.Wrapf(types.ErrInvalidParam, "key=%s,val=%s", apply.Config.Key, apply.GetConfig().Value)
	}
	if apply.Config.Op != mty.OpAdd && apply.Config.Op != mty.OpDelete {
		return nil, errors.Wrapf(mty.ErrBadConfigOp, "op=%s", apply.Config.Op)
	}

	configStatus := &mty.ConfigStatus{
		Id:       common.ToHex(a.txhash),
		Config:   apply.Config,
		Status:   mty.ManageConfigStatusApply,
		Proposer: a.fromaddr,
		Height:   a.height,
		Index:    a.index,
	}

	return makeApplyReceipt(configStatus), nil
}

func getConfig(db dbm.KV, ID string) (*mty.ConfigStatus, error) {
	value, err := db.Get(managerIDKey(ID))
	if err != nil {
		return nil, err
	}
	var status mty.ConfigStatus
	err = types.Decode(value, &status)
	if err != nil {
		return nil, err
	}
	return &status, nil
}

func (a *action) approveConfig(approve *mty.ApproveConfig) (*types.Receipt, error) {
	if len(approve.AutonomyItemId) <= 0 || len(approve.ApplyConfigId) <= 0 {
		return nil, errors.Wrapf(types.ErrInvalidParam, "id nil, appoved=%s,id=%s", approve.AutonomyItemId, approve.ApplyConfigId)
	}

	s, err := getConfig(a.db, approve.ApplyConfigId)
	if err != nil {
		return nil, errors.Wrapf(err, "get Config id=%s", approve.ApplyConfigId)
	}

	if s.Status != mty.ManageConfigStatusApply {
		return nil, errors.Wrapf(types.ErrNotAllow, "id status =%d", s.Status)
	}

	cfg := a.api.GetConfig()
	confManager := types.ConfSub(cfg, mty.ManageX)
	autonomyExec := confManager.GStr(types.AutonomyCfgKey)
	if len(autonomyExec) <= 0 {
		return nil, errors.Wrapf(types.ErrNotFound, "manager autonomy key not config")
	}

	//去autonomy 合约检验是否id approved, 成功 err返回nil
	_, err = a.api.QueryChain(&types.ChainExecutor{
		Driver:   autonomyExec,
		FuncName: "IsAutonomyApprovedItem",
		Param:    types.Encode(&types.ReqMultiStrings{Datas: []string{approve.AutonomyItemId, approve.ApplyConfigId}}),
	})
	if err != nil {
		return nil, errors.Wrapf(err, "query autonomy,approveid=%s,hashId=%s", approve.AutonomyItemId, approve.ApplyConfigId)
	}

	copyStat := proto.Clone(s).(*mty.ConfigStatus)
	s.Status = mty.ManageConfigStatusApproved

	r := makeApproveReceipt(copyStat, s)

	cr, err := a.modifyConfig(s.Config)
	if err != nil {
		return nil, errors.Wrap(err, "modify config")
	}

	return mergeReceipt(r, cr), nil

}

func mergeReceipt(receipt1, receipt2 *types.Receipt) *types.Receipt {
	if receipt2 != nil {
		receipt1.KV = append(receipt1.KV, receipt2.KV...)
		receipt1.Logs = append(receipt1.Logs, receipt2.Logs...)
	}

	return receipt1
}

func makeApplyReceipt(status *mty.ConfigStatus) *types.Receipt {
	key := managerIDKey(status.Id)
	log := &mty.ReceiptApplyConfig{
		Status: status,
	}

	return &types.Receipt{
		Ty: types.ExecOk,
		KV: []*types.KeyValue{
			{Key: key, Value: types.Encode(status)},
		},
		Logs: []*types.ReceiptLog{
			{
				Ty:  mty.TyLogApplyConfig,
				Log: types.Encode(log),
			},
		},
	}
}

func makeApproveReceipt(pre, cur *mty.ConfigStatus) *types.Receipt {
	key := managerIDKey(cur.Id)
	log := &mty.ReceiptApproveConfig{
		Pre: pre,
		Cur: cur,
	}

	return &types.Receipt{
		Ty: types.ExecOk,
		KV: []*types.KeyValue{
			{Key: key, Value: types.Encode(cur)},
		},
		Logs: []*types.ReceiptLog{
			{
				Ty:  mty.TyLogApproveConfig,
				Log: types.Encode(log),
			},
		},
	}
}
