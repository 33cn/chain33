package executor

import (
	//"bytes"

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common/address"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/types"

	//log "github.com/inconshreveable/log15"
	rt "gitlab.33.cn/chain33/chain33/plugin/dapp/retrieve/types"
	"gitlab.33.cn/chain33/chain33/system/dapp"
)

const (
	retrieveBackup  = 1
	retrievePrepare = 2
	retrievePerform = 3
	retrieveCancel  = 4
)

const MaxRelation = 10

type DB struct {
	rt.Retrieve
}

func NewDB(backupaddress string) *DB {
	r := &DB{}
	r.BackupAddress = backupaddress

	return r
}

func (r *DB) RelateDB(defaultAddress string, createTime int64, delayPeriod int64) bool {
	if len(r.RetPara) >= MaxRelation {
		return false
	}
	rlog.Debug("RetrieveBackup", "RelateDB", defaultAddress)
	para := &rt.RetrievePara{defaultAddress, retrieveBackup, createTime, 0, delayPeriod}
	r.RetPara = append(r.RetPara, para)

	return true
}

func (r *DB) UnRelateDB(index int) bool {
	r.RetPara = append(r.RetPara[:index], r.RetPara[index+1:]...)
	return true
}

func (r *DB) CheckRelation(defaultAddress string) (int, bool) {
	for i := 0; i < len(r.RetPara); i++ {
		if r.RetPara[i].DefaultAddress == defaultAddress {
			return i, true
		}
	}
	return MaxRelation, false
}

func (r *DB) GetKVSet() (kvset []*types.KeyValue) {
	value := types.Encode(&r.Retrieve)
	kvset = append(kvset, &types.KeyValue{Key(r.BackupAddress), value})
	return kvset
}

func (r *DB) Save(db dbm.KV) {
	set := r.GetKVSet()
	for i := 0; i < len(set); i++ {
		db.Set(set[i].GetKey(), set[i].Value)
	}
}

func Key(address string) (key []byte) {
	key = append(key, []byte("mavl-retrieve-")...)
	key = append(key, address...)
	return key
}

type Action struct {
	coinsAccount *account.DB
	db           dbm.KV
	txhash       []byte
	fromaddr     string
	blocktime    int64
	height       int64
	execaddr     string
}

func NewRetrieveAcction(r *Retrieve, tx *types.Transaction) *Action {
	hash := tx.Hash()
	fromaddr := tx.From()
	return &Action{r.GetCoinsAccount(), r.GetStateDB(), hash, fromaddr,
		r.GetBlockTime(), r.GetHeight(), dapp.ExecAddress(string(tx.Execer))}
}

//wait for valuable comment
func (action *Action) RetrieveBackup(backupRet *rt.BackupRetrieve) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	var receipt *types.Receipt
	var r *DB
	var newRetrieve = false
	if types.IsDappFork(action.height, rt.RetrieveX, "ForkRetrive") {
		if err := address.CheckAddress(backupRet.BackupAddress); err != nil {
			rlog.Debug("retrieve checkaddress")
			return nil, err
		}
		if err := address.CheckAddress(backupRet.DefaultAddress); err != nil {
			rlog.Debug("retrieve checkaddress")
			return nil, err
		}

		if action.fromaddr != backupRet.DefaultAddress {
			rlog.Debug("RetrieveBackup", "action.fromaddr", action.fromaddr, "backupRet.DefaultAddress", backupRet.DefaultAddress)
			return nil, rt.ErrRetrieveDefaultAddress
		}
	}
	//用备份地址检索，如果没有，就建立新的，然后检查并处理关联
	retrieve, err := readRetrieve(action.db, backupRet.BackupAddress)
	if err != nil && err != types.ErrNotFound {
		rlog.Error("RetrieveBackup", "readRetrieve", err)
		return nil, err
	} else if err == types.ErrNotFound {
		newRetrieve = true
		rlog.Debug("RetrieveBackup", "newAddress", backupRet.BackupAddress)
	}

	if newRetrieve {
		r = NewDB(backupRet.BackupAddress)
	} else {
		r = &DB{*retrieve}
	}

	if index, related := r.CheckRelation(backupRet.DefaultAddress); !related {
		if !r.RelateDB(backupRet.DefaultAddress, action.blocktime, backupRet.DelayPeriod) {
			rlog.Debug("RetrieveBackup", "index", index)
			return nil, rt.ErrRetrieveRelateLimit
		}
	} else {
		rlog.Debug("RetrieveBackup", "repeataddr")
		return nil, rt.ErrRetrieveRepeatAddress
	}

	r.Save(action.db)
	kv = append(kv, r.GetKVSet()...)

	receipt = &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil
}

