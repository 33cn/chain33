package executor

import (
	"fmt"

	log "github.com/inconshreveable/log15"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/retrieve/types"
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	minPeriod int64 = 60
	rlog            = log.New("module", "execs.retrieve")
)

var (
	zeroDelay       int64
	zeroPrepareTime int64
	zeroRemainTime  int64
)

var driverName = "retrieve"

func init() {
	ety := types.LoadExecutorType(driverName)
	ety.InitFuncList(types.ListMethod(&Retrieve{}))
}

//const maxTimeWeight = 2
func Init(name string) {
	drivers.Register(GetName(), newRetrieve, 0)
}

func GetName() string {
	return newRetrieve().GetName()
}

type Retrieve struct {
	drivers.DriverBase
}

func newRetrieve() drivers.Driver {
	r := &Retrieve{}
	r.SetChild(r)
	return r
}

func (r *Retrieve) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	var action ty.RetrieveAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return nil, err
	}

	rlog.Debug("Exec retrieve tx=", "tx=", action)

	actiondb := NewRetrieveAcction(r, tx)
	if action.Ty == ty.RetrieveBackup && action.GetBackup() != nil {
		backupRet := action.GetBackup()
		if backupRet.DelayPeriod < minPeriod {
			return nil, types.ErrRetrievePeriodLimit
		}
		rlog.Debug("RetrieveBackup action")
		return actiondb.RetrieveBackup(backupRet)
	} else if action.Ty == ty.RetrievePre && action.GetPreRet() != nil {
		preRet := action.GetPreRet()
		rlog.Debug("PreRetrieve action")
		return actiondb.RetrievePrepare(preRet)
	} else if action.Ty == ty.RetrievePerf && action.GetPerfRet() != nil {
		perfRet := action.GetPerfRet()
		rlog.Debug("PerformRetrieve action")
		return actiondb.RetrievePerform(perfRet)
	} else if action.Ty == ty.RetrieveCancel && action.GetCancel() != nil {
		cancel := action.GetCancel()
		rlog.Debug("RetrieveCancel action")
		return actiondb.RetrieveCancel(cancel)
	}
	//return error
	return nil, types.ErrActionNotSupport
}

func (r *Retrieve) GetDriverName() string {
	return driverName
}

