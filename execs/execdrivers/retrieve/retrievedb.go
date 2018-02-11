package retrieve

import (
	//"bytes"

	"code.aliyun.com/chain33/chain33/account"
	//"code.aliyun.com/chain33/chain33/common"
	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/types"
	//log "github.com/inconshreveable/log15"
)

const (
	Retrieve_Backup    = 1
	Retrieve_Prepared  = 2
	Retrieve_Performed = 3
	Retrieve_Canceled  = 4
)

const MaxRelation = 10

type RetrieveDB struct {
	types.Retrieve
}

func NewRetrieveDB(backupaddress string) *RetrieveDB {
	r := &RetrieveDB{}
	r.BackupAddress = backupaddress
	r.RetPara = make([]*types.RetrievePara, MaxRelation)

	return r
}

func (r *RetrieveDB) RelateRetrieveDB(defaultAddress string, createTime int64, delayPeriod int64, index int32) bool {
	if index >= MaxRelation {
		return false
	}
	r.RetPara[index].DefaultAddress = defaultAddress
	r.RetPara[index].Status = Retrieve_Backup
	r.RetPara[index].CreateTime = createTime
	r.RetPara[index].DelayPeriod = delayPeriod
	return true
}

func (r *RetrieveDB) UnRelateRetrieveDB(index int) bool {
	r.RetPara = append(r.RetPara[:index], r.RetPara[index+1:]...)
	return true
}

func (r *RetrieveDB) CheckRelation(defaultAddress string) (int, bool) {
	for i := 0; i < len(r.RetPara); i++ {
		if r.RetPara[i].DefaultAddress == defaultAddress {
			return i, true
		}
	}
	return MaxRelation, false
}

func (r *RetrieveDB) GetKVSet() (kvset []*types.KeyValue) {
	value := types.Encode(r)
	kvset = append(kvset, &types.KeyValue{RetrieveKey(r.BackupAddress), value})
	return kvset
}

func (r *RetrieveDB) Save(db dbm.KVDB) {
	set := r.GetKVSet()
	for i := 0; i < len(set); i++ {
		db.Set(set[i].GetKey(), set[i].Value)
	}
}

func RetrieveKey(address string) (key []byte) {
	key = append(key, []byte("mavl-retrieve-")...)
	key = append(key, address...)
	return key
}

type RetrieveAction struct {
	db        dbm.KVDB
	txhash    []byte
	fromaddr  string
	blocktime int64
	height    int64
	execaddr  string
}

func NewRetrieveAcction(db dbm.KVDB, tx *types.Transaction, execaddr string, blocktime, height int64) *RetrieveAction {
	hash := tx.Hash()
	fromaddr := account.PubKeyToAddress(tx.GetSignature().GetPubkey()).String()
	return &RetrieveAction{db, hash, fromaddr, blocktime, height, execaddr}
}

//wait for valuable comment
func (action *RetrieveAction) RetrieveBackup(backupRet *types.BackupRetrieve) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	var receipt *types.Receipt
	var r *RetrieveDB
	var newRetrieve bool = false

	//用备份地址检索，如果没有，就建立新的，然后检查并处理关联
	retrieve, err := readRetrieve(action.db, backupRet.BackupAddress)
	if err != nil && err != types.ErrNotFound {
		rlog.Error("RetrieveBackup", "readRetrieve err")
		return nil, err
	} else if err == types.ErrNotFound {
		newRetrieve = true
		rlog.Debug("RetrieveBackup", "newAddress", backupRet.BackupAddress)
	}

	if newRetrieve {
		r = NewRetrieveDB(backupRet.BackupAddress)
	} else {
		r = &RetrieveDB{*retrieve}
	}

	if index, related := r.CheckRelation(backupRet.DefaultAddress); !related {
		if !r.RelateRetrieveDB(backupRet.DefaultAddress, action.blocktime, backupRet.DelayPeriod, int32(index)) {
			return nil, types.ErrRetrieveRelateLimit
		}
	} else {
		return nil, types.ErrRetrieveRepeatAddress
	}

	r.Save(action.db)
	logs = append(logs, receipt.Logs...)
	kv = append(kv, receipt.KV...)
	//logs = append(logs, h.GetReceiptLog())
	kv = append(kv, r.GetKVSet()...)

	receipt = &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil
}

