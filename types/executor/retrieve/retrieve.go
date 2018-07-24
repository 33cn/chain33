package retrieve

import (
	"encoding/json"
	"math/rand"
	"time"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/types"
)

var name string

var rlog = log.New("module", name)

func Init() {
	name = types.ExecName("retrieve")
	// init executor type
	types.RegistorExecutor(name, &RetrieveType{})

	// init log
	//types.RegistorLog(types.TyLogDeposit, &CoinsDepositLog{})

	// init query rpc
	types.RegistorRpcType("GetRetrieveInfo", &RetrieveGetInfo{})
}

type RetrieveType struct {
	types.ExecTypeBase
}

func (r RetrieveType) ActionName(tx *types.Transaction) string {
	var action types.RetrieveAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return "unknow-err"
	}
	if action.Ty == types.RetrievePre && action.GetPreRet() != nil {
		return "prepare"
	} else if action.Ty == types.RetrievePerf && action.GetPerfRet() != nil {
		return "perform"
	} else if action.Ty == types.RetrieveBackup && action.GetBackup() != nil {
		return "backup"
	} else if action.Ty == types.RetrieveCancel && action.GetCancel() != nil {
		return "cancel"
	}
	return "unknow"
}

func (r RetrieveType) Amount(tx *types.Transaction) (int64, error) {
	return 0, nil
}

func (retrieve RetrieveType) CreateTx(action string, message json.RawMessage) (*types.Transaction, error) {
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

type RetrieveGetInfo struct {
}

func (t *RetrieveGetInfo) Input(message json.RawMessage) ([]byte, error) {
	var req types.ReqRetrieveInfo
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *RetrieveGetInfo) Output(reply interface{}) (interface{}, error) {
	return reply, nil
}

func CreateRawRetrieveBackupTx(parm *RetrieveBackupTx) (*types.Transaction, error) {
	if parm == nil {
		rlog.Error("CreateRawRetrieveBackupTx", "parm", parm)
		return nil, types.ErrInvalidParam
	}

	v := &types.BackupRetrieve{
		BackupAddress:  parm.BackupAddr,
		DefaultAddress: parm.DefaultAddr,
		DelayPeriod:    parm.DelayPeriod,
	}
	backup := &types.RetrieveAction{
		Ty:    types.RetrieveBackup,
		Value: &types.RetrieveAction_Backup{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(name),
		Payload: types.Encode(backup),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(name),
	}

	return tx, nil
}

func CreateRawRetrievePrepareTx(parm *RetrievePrepareTx) (*types.Transaction, error) {
	if parm == nil {
		rlog.Error("CreateRawRetrievePrepareTx", "parm", parm)
		return nil, types.ErrInvalidParam
	}

	v := &types.PreRetrieve{
		BackupAddress:  parm.BackupAddr,
		DefaultAddress: parm.DefaultAddr,
	}
	prepare := &types.RetrieveAction{
		Ty:    types.RetrievePre,
		Value: &types.RetrieveAction_PreRet{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(name),
		Payload: types.Encode(prepare),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(name),
	}

	return tx, nil
}

func CreateRawRetrievePerformTx(parm *RetrievePerformTx) (*types.Transaction, error) {
	if parm == nil {
		rlog.Error("CreateRawRetrievePrepareTx", "parm", parm)
		return nil, types.ErrInvalidParam
	}

	v := &types.PerformRetrieve{
		BackupAddress:  parm.BackupAddr,
		DefaultAddress: parm.DefaultAddr,
	}
	perform := &types.RetrieveAction{
		Ty:    types.RetrievePerf,
		Value: &types.RetrieveAction_PerfRet{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(name),
		Payload: types.Encode(perform),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(name),
	}

	return tx, nil
}

func CreateRawRetrieveCancelTx(parm *RetrieveCancelTx) (*types.Transaction, error) {
	if parm == nil {
		rlog.Error("CreateRawRetrieveCancelTx", "parm", parm)
		return nil, types.ErrInvalidParam
	}

	v := &types.CancelRetrieve{
		BackupAddress:  parm.BackupAddr,
		DefaultAddress: parm.DefaultAddr,
	}
	cancel := &types.RetrieveAction{
		Ty:    types.RetrieveCancel,
		Value: &types.RetrieveAction_Cancel{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(name),
		Payload: types.Encode(cancel),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(name),
	}

	return tx, nil
}
