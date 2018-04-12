package manage

import (
	"gitlab.33.cn/chain33/chain33/account"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/types"
)

type Action struct {
	coinsAccount *account.DB
	db           dbm.KVDB
	txhash       []byte
	fromaddr     string
	blocktime    int64
	height       int64
}

func NewAction(m *Manage, tx *types.Transaction) *Action {
	return &Action{db: m.GetStateDB(), fromaddr: account.PubKeyToAddress(tx.GetSignature().GetPubkey()).String()}

}

func (m *Action) modifyConfig(modify *types.ModifyConfig) (*types.Receipt, error) {

	//if modify.Key == "Manager-managers" && !wallet.IsSuperManager(modify.GetAddr()) {
	//	return nil, types.ErrNoPrivilege
	//}
	//if !wallet.IsSuperManager(m.fromaddr) && ! IsManager(m.fromaddr) {
	//	return nil, types.ErrNoPrivilege
	//}

	if !IsSuperManager(m.fromaddr) {
		return nil, types.ErrNoPrivilege
	}
	if len(modify.Key) == 0 {
		return nil, types.ErrBadConfigKey
	}
	if modify.Op != "add" && modify.Op != "delete" {
		return nil, types.ErrBadConfigOp
	}

	var item types.ConfigItem
	value, err := m.db.Get([]byte(types.ConfigKey(modify.Key)))
	if err != nil {
		value = nil
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
		item.Ty = types.ConfigItemArrayConfig
		emptyValue := &types.ArrayConfig{make([]string, 0)}
		arr := types.ConfigItem_Arr{emptyValue}
		item.Value = &arr
	}
	copyValue := *item.GetArr()
	copyItem := item
	copyItem.Value = &types.ConfigItem_Arr{&copyValue}

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
	key := types.ConfigKey(modify.Key)
	valueSave := types.Encode(&item)
	m.db.Set([]byte(key), valueSave)
	kv = append(kv, &types.KeyValue{[]byte(key), valueSave})
	log := types.ReceiptConfig{Prev: &copyItem, Current: &item}
	logs = append(logs, &types.ReceiptLog{Ty: types.TyLogModifyConfig, Log: types.Encode(&log)})
	receipt := &types.Receipt{Ty: types.ExecOk, KV: kv, Logs: logs}
	return receipt, nil
}
