package hashlock

import (
	"encoding/json"
	"math/rand"
	"time"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/types"
)

var name string

var hlog = log.New("module", name)

func Init() {
	name = types.ExecName("hashlock")
	// init executor type
	types.RegistorExecutor(name, &HashlockType{})

	// init log
	//types.RegistorLog(types.TyLogDeposit, &CoinsDepositLog{})

	// init query rpc
	//types.RegistorRpcType("q2", &CoinsGetTxsByAddr{})
}

type HashlockType struct {
	types.ExecTypeBase
}

func (hashlock HashlockType) ActionName(tx *types.Transaction) string {
	var action types.HashlockAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return "unknow-err"
	}
	if action.Ty == types.HashlockActionLock && action.GetHlock() != nil {
		return "lock"
	} else if action.Ty == types.HashlockActionUnlock && action.GetHunlock() != nil {
		return "unlock"
	} else if action.Ty == types.HashlockActionSend && action.GetHsend() != nil {
		return "send"
	}
	return "unknow"
}

func (t HashlockType) Amount(tx *types.Transaction) (int64, error) {
	return 0, nil
}

// TODO 暂时不修改实现， 先完成结构的重构
func (hashlock HashlockType) CreateTx(action string, message json.RawMessage) (*types.Transaction, error) {
	hlog.Debug("hashlock.CreateTx", "action", action)
	var tx *types.Transaction
	if action == "HashlockLock" {
		var param HashlockLockTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			hlog.Error("CreateTx", "Error", err)
			return nil, types.ErrInputPara
		}
		return CreateRawHashlockLockTx(&param)
	} else if action == "HashlockUnlock" {
		var param HashlockUnlockTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			hlog.Error("CreateTx", "Error", err)
			return nil, types.ErrInputPara
		}
		return CreateRawHashlockUnlockTx(&param)
	} else if action == "HashlockSend" {
		var param HashlockSendTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			hlog.Error("CreateTx", "Error", err)
			return nil, types.ErrInputPara
		}
		return CreateRawHashlockSendTx(&param)
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

type CoinsGetTxsByAddr struct {
}

func (t *CoinsGetTxsByAddr) Input(message json.RawMessage) ([]byte, error) {
	var req types.ReqAddr
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *CoinsGetTxsByAddr) Output(reply interface{}) (interface{}, error) {
	return reply, nil
}

func CreateRawHashlockLockTx(parm *HashlockLockTx) (*types.Transaction, error) {
	if parm == nil {
		hlog.Error("CreateRawHashlockLockTx", "parm", parm)
		return nil, types.ErrInvalidParam
	}

	v := &types.HashlockLock{
		Amount:        parm.Amount,
		Time:          parm.Time,
		Hash:          common.Sha256([]byte(parm.Secret)),
		ToAddress:     parm.ToAddr,
		ReturnAddress: parm.ReturnAddr,
	}
	lock := &types.HashlockAction{
		Ty:    types.HashlockActionLock,
		Value: &types.HashlockAction_Hlock{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(name),
		Payload: types.Encode(lock),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(name),
	}

	err := tx.SetRealFee(types.MinFee)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func CreateRawHashlockUnlockTx(parm *HashlockUnlockTx) (*types.Transaction, error) {
	if parm == nil {
		hlog.Error("CreateRawHashlockUnlockTx", "parm", parm)
		return nil, types.ErrInvalidParam
	}

	v := &types.HashlockUnlock{
		Secret: []byte(parm.Secret),
	}
	unlock := &types.HashlockAction{
		Ty:    types.HashlockActionUnlock,
		Value: &types.HashlockAction_Hunlock{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(name),
		Payload: types.Encode(unlock),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(name),
	}

	err := tx.SetRealFee(types.MinFee)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func CreateRawHashlockSendTx(parm *HashlockSendTx) (*types.Transaction, error) {
	if parm == nil {
		hlog.Error("CreateRawHashlockSendTx", "parm", parm)
		return nil, types.ErrInvalidParam
	}

	v := &types.HashlockSend{
		Secret: []byte(parm.Secret),
	}
	send := &types.HashlockAction{
		Ty:    types.HashlockActionSend,
		Value: &types.HashlockAction_Hsend{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(name),
		Payload: types.Encode(send),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(name),
	}

	err := tx.SetRealFee(types.MinFee)
	if err != nil {
		return nil, err
	}

	return tx, nil
}
