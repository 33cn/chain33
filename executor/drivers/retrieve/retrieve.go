package retrieve

import (
	"fmt"

	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	minPeriod int64 = 60
	rlog            = log.New("module", "execs.retrieve")
)

var (
	zeroDelay       int64 = 0
	zeroPrepareTime int64 = 0
	zeroRemainTime  int64 = 0
)

//const maxTimeWeight = 2
func init() {
	h := newRetrieve()
	drivers.Register(h.GetName(), h, 0)
}

type Retrieve struct {
	drivers.DriverBase
}

func newRetrieve() *Retrieve {
	r := &Retrieve{}
	r.SetChild(r)
	return r
}

func (r *Retrieve) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	var action types.RetrieveAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return nil, err
	}

	rlog.Debug("Exec retrieve tx=", "tx=", action)

	actiondb := NewRetrieveAcction(r, tx)
	if action.Ty == types.RetrieveBackup && action.GetBackup() != nil {
		backupRet := action.GetBackup()
		if backupRet.DelayPeriod < minPeriod {
			return nil, types.ErrRetrievePeriodLimit
		}
		rlog.Debug("RetrieveBackup action")
		return actiondb.RetrieveBackup(backupRet)
	} else if action.Ty == types.RetrievePre && action.GetPreRet() != nil {
		preRet := action.GetPreRet()
		rlog.Debug("PreRetrieve action")
		return actiondb.RetrievePrepare(preRet)
	} else if action.Ty == types.RetrievePerf && action.GetPerfRet() != nil {
		perfRet := action.GetPerfRet()
		rlog.Debug("PerformRetrieve action")
		return actiondb.RetrievePerform(perfRet)
	} else if action.Ty == types.RetrieveCancel && action.GetCancel() != nil {
		cancel := action.GetCancel()
		rlog.Debug("RetrieveCancel action")
		return actiondb.RetrieveCancel(cancel)
	}
	//return error
	return nil, types.ErrActionNotSupport
}

func (r *Retrieve) GetName() string {
	return "retrieve"
}

func (r *Retrieve) GetActionName(tx *types.Transaction) string {
	var action types.RetrieveAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return "unknow"
	}
	if action.Ty == types.RetrievePre && action.GetPreRet() != nil {
		return "retrieve prepare"
	} else if action.Ty == types.RetrievePerf && action.GetPerfRet() != nil {
		return "retrieve perform"
	} else if action.Ty == types.RetrieveBackup && action.GetBackup != nil {
		return "retrieve backup"
	} else if action.Ty == types.RetrieveCancel && action.GetCancel != nil {
		return "retrieve cancel"
	}
	return "unknow"
}

func (r *Retrieve) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := r.DriverBase.ExecLocal(tx, receipt, index)
	if err != nil {
		return nil, err
	}
	if receipt.GetTy() != types.ExecOk {
		return set, nil
	}

	var action types.RetrieveAction
	err = types.Decode(tx.GetPayload(), &action)
	if err != nil {
		panic(err)
	}

	var kv *types.KeyValue
	if action.Ty == types.RetrieveBackup && action.GetBackup() != nil {
		backupRet := action.GetBackup()
		info := types.RetrieveQuery{backupRet.BackupAddress, backupRet.DefaultAddress, backupRet.DelayPeriod, zeroPrepareTime, zeroRemainTime, Retrieve_Backup}
		kv, err = SaveRetrieveInfo(&info, Retrieve_Backup, r.GetDB())
	} else if action.Ty == types.RetrievePre && action.GetPreRet() != nil {
		preRet := action.GetPreRet()
		info := types.RetrieveQuery{preRet.BackupAddress, preRet.DefaultAddress, zeroDelay, r.GetBlockTime(), zeroRemainTime, Retrieve_Prepared}
		kv, err = SaveRetrieveInfo(&info, Retrieve_Prepared, r.GetDB())
	} else if action.Ty == types.RetrievePerf && action.GetPerfRet() != nil {
		perfRet := action.GetPerfRet()
		info := types.RetrieveQuery{perfRet.BackupAddress, perfRet.DefaultAddress, zeroDelay, zeroPrepareTime, zeroRemainTime, Retrieve_Performed}
		kv, err = SaveRetrieveInfo(&info, Retrieve_Performed, r.GetDB())
	} else if action.Ty == types.RetrieveCancel && action.GetCancel() != nil {
		cancel := action.GetCancel()
		info := types.RetrieveQuery{cancel.BackupAddress, cancel.DefaultAddress, zeroDelay, zeroPrepareTime, zeroRemainTime, Retrieve_Canceled}
		kv, err = SaveRetrieveInfo(&info, Retrieve_Canceled, r.GetDB())
	}

	if err != nil {
		return set, nil
	}
	if kv != nil {
		set.KV = append(set.KV, kv)
	}

	return set, nil
}