func (action *Action) RetrievePrepare(preRet *rt.PrepareRetrieve) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	var receipt *types.Receipt
	var r *DB
	var index int
	var related bool

	retrieve, err := readRetrieve(action.db, preRet.BackupAddress)
	if err != nil {
		rlog.Debug("RetrievePrepare", "readRetrieve", err)
		return nil, err
	}
	r = &DB{*retrieve}
	if action.fromaddr != r.BackupAddress {
		rlog.Debug("RetrievePrepare", "action.fromaddr", action.fromaddr, "r.BackupAddress", r.BackupAddress)
		return nil, rt.ErrRetrievePrepareAddress
	}

	if index, related = r.CheckRelation(preRet.DefaultAddress); !related {
		rlog.Debug("RetrievePrepare", "CheckRelation", preRet.DefaultAddress)
		return nil, rt.ErrRetrieveRelation
	}

	if r.RetPara[index].Status != retrieveBackup {
		rlog.Debug("RetrievePrepare", "Status", r.RetPara[index].Status)
		return nil, rt.ErrRetrieveStatus
	}
	r.RetPara[index].PrepareTime = action.blocktime
	r.RetPara[index].Status = retrievePrepare

	r.Save(action.db)
	kv = append(kv, r.GetKVSet()...)

	receipt = &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil
}

func (action *Action) RetrievePerform(perfRet *rt.PerformRetrieve) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	var receipt *types.Receipt
	var index int
	var related bool
	var acc *types.Account

	retrieve, err := readRetrieve(action.db, perfRet.BackupAddress)
	if err != nil {
		rlog.Debug("RetrievePerform", "readRetrieve", perfRet.BackupAddress)
		return nil, err
	}

	r := &DB{*retrieve}

	if index, related = r.CheckRelation(perfRet.DefaultAddress); !related {
		rlog.Debug("RetrievePerform", "CheckRelation", perfRet.DefaultAddress)
		return nil, rt.ErrRetrieveRelation
	}

	if r.BackupAddress != action.fromaddr {
		rlog.Debug("RetrievePerform", "BackupAddress", r.BackupAddress, "action.fromaddr", action.fromaddr)
		return nil, rt.ErrRetrievePerformAddress
	}

	if r.RetPara[index].Status != retrievePrepare {
		rlog.Debug("RetrievePerform", "Status", r.RetPara[index].Status)
		return nil, rt.ErrRetrieveStatus
	}
	if action.blocktime-r.RetPara[index].PrepareTime < r.RetPara[index].DelayPeriod {
		rlog.Debug("RetrievePerform", "ErrRetrievePeriodLimit")
		return nil, rt.ErrRetrievePeriodLimit
	}

	acc = action.coinsAccount.LoadExecAccount(r.RetPara[index].DefaultAddress, action.execaddr)
	rlog.Debug("RetrievePerform", "acc.Balance", acc.Balance)
	if acc.Balance > 0 {
		receipt, err = action.coinsAccount.ExecTransfer(r.RetPara[index].DefaultAddress, r.BackupAddress, action.execaddr, acc.Balance)
		if err != nil {
			rlog.Debug("RetrievePerform", "ExecTransfer", err)
			return nil, err
		}
	} else {
		return nil, rt.ErrRetrieveNoBalance
	}

	//r.RetPara[index].Status = Retrieve_Performed
	//remove the relation
	r.UnRelateDB(index)
	r.Save(action.db)
	logs = append(logs, receipt.Logs...)
	kv = append(kv, receipt.KV...)
	kv = append(kv, r.GetKVSet()...)

	receipt = &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil
}

func (action *Action) RetrieveCancel(cancel *rt.CancelRetrieve) (*types.Receipt, error) {
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
	r := &DB{*retrieve}

	if index, related = r.CheckRelation(cancel.DefaultAddress); !related {
		rlog.Debug("RetrieveCancel", "CheckRelation", cancel.DefaultAddress)
		return nil, rt.ErrRetrieveRelation
	}

	if action.fromaddr != r.RetPara[index].DefaultAddress {
		rlog.Debug("RetrieveCancel", "action.fromaddr", action.fromaddr, "DefaultAddress", r.RetPara[index].DefaultAddress)
		return nil, rt.ErrRetrieveCancelAddress
	}

	if r.RetPara[index].Status != retrievePrepare {
		rlog.Debug("RetrieveCancel", "Status", r.RetPara[index].Status)
		return nil, rt.ErrRetrieveStatus
	}

	//r.RetPara[index].Status = Retrieve_Canceled
	//remove the relation
	r.UnRelateDB(index)
	r.Save(action.db)
	kv = append(kv, r.GetKVSet()...)

	receipt = &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil
}

func readRetrieve(db dbm.KV, address string) (*rt.Retrieve, error) {
	data, err := db.Get(Key(address))
	if err != nil {
		rlog.Debug("readRetrieve", "get", err)
		return nil, err
	}
	var retrieve rt.Retrieve
	//decode
	err = types.Decode(data, &retrieve)
	if err != nil {
		rlog.Debug("readRetrieve", "decode", err)
		return nil, err
	}
	return &retrieve, nil
}
