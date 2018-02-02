package retrieve

import (
	//"code.aliyun.com/chain33/chain33/account"
	"code.aliyun.com/chain33/chain33/execs/execdrivers"
	"code.aliyun.com/chain33/chain33/types"
	log "github.com/inconshreveable/log15"
)

var rlog = log.New("module", "execs.retrieve")

const minPeriod = 60
const maxFactor = 2

func init() {
	execdrivers.Register("retrieve", newRetrieve())
	execdrivers.RegisterAddress("retrieve")
}

type Retrieve struct {
	execdrivers.ExecBase
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

	if action.Ty == types.RetrievePre && action.GetPreRet() != nil {
		preRet := action.GetPreRet()
		rlog.Debug("PreRetrieve action")

		if preRet.BackoffPeriod < minPeriod {
			return nil, types.ErrRetrievePeriodLimit
		}
		if preRet.Amount < 0 {
			return nil, types.ErrRetrieveAmountLimit
		}

		return actiondb.RetrievePrepare(preRet)
	} else if action.Ty == types.RetrievePerf && action.GetPerfRet() != nil {
		perfRet := action.GetPerfRet()
		rlog.Debug("PerformRetrieve action")

		if perfRet.Factor < 0 || perfRet.Factor > maxFactor {
			return nil, types.ErrRetrieveFactorLimit
		}
		return actiondb.RetrievePerform(perfRet)
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
	}
	return "unknow"
}

func (r *Retrieve) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return r.ExecLocalCommon(tx, receipt, index)
}

func (r *Retrieve) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return r.ExecDelLocalCommon(tx, receipt, index)
}