func (r *Retrieve) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := r.DriverBase.ExecDelLocal(tx, receipt, index)
	if err != nil {
		return nil, err
	}
	if receipt.GetTy() != types.ExecOk {
		return set, nil
	}
	//执行成功
	var action types.RetrieveAction
	err = types.Decode(tx.GetPayload(), &action)
	if err != nil {
		panic(err)
	}
	var kv *types.KeyValue
	if action.Ty == types.RetrieveBackup && action.GetBackup() != nil {
		backupRet := action.GetBackup()
		info := types.RetrieveQuery{backupRet.BackupAddress, backupRet.DefaultAddress, backupRet.DelayPeriod, zeroPrepareTime, zeroRemainTime, Retrieve_Backup}
		kv, err = DelRetrieveInfo(&info, Retrieve_Backup, r.GetDB())
	} else if action.Ty == types.RetrievePre && action.GetPreRet() != nil {
		preRet := action.GetPreRet()
		info := types.RetrieveQuery{preRet.BackupAddress, preRet.DefaultAddress, zeroDelay, r.GetBlockTime(), zeroRemainTime, Retrieve_Prepared}
		kv, err = DelRetrieveInfo(&info, Retrieve_Prepared, r.GetDB())
	} else if action.Ty == types.RetrievePerf && action.GetPerfRet() != nil {
		perfRet := action.GetPerfRet()
		info := types.RetrieveQuery{perfRet.BackupAddress, perfRet.DefaultAddress, zeroDelay, zeroPrepareTime, zeroRemainTime, Retrieve_Performed}
		kv, err = DelRetrieveInfo(&info, Retrieve_Performed, r.GetDB())
	} else if action.Ty == types.RetrieveCancel && action.GetCancel() != nil {
		cancel := action.GetCancel()
		info := types.RetrieveQuery{cancel.BackupAddress, cancel.DefaultAddress, zeroDelay, zeroPrepareTime, zeroRemainTime, Retrieve_Canceled}
		kv, err = DelRetrieveInfo(&info, Retrieve_Canceled, r.GetDB())
	}
	if err != nil {
		return set, nil
	}
	if kv != nil {
		set.KV = append(set.KV, kv)
	}
	return set, nil
}

func SaveRetrieveInfo(info *types.RetrieveQuery, Status int64, db dbm.KVDB) (*types.KeyValue, error) {
	switch Status {
	case Retrieve_Backup:
		//TRY TO USE DB
		oldInfo, err := getRetrieveInfo(db, info.BackupAddress, info.DefaultAddress)
		if oldInfo != nil && oldInfo.Status == Retrieve_Backup {
			return nil, err
		}
		value := types.Encode(info)
		kv := &types.KeyValue{calcRetrieveKey(info.BackupAddress, info.DefaultAddress), value}
		return kv, nil
	case Retrieve_Prepared:
		oldInfo, err := getRetrieveInfo(db, info.BackupAddress, info.DefaultAddress)
		if oldInfo == nil {
			return nil, err
		}
		info.DelayPeriod = oldInfo.DelayPeriod
		value := types.Encode(info)
		kv := &types.KeyValue{calcRetrieveKey(info.BackupAddress, info.DefaultAddress), value}
		return kv, nil
	case Retrieve_Performed:
		fallthrough
	case Retrieve_Canceled:
		oldInfo, err := getRetrieveInfo(db, info.BackupAddress, info.DefaultAddress)
		if oldInfo == nil {
			return nil, err
		}
		info.DelayPeriod = oldInfo.DelayPeriod
		info.PrepareTime = oldInfo.PrepareTime
		value := types.Encode(info)
		kv := &types.KeyValue{calcRetrieveKey(info.BackupAddress, info.DefaultAddress), value}
		return kv, nil
	default:
		return nil, nil
	}

}

func DelRetrieveInfo(info *types.RetrieveQuery, Status int64, db dbm.KVDB) (*types.KeyValue, error) {
	switch Status {
	case Retrieve_Backup:
		kv := &types.KeyValue{calcRetrieveKey(info.BackupAddress, info.DefaultAddress), nil}
		return kv, nil
	case Retrieve_Prepared:
		oldInfo, err := getRetrieveInfo(db, info.BackupAddress, info.DefaultAddress)
		if oldInfo == nil {
			return nil, err
		}
		info.DelayPeriod = oldInfo.DelayPeriod
		info.Status = Retrieve_Backup
		info.PrepareTime = 0
		value := types.Encode(info)
		kv := &types.KeyValue{calcRetrieveKey(info.BackupAddress, info.DefaultAddress), value}
		return kv, nil
	case Retrieve_Performed:
		fallthrough
	case Retrieve_Canceled:
		oldInfo, err := getRetrieveInfo(db, info.BackupAddress, info.DefaultAddress)
		if oldInfo == nil {
			return nil, err
		}
		info.DelayPeriod = oldInfo.DelayPeriod
		info.Status = Retrieve_Prepared
		info.PrepareTime = oldInfo.PrepareTime
		value := types.Encode(info)
		kv := &types.KeyValue{calcRetrieveKey(info.BackupAddress, info.DefaultAddress), value}
		return kv, nil
	default:
		return nil, nil
	}
}

func (r *Retrieve) Query(funcName string, params []byte) (types.Message, error) {
	if funcName == "GetRetrieveInfo" {
		rlog.Debug("Query action")
		var req types.ReqRetrieveInfo
		err := types.Decode(params, &req)
		if err != nil {
			return nil, err
		}
		info, err := getRetrieveInfo(r.GetDB(), req.BackupAddress, req.DefaultAddress)
		if info == nil {
			return nil, err
		}
		return info, nil
	}
	return nil, types.ErrActionNotSupport
}

func calcRetrieveKey(backupAddr string, defaultAddr string) []byte {
	key := fmt.Sprintf("Retrieve-backup:%s:%s", backupAddr, defaultAddr)
	return []byte(key)
}

func getRetrieveInfo(db dbm.KVDB, backupAddr string, defaultAddr string) (*types.RetrieveQuery, error) {
	info := types.RetrieveQuery{}
	retInfo, err := db.Get(calcRetrieveKey(backupAddr, defaultAddr))
	if err != nil && err != types.ErrNotFound {
		return nil, err
	}

	err = types.Decode(retInfo, &info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}
