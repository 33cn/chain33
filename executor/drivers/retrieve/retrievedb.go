package retrieve

import (
	//"bytes"

	"gitlab.33.cn/chain33/chain33/account"
	//"gitlab.33.cn/chain33/chain33/common"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/types"
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

	return r
}

func (r *RetrieveDB) RelateRetrieveDB(defaultAddress string, createTime int64, delayPeriod int64) bool {
	if len(r.RetPara) >= MaxRelation {
		return false
	}
	rlog.Debug("RetrieveBackup", "RelateRetrieveDB", defaultAddress)
	para := &types.RetrievePara{defaultAddress, Retrieve_Backup, createTime, 0, delayPeriod}
	r.RetPara = append(r.RetPara, para)

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
	value := types.Encode(&r.Retrieve)
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
	coinsAccount *account.DB
	db           dbm.KVDB
	txhash       []byte
	fromaddr     string
	blocktime    int64
	height       int64
	execaddr     string
}

func NewRetrieveAcction(r *Retrieve, tx *types.Transaction) *RetrieveAction {
	hash := tx.Hash()
	fromaddr := account.PubKeyToAddress(tx.GetSignature().GetPubkey()).String()
	return &RetrieveAction{r.GetCoinsAccount(), r.GetDB(), hash, fromaddr, r.GetBlockTime(), r.GetHeight(), r.GetAddr()}
}

//wait for valuable comment
func (action *RetrieveAction) RetrieveBackup(backupRet *types.BackupRetrieve) (*types.Receipt, error) {
	var logs []*types.ReceiptLog
	var kv []*types.KeyValue
	var receipt *types.Receipt
	var r *RetrieveDB
	var newRetrieve bool = false
	if action.height >= types.ForkV5_retrieve {
		if err := account.CheckAddress(backupRet.BackupAddress); err != nil {
			rlog.Debug("retrieve checkaddress")
			return nil, err
		}
		if err := account.CheckAddress(backupRet.DefaultAddress); err != nil {
			rlog.Debug("retrieve checkaddress")
			return nil, err
		}

		if action.fromaddr != backupRet.DefaultAddress {
			rlog.Debug("RetrieveBackup", "action.fromaddr", action.fromaddr, "backupRet.DefaultAddress", backupRet.DefaultAddress)
			return nil, types.ErrRetrieveDefaultAddress
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
		r = NewRetrieveDB(backupRet.BackupAddress)
	} else {
		r = &RetrieveDB{*retrieve}
	}

	if index, related := r.CheckRelation(backupRet.DefaultAddress); !related {
		if !r.RelateRetrieveDB(backupRet.DefaultAddress, action.blocktime, backupRet.DelayPeriod) {
			rlog.Debug("RetrieveBackup", "index", index)
			return nil, types.ErrRetrieveRelateLimit
		}
	} else {
		rlog.Debug("RetrieveBackup", "repeataddr")
		return nil, types.ErrRetrieveRepeatAddress
	}

	r.Save(action.db)
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
		rlog.Debug("RetrievePrepare", "readRetrieve", err)
		return nil, err
	}
	r = &RetrieveDB{*retrieve}
	if action.fromaddr != r.BackupAddress {
		rlog.Debug("RetrievePrepare", "action.fromaddr", action.fromaddr, "r.BackupAddress", r.BackupAddress)
		return nil, types.ErrRetrievePrepareAddress
	}

	if index, related = r.CheckRelation(preRet.DefaultAddress); !related {
		rlog.Debug("RetrievePrepare", "CheckRelation", preRet.DefaultAddress)
		return nil, types.ErrRetrieveRelation
	}

	if r.RetPara[index].Status != Retrieve_Backup {
		rlog.Debug("RetrievePrepare", "Status", r.RetPara[index].Status)
		return nil, types.ErrRetrieveStatus
	}
	r.RetPara[index].PrepareTime = action.blocktime
	r.RetPara[index].Status = Retrieve_Prepared

	r.Save(action.db)
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
		rlog.Debug("RetrievePerform", "readRetrieve", perfRet.BackupAddress)
		return nil, err
	}

	r := &RetrieveDB{*retrieve}

	if index, related = r.CheckRelation(perfRet.DefaultAddress); !related {
		rlog.Debug("RetrievePerform", "CheckRelation", perfRet.DefaultAddress)
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

	acc = action.coinsAccount.LoadExecAccount(r.RetPara[index].DefaultAddress, action.execaddr)
	rlog.Debug("RetrievePerform", "acc.Balance", acc.Balance)
	if acc.Balance > 0 {
		receipt, err = action.coinsAccount.ExecTransfer(r.RetPara[index].DefaultAddress, r.BackupAddress, action.execaddr, acc.Balance)
		if err != nil {
			rlog.Debug("RetrievePerform", "ExecTransfer", err)
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
		rlog.Debug("RetrieveCancel", "CheckRelation", cancel.DefaultAddress)
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
	kv = append(kv, r.GetKVSet()...)

	receipt = &types.Receipt{types.ExecOk, kv, logs}
	return receipt, nil
}

func readRetrieve(db dbm.KVDB, address string) (*types.Retrieve, error) {
	data, err := db.Get(RetrieveKey(address))
	if err != nil {
		rlog.Debug("readRetrieve", "get", err)
		return nil, err
	}
	var retrieve types.Retrieve
	//decode
	err = types.Decode(data, &retrieve)
	if err != nil {
		rlog.Debug("readRetrieve", "decode", err)
		return nil, err
	}
	return &retrieve, nil
}