func (action *RetrieveAction) RetrievePrepare(preRet *types.PreRetrieve) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	var receipt *types.Receipt
	var r *RetrieveDB
	var index int
	var related bool

	retrieve, err := readRetrieve(action.db, preRet.BackupAddress)
	if err != nil {
		rlog.Debug("RetrievePrepare", "readRetrieve err")
		return nil, err
	}
	r = &RetrieveDB{*retrieve}
	if action.fromaddr != r.BackupAddress {
		rlog.Debug("RetrievePrepare", "action.fromaddr", action.fromaddr, "r.BackupAddress", r.BackupAddress)
		return nil, types.ErrRetrievePrepareAddress
	}

	if index, related = r.CheckRelation(preRet.DefaultAddress); !related {
		return nil, types.ErrRetrieveRelation
	}

	if r.RetPara[index].Status != Retrieve_Backup {
		rlog.Debug("RetrieveBackup", "Status", r.RetPara[index].Status)
		return nil, types.ErrRetrieveStatus
	}
	r.RetPara[index].PrepareTime = action.blocktime
	r.RetPara[index].Status = Retrieve_Prepared

	r.Save(action.db)

	logs = append(logs, receipt.Logs...)
	kv = append(kv, receipt.KV...)
	//logs = append(logs, h.GetReceiptLog())
	kv = append(kv, r.GetKVSet()...)

	receipt = &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil
}

func (action *RetrieveAction) RetrievePerform(perfRet *types.PerformRetrieve) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	var receipt *types.Receipt
	var index int
	var related bool
	var acc *types.Account

	retrieve, err := readRetrieve(action.db, perfRet.BackupAddress)
	if err != nil {
		rlog.Debug("RetrievePerform", "readRetrieve err", perfRet.BackupAddress)
		return nil, err
	}

	r := &RetrieveDB{*retrieve}

	if index, related = r.CheckRelation(perfRet.DefaultAddress); !related {
		return nil, types.ErrRetrieveRelation
	}

	if r.BackupAddress != action.fromaddr {
		rlog.Debug("RetrievePerform", "BackupAddress", r.BackupAddress, "action.fromaddr", action.fromaddr)
		return nil, types.ErrRetrievePerformAddress
	}

	if r.RetPara[index].Status != Retrieve_Prepared {
		rlog.Debug("RetrievePerform", "Status", r.RetPara[index].Status)
		return nil, types.ErrRetrieveStatus
	}
	if action.blocktime-r.RetPara[index].PrepareTime < r.RetPara[index].DelayPeriod {
		rlog.Debug("RetrievePerform", "ErrRetrievePeriodLimit")
		return nil, types.ErrRetrievePeriodLimit
	}

	acc = account.LoadExecAccount(action.db, r.RetPara[index].DefaultAddress, action.execaddr)

	if acc.Balance > 0 {
		receipt, err = account.ExecTransfer(action.db, r.RetPara[index].DefaultAddress, r.BackupAddress, action.execaddr, acc.Balance)
		if err != nil {
			rlog.Debug("RetrievePerform", "ExecTransfer err")
			return nil, err
		}
	} else {
		return nil, types.ErrRetrieveNoBalance
	}

	//r.RetPara[index].Status = Retrieve_Performed
	//remove the relation
	r.UnRelateRetrieveDB(index)

	r.Save(action.db)
	logs = append(logs, receipt.Logs...)
	kv = append(kv, receipt.KV...)
	//logs = append(logs, t.GetReceiptLog())
	kv = append(kv, r.GetKVSet()...)

	receipt = &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil
}

func (action *RetrieveAction) RetrieveCancel(cancel *types.CancelRetrieve) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	var receipt *types.Receipt
	var index int
	var related bool

	retrieve, err := readRetrieve(action.db, cancel.BackupAddress)
	if err != nil {
		rlog.Debug("RetrieveCancel", "readRetrieve err", cancel.BackupAddress)
		return nil, err
	}
	r := &RetrieveDB{*retrieve}

	if index, related = r.CheckRelation(cancel.DefaultAddress); !related {
		return nil, types.ErrRetrieveRelation
	}

	if action.fromaddr != r.RetPara[index].DefaultAddress {
		rlog.Debug("RetrieveCancel", "action.fromaddr", action.fromaddr, "DefaultAddress", r.RetPara[index].DefaultAddress)
		return nil, types.ErrRetrieveCancelAddress
	}

	if r.RetPara[index].Status != Retrieve_Prepared {
		rlog.Debug("RetrieveCancel", "Status", r.RetPara[index].Status)
		return nil, types.ErrRetrieveStatus
	}

	//r.RetPara[index].Status = Retrieve_Canceled
	//remove the relation
	r.UnRelateRetrieveDB(index)
	r.Save(action.db)

	logs = append(logs, receipt.Logs...)
	kv = append(kv, receipt.KV...)
	//logs = append(logs, h.GetReceiptLog())
	kv = append(kv, r.GetKVSet()...)

	receipt = &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil
}

func readRetrieve(db dbm.KVDB, address string) (*types.Retrieve, error) {
	data, err := db.Get(RetrieveKey(address))
	if err != nil {
		return nil, err
	}
	var retrieve types.Retrieve
	//decode
	err = types.Decode(data, &retrieve)
	if err != nil {
		return nil, err
	}
	return &retrieve, nil
}
