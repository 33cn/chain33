package executor

import (
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	rt "gitlab.33.cn/chain33/chain33/plugin/dapp/retrieve/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func SaveRetrieveInfo(info *rt.RetrieveQuery, Status int64, db dbm.KVDB) (*types.KeyValue, error) {
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
	case retrievePrepare:
		oldInfo, err := getRetrieveInfo(db, info.BackupAddress, info.DefaultAddress)
		if oldInfo == nil {
			return nil, err
		}
		info.DelayPeriod = oldInfo.DelayPeriod
		value := types.Encode(info)
		kv := &types.KeyValue{calcRetrieveKey(info.BackupAddress, info.DefaultAddress), value}
		db.Set(kv.Key, kv.Value)
		return kv, nil
	case retrievePerform:
		fallthrough
	case retrieveCancel:
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

func (c *Retrieve) ExecLocal_Backup(backup *rt.BackupRetrieve, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set := &types.LocalDBSet{}
	if receiptData.GetTy() != types.ExecOk {
		return set, nil
	}
	rlog.Debug("Retrieve ExecLocal Backup")

	info := rt.RetrieveQuery{backup.BackupAddress, backup.DefaultAddress, backup.DelayPeriod, zeroPrepareTime, zeroRemainTime, retrieveBackup}
	kv, err := SaveRetrieveInfo(&info, retrieveBackup, c.GetLocalDB())
	if err != nil {
		return set, nil
	}

	if kv != nil {
		set.KV = append(set.KV, kv)
	}

	return set, nil
}

func (c *Retrieve) ExecLocal_Prepare(pre *rt.PrepareRetrieve, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set := &types.LocalDBSet{}
	if receiptData.GetTy() != types.ExecOk {
		return set, nil
	}
	rlog.Debug("Retrieve ExecLocal Prepare")

	info := rt.RetrieveQuery{pre.BackupAddress, pre.DefaultAddress, zeroDelay, zeroPrepareTime, zeroRemainTime, retrievePrepare}
	kv, err := SaveRetrieveInfo(&info, retrievePrepare, c.GetLocalDB())
	if err != nil {
		return set, nil
	}

	if kv != nil {
		set.KV = append(set.KV, kv)
	}

	return set, nil
}

func (c *Retrieve) ExecLocal_Perform(perf *rt.PerformRetrieve, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set := &types.LocalDBSet{}
	if receiptData.GetTy() != types.ExecOk {
		return set, nil
	}
	rlog.Debug("Retrieve ExecLocal Perform")

	info := rt.RetrieveQuery{perf.BackupAddress, perf.DefaultAddress, zeroDelay, zeroPrepareTime, zeroRemainTime, retrievePerform}
	kv, err := SaveRetrieveInfo(&info, retrievePerform, c.GetLocalDB())
	if err != nil {
		return set, nil
	}

	if kv != nil {
		set.KV = append(set.KV, kv)
	}

	return set, nil
}

func (c *Retrieve) ExecLocal_Cancel(cancel *rt.CancelRetrieve, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set := &types.LocalDBSet{}
	if receiptData.GetTy() != types.ExecOk {
		return set, nil
	}
	rlog.Debug("Retrieve ExecLocal Cancel")

	info := rt.RetrieveQuery{cancel.BackupAddress, cancel.DefaultAddress, zeroDelay, zeroPrepareTime, zeroRemainTime, retrieveCancel}
	kv, err := SaveRetrieveInfo(&info, retrieveCancel, c.GetLocalDB())
	if err != nil {
		return set, nil
	}

	if kv != nil {
		set.KV = append(set.KV, kv)
	}

	return set, nil
}
