package types

import (
	"encoding/json"
	"math/rand"
	"time"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/types"
)

var rlog = log.New("module", types.RetrieveX)
var RetrieveX = types.RetrieveX

// retrieve op
const (
	RetrievePre    = 1
	RetrievePerf   = 2
	RetrieveBackup = 3
	RetrieveCancel = 4
)

func init() {
	types.RegistorExecutor(types.RetrieveX, NewType())
}

type RetrieveType struct {
	types.ExecTypeBase
}

func NewType() *RetrieveType {
	c := &RetrieveType{}
	c.SetChild(c)
	return c
}

func (at *RetrieveType) GetPayload() types.Message {
	return &RetrieveAction{}
}

func (at *RetrieveType) GetLogMap() map[int64]*types.LogInfo {
	return map[int64]*types.LogInfo{}
}

func (at *RetrieveType) GetTypeMap() map[string]int32 {
	return map[string]int32{
		"PreRet":  RetrievePre,
		"PerfRet": RetrievePerf,
		"Backup":  RetrieveBackup,
		"Cancel":  RetrieveCancel,
	}
}

func (r *RetrieveType) ActionName(tx *types.Transaction) string {
	var action RetrieveAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return "unknown-err"
	}
	if action.Ty == RetrievePre && action.GetPreRet() != nil {
		return "prepare"
	} else if action.Ty == RetrievePerf && action.GetPerfRet() != nil {
		return "perform"
	} else if action.Ty == RetrieveBackup && action.GetBackup() != nil {
		return "backup"
	} else if action.Ty == RetrieveCancel && action.GetCancel() != nil {
		return "cancel"
	}
	return "unknown"
}

func (retrieve *RetrieveType) CreateTx(action string, message json.RawMessage) (*types.Transaction, error) {
	rlog.Debug("retrieve.CreateTx", "action", action)
	var tx *types.Transaction
	if action == "RetrieveBackup" {
		var param RetrieveBackupTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			rlog.Error("CreateTx", "Error", err)
			return nil, types.ErrInputPara
		}
		return CreateRawRetrieveBackupTx(&param)
	} else if action == "RetrievePrepare" {
		var param RetrievePrepareTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			rlog.Error("CreateTx", "Error", err)
			return nil, types.ErrInputPara
		}
		return CreateRawRetrievePrepareTx(&param)
	} else if action == "RetrievePerform" {
		var param RetrievePerformTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			rlog.Error("CreateTx", "Error", err)
			return nil, types.ErrInputPara
		}
		return CreateRawRetrievePerformTx(&param)
	} else if action == "RetrieveCancel" {
		var param RetrieveCancelTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			rlog.Error("CreateTx", "Error", err)
			return nil, types.ErrInputPara
		}
		return CreateRawRetrieveCancelTx(&param)
	} else {
		return nil, types.ErrNotSupport
	}

	return tx, nil
}

type CoinsDepositLog struct {
}

func (l CoinsDepositLog) Name() string {
	return "LogDeposit"
}

func (l CoinsDepositLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptAccountTransfer
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

func CreateRawRetrieveBackupTx(parm *RetrieveBackupTx) (*types.Transaction, error) {
	if parm == nil {
		rlog.Error("CreateRawRetrieveBackupTx", "parm", parm)
		return nil, types.ErrInvalidParam
	}

	v := &BackupRetrieve{
		BackupAddress:  parm.BackupAddr,
		DefaultAddress: parm.DefaultAddr,
		DelayPeriod:    parm.DelayPeriod,
	}
	backup := &RetrieveAction{
		Ty:    RetrieveBackup,
		Value: &RetrieveAction_Backup{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(types.ExecName(RetrieveX)),
		Payload: types.Encode(backup),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(types.ExecName(RetrieveX)),
	}

	err := tx.SetRealFee(types.MinFee)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func CreateRawRetrievePrepareTx(parm *RetrievePrepareTx) (*types.Transaction, error) {
	if parm == nil {
		rlog.Error("CreateRawRetrievePrepareTx", "parm", parm)
		return nil, types.ErrInvalidParam
	}

	v := &PreRetrieve{
		BackupAddress:  parm.BackupAddr,
		DefaultAddress: parm.DefaultAddr,
	}
	prepare := &RetrieveAction{
		Ty:    RetrievePre,
		Value: &RetrieveAction_PreRet{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(types.ExecName(RetrieveX)),
		Payload: types.Encode(prepare),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(types.ExecName(RetrieveX)),
	}

	err := tx.SetRealFee(types.MinFee)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func CreateRawRetrievePerformTx(parm *RetrievePerformTx) (*types.Transaction, error) {
	if parm == nil {
		rlog.Error("CreateRawRetrievePrepareTx", "parm", parm)
		return nil, types.ErrInvalidParam
	}

	v := &PerformRetrieve{
		BackupAddress:  parm.BackupAddr,
		DefaultAddress: parm.DefaultAddr,
	}
	perform := &RetrieveAction{
		Ty:    RetrievePerf,
		Value: &RetrieveAction_PerfRet{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(types.ExecName(RetrieveX)),
		Payload: types.Encode(perform),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(types.ExecName(RetrieveX)),
	}

	err := tx.SetRealFee(types.MinFee)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func CreateRawRetrieveCancelTx(parm *RetrieveCancelTx) (*types.Transaction, error) {
	if parm == nil {
		rlog.Error("CreateRawRetrieveCancelTx", "parm", parm)
		return nil, types.ErrInvalidParam
	}

	v := &CancelRetrieve{
		BackupAddress:  parm.BackupAddr,
		DefaultAddress: parm.DefaultAddr,
	}
	cancel := &RetrieveAction{
		Ty:    RetrieveCancel,
		Value: &RetrieveAction_Cancel{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(types.ExecName(RetrieveX)),
		Payload: types.Encode(cancel),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(types.ExecName(RetrieveX)),
	}
	err := tx.SetRealFee(types.MinFee)
	if err != nil {
		return nil, err
	}
	return tx, nil
}