func (r *Retrieve) GetActionName(tx *types.Transaction) string {
	var action ty.RetrieveAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return "unknow"
	}
	if action.Ty == ty.RetrievePre && action.GetPreRet() != nil {
		return "retrieve prepare"
	} else if action.Ty == ty.RetrievePerf && action.GetPerfRet() != nil {
		return "retrieve perform"
	} else if action.Ty == ty.RetrieveBackup && action.GetBackup() != nil {
		return "retrieve backup"
	} else if action.Ty == ty.RetrieveCancel && action.GetCancel() != nil {
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
	rlog.Debug("Retrieve ExecLocal")
	var action ty.RetrieveAction
	err = types.Decode(tx.GetPayload(), &action)
	if err != nil {
		panic(err)
	}

	var kv *types.KeyValue
	if action.Ty == ty.RetrieveBackup && action.GetBackup() != nil {
		backupRet := action.GetBackup()
		info := ty.RetrieveQuery{backupRet.BackupAddress, backupRet.DefaultAddress, backupRet.DelayPeriod, zeroPrepareTime, zeroRemainTime, retrieveBackup}
		kv, err = SaveRetrieveInfo(&info, retrieveBackup, r.GetLocalDB())
	} else if action.Ty == ty.RetrievePre && action.GetPreRet() != nil {
		preRet := action.GetPreRet()
		info := ty.RetrieveQuery{preRet.BackupAddress, preRet.DefaultAddress, zeroDelay, r.GetBlockTime(), zeroRemainTime, retrievePrepared}
		kv, err = SaveRetrieveInfo(&info, retrievePrepared, r.GetLocalDB())
	} else if action.Ty == ty.RetrievePerf && action.GetPerfRet() != nil {
		perfRet := action.GetPerfRet()
		info := ty.RetrieveQuery{perfRet.BackupAddress, perfRet.DefaultAddress, zeroDelay, zeroPrepareTime, zeroRemainTime, retrievePerformed}
		kv, err = SaveRetrieveInfo(&info, retrievePerformed, r.GetLocalDB())
	} else if action.Ty == ty.RetrieveCancel && action.GetCancel() != nil {
		cancel := action.GetCancel()
		info := ty.RetrieveQuery{cancel.BackupAddress, cancel.DefaultAddress, zeroDelay, zeroPrepareTime, zeroRemainTime, retrieveCanceled}
		kv, err = SaveRetrieveInfo(&info, retrieveCanceled, r.GetLocalDB())
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
	var action ty.RetrieveAction
	err = types.Decode(tx.GetPayload(), &action)
	if err != nil {
		panic(err)
	}
	var kv *types.KeyValue
	if action.Ty == ty.RetrieveBackup && action.GetBackup() != nil {
		backupRet := action.GetBackup()
		info := ty.RetrieveQuery{backupRet.BackupAddress, backupRet.DefaultAddress, backupRet.DelayPeriod, zeroPrepareTime, zeroRemainTime, retrieveBackup}
		kv, err = DelRetrieveInfo(&info, retrieveBackup, r.GetLocalDB())
	} else if action.Ty == ty.RetrievePre && action.GetPreRet() != nil {
		preRet := action.GetPreRet()
		info := ty.RetrieveQuery{preRet.BackupAddress, preRet.DefaultAddress, zeroDelay, r.GetBlockTime(), zeroRemainTime, retrievePrepared}
		kv, err = DelRetrieveInfo(&info, retrievePrepared, r.GetLocalDB())
	} else if action.Ty == ty.RetrievePerf && action.GetPerfRet() != nil {
		perfRet := action.GetPerfRet()
		info := ty.RetrieveQuery{perfRet.BackupAddress, perfRet.DefaultAddress, zeroDelay, zeroPrepareTime, zeroRemainTime, retrievePerformed}
		kv, err = DelRetrieveInfo(&info, retrievePerformed, r.GetLocalDB())
	} else if action.Ty == ty.RetrieveCancel && action.GetCancel() != nil {
		cancel := action.GetCancel()
		info := ty.RetrieveQuery{cancel.BackupAddress, cancel.DefaultAddress, zeroDelay, zeroPrepareTime, zeroRemainTime, retrieveCanceled}
		kv, err = DelRetrieveInfo(&info, retrieveCanceled, r.GetLocalDB())
	}
	if err != nil {
		return set, nil
	}
	if kv != nil {
		set.KV = append(set.KV, kv)
	}
	return set, nil
}

func SaveRetrieveInfo(info *ty.RetrieveQuery, Status int64, db dbm.KVDB) (*types.KeyValue, error) {
	rlog.Debug("Retrieve SaveRetrieveInfo", "backupaddr", info.BackupAddress, "defaddr", info.DefaultAddress)
	switch Status {
	case retrieveBackup:
		oldInfo, err := getRetrieveInfo(db, info.BackupAddress, info.DefaultAddress)
		if oldInfo != nil && oldInfo.Status == retrieveBackup {
			return nil, err
		}
		value := types.Encode(info)
		kv := &types.KeyValue{calcRetrieveKey(info.BackupAddress, info.DefaultAddress), value}
		db.Set(kv.Key, kv.Value)
		return kv, nil
	case retrievePrepared:
		oldInfo, err := getRetrieveInfo(db, info.BackupAddress, info.DefaultAddress)
		if oldInfo == nil {
			return nil, err
		}
		info.DelayPeriod = oldInfo.DelayPeriod
		value := types.Encode(info)
		kv := &types.KeyValue{calcRetrieveKey(info.BackupAddress, info.DefaultAddress), value}
		db.Set(kv.Key, kv.Value)
		return kv, nil
	case retrievePerformed:
		fallthrough
	case retrieveCanceled:
		oldInfo, err := getRetrieveInfo(db, info.BackupAddress, info.DefaultAddress)
		if oldInfo == nil {
			return nil, err
		}
		info.DelayPeriod = oldInfo.DelayPeriod
		info.PrepareTime = oldInfo.PrepareTime
		value := types.Encode(info)
		kv := &types.KeyValue{calcRetrieveKey(info.BackupAddress, info.DefaultAddress), value}
		db.Set(kv.Key, kv.Value)
		return kv, nil
	default:
		return nil, nil
	}

}

func DelRetrieveInfo(info *ty.RetrieveQuery, Status int64, db dbm.KVDB) (*types.KeyValue, error) {
	switch Status {
	case retrieveBackup:
		kv := &types.KeyValue{calcRetrieveKey(info.BackupAddress, info.DefaultAddress), nil}
		db.Set(kv.Key, kv.Value)
		return kv, nil
	case retrievePrepared:
		oldInfo, err := getRetrieveInfo(db, info.BackupAddress, info.DefaultAddress)
		if oldInfo == nil {
			return nil, err
		}
		info.DelayPeriod = oldInfo.DelayPeriod
		info.Status = retrieveBackup
		info.PrepareTime = 0
		value := types.Encode(info)
		kv := &types.KeyValue{calcRetrieveKey(info.BackupAddress, info.DefaultAddress), value}
		db.Set(kv.Key, kv.Value)
		return kv, nil
	case retrievePerformed:
		fallthrough
	case retrieveCanceled:
		oldInfo, err := getRetrieveInfo(db, info.BackupAddress, info.DefaultAddress)
		if oldInfo == nil {
			return nil, err
		}
		info.DelayPeriod = oldInfo.DelayPeriod
		info.Status = retrievePrepared
		info.PrepareTime = oldInfo.PrepareTime
		value := types.Encode(info)
		kv := &types.KeyValue{calcRetrieveKey(info.BackupAddress, info.DefaultAddress), value}
		db.Set(kv.Key, kv.Value)
		return kv, nil
	default:
		return nil, nil
	}
}

func (r *Retrieve) Query(funcName string, params []byte) (types.Message, error) {
	if funcName == "GetRetrieveInfo" {
		var req ty.ReqRetrieveInfo
		err := types.Decode(params, &req)
		if err != nil {
			return nil, err
		}
		rlog.Debug("Retrieve Query", "backupaddr", req.BackupAddress, "defaddr", req.DefaultAddress)
		info, err := getRetrieveInfo(r.GetLocalDB(), req.BackupAddress, req.DefaultAddress)
		if info == nil {
			return nil, err
		}
		if info.Status == retrievePrepared {
			info.RemainTime = info.DelayPeriod - (r.GetBlockTime() - info.PrepareTime)
			if info.RemainTime < 0 {
				info.RemainTime = 0
			}
		}
		return info, nil
	}
	return nil, types.ErrActionNotSupport
}

func calcRetrieveKey(backupAddr string, defaultAddr string) []byte {
	key := fmt.Sprintf("Retrieve-backup:%s:%s", backupAddr, defaultAddr)
	return []byte(key)
}

func getRetrieveInfo(db dbm.KVDB, backupAddr string, defaultAddr string) (*ty.RetrieveQuery, error) {
	info := ty.RetrieveQuery{}
	retInfo, err := db.Get(calcRetrieveKey(backupAddr, defaultAddr))
	if err != nil {
		return nil, err
	}

	err = types.Decode(retInfo, &info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}
