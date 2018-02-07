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

type RetrieveDB struct {
	types.Retrieve
}

//PrepareTime & DelayTime is decided in parepare action
func NewRetrieveDB(address string, defaultAddress string, blocktime int64, amount int64) *RetrieveDB {
	r := &RetrieveDB{}
	r.Address = address
	r.DefaultAddress = defaultAddress
	r.CreateTime = blocktime
	r.PrepareTime = 0
	r.Amount = amount
	r.DelayTime = 0
	r.Status = Retrieve_Backup
	return r
}

func (r *RetrieveDB) GetKVSet() (kvset []*types.KeyValue) {
	value := types.Encode(&r.Retrieve)
	kvset = append(kvset, &types.KeyValue{RetrieveKey(r.Address), value})
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

	retrieve, err := readRetrieve(action.db, backupRet.Address)
	if err != nil && err != types.ErrNotFound {
		rlog.Error("RetrieveBackup", "readRetrieve err")
		return nil, err
	} else if err == types.ErrNotFound {
		newRetrieve = true
		rlog.Debug("RetrieveBackup", "newAddress", backupRet.Address)
	} else if retrieve != nil {
		if retrieve.Status == Retrieve_Backup {
			rlog.Debug("RetrieveBackup", "Updatebackup", backupRet.Address)
		} else {
			rlog.Error("RetrieveBackup", "retrieve.Status", retrieve.Status)
			return nil, types.ErrRetrieveStatus
		}
	}
	if newRetrieve == true {
		if backupRet.Amount < 0 {
			rlog.Debug("RetrieveBackup", "Amount", backupRet.Amount)
			return nil, types.ErrRetrieveAmountLimit
		}
		r = NewRetrieveDB(backupRet.Address, action.fromaddr, action.blocktime, backupRet.Amount)

		receipt, err = account.ExecFrozen(action.db, action.fromaddr, action.execaddr, backupRet.Amount)
	} else {
		if action.fromaddr != r.DefaultAddress {
			rlog.Error("RetrieveBackup", "address repeated", backupRet.Address)
			return nil, types.ErrRetrieveRepeatAddress
		}
		r = &RetrieveDB{*retrieve}
		if backupRet.Amount > 0 {
			receipt, err = account.ExecFrozen(action.db, action.fromaddr, action.execaddr, backupRet.Amount)
			if err != nil {
				rlog.Error("RetrieveBackup", "ExecFrozen err")
				return nil, err
			}
			r.Amount += backupRet.Amount
		} else {
			if backupRet.Amount+r.Amount < 0 {
				rlog.Debug("RetrieveBackup", "Amount", backupRet.Amount, "current amount", r.Amount)
				return nil, types.ErrRetrieveAmountLimit
			}
			receipt, err = account.ExecActive(action.db, action.fromaddr, action.execaddr, backupRet.Amount)
			if err != nil {
				rlog.Error("RetrieveBackup", "ExecActive err")
				return nil, err
			}
			r.Amount += backupRet.Amount
		}
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

	retrieve, err := readRetrieve(action.db, preRet.Address)
	if err != nil {
		rlog.Debug("RetrievePrepare", "readRetrieve err")
		return nil, err
	}
	r = &RetrieveDB{*retrieve}
	if action.fromaddr != r.Address {
		rlog.Debug("RetrievePrepare", "action.fromaddr", action.fromaddr, "r.Address", r.Address)
		return nil, types.ErrRetrievePrepareAddress
	}

	if r.Status != Retrieve_Backup {
		rlog.Debug("RetrieveBackup", "r.Status", r.Status)
		return nil, types.ErrRetrieveStatus
	}
	r.PrepareTime = action.blocktime
	r.Status = Retrieve_Prepared
	r.DelayTime = preRet.DelayPeriod
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

	retrieve, err := readRetrieve(action.db, perfRet.Address)
	if err != nil {
		rlog.Error("RetrievePerform", "readRetrieve err", perfRet.Address)
		return nil, err
	}

	if retrieve.Status != Retrieve_Prepared {
		rlog.Error("RetrievePerform", "retrieve.Status", retrieve.Status)
		return nil, types.ErrRetrieveStatus
	}

	if retrieve.Address != action.fromaddr {
		rlog.Error("RetrievePerform", "retrieve.Address", retrieve.Address, "action.fromaddr", action.fromaddr)
		return nil, types.ErrRetrievePerformAddress
	}

	if perfRet.Timeweight < maxTimeWeight {
		if action.blocktime-retrieve.GetPrepareTime() < (maxTimeWeight-perfRet.Timeweight)*retrieve.DelayTime {
			rlog.Error("RetrievePerform", "action.blocktime-retrieve.GetCreateTime", action.blocktime-retrieve.GetCreateTime())
			return nil, types.ErrRetrievePeriodLimit
		}
	}

	r := &RetrieveDB{*retrieve}

	receipt, err = account.ExecTransferFrozen(action.db, r.DefaultAddress, r.Address, action.execaddr, r.Amount)
	if err != nil {
		rlog.Error("RetrievePerform", "ExecTransferFrozen err")
		return nil, err
	}
	r.Status = Retrieve_Performed
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

	retrieve, err := readRetrieve(action.db, cancel.Address)
	if err != nil {
		rlog.Error("RetrieveCancel", "readRetrieve err", cancel.Address)
		return nil, err
	}
	r := &RetrieveDB{*retrieve}
	if r.Status != Retrieve_Prepared {
		rlog.Error("RetrieveBackup", "r.Status", r.Status)
		return nil, types.ErrRetrieveStatus
	}

	if action.fromaddr != r.DefaultAddress {
		rlog.Error("RetrieveCancel", "action.fromaddr", action.fromaddr, "r.DefaultAddress", r.DefaultAddress)
		return nil, types.ErrRetrieveCancelAddress
	}

	receipt, err = account.ExecActive(action.db, action.fromaddr, action.execaddr, r.Amount)
	if err != nil {
		rlog.Error("RetrieveCancel", "ExecActive err")
		return nil, err
	}

	r.Status = Retrieve_Canceled
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
