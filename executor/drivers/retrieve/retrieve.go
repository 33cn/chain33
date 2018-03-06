package retrieve

import (
	//"code.aliyun.com/chain33/chain33/account"
	"code.aliyun.com/chain33/chain33/executor/drivers"
	"code.aliyun.com/chain33/chain33/types"
	log "github.com/inconshreveable/log15"
)

var rlog = log.New("module", "execs.retrieve")

var minPeriod int64 = 60

//const maxTimeWeight = 2

func init() {
	drivers.Register("retrieve", newRetrieve())
	drivers.RegisterAddress("retrieve")
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

	actiondb := NewRetrieveAcction(r.GetDB(), tx, r.GetAddr(), r.GetBlockTime(), r.GetHeight())
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
	return r.ExecLocalCommon(tx, receipt, index)
}

func (r *Retrieve) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return r.ExecDelLocalCommon(tx, receipt, index)
}
