package types

import (
	"encoding/json"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	hlog = log.New("module", "exectype.hashlock")
)

func init() {
	types.AllowUserExec = append(types.AllowUserExec, []byte(HashlockX))
	types.RegistorExecutor(HashlockX, NewType())
	types.RegisterDappFork(HashlockX, "Enable", 0)
}

type HashlockType struct {
	types.ExecTypeBase
}

func NewType() *HashlockType {
	c := &HashlockType{}
	c.SetChild(c)
	return c
}

func (hashlock *HashlockType) GetPayload() types.Message {
	return &HashlockAction{}
}

// TODO 暂时不修改实现， 先完成结构的重构
func (hashlock *HashlockType) CreateTx(action string, message json.RawMessage) (*types.Transaction, error) {
	hlog.Debug("hashlock.CreateTx", "action", action)
	var tx *types.Transaction
	if action == "HashlockLock" {
		var param HashlockLockTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			hlog.Error("CreateTx", "Error", err)
			return nil, types.ErrInvalidParam
		}
		return CreateRawHashlockLockTx(&param)
	} else if action == "HashlockUnlock" {
		var param HashlockUnlockTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			hlog.Error("CreateTx", "Error", err)
			return nil, types.ErrInvalidParam
		}
		return CreateRawHashlockUnlockTx(&param)
	} else if action == "HashlockSend" {
		var param HashlockSendTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			hlog.Error("CreateTx", "Error", err)
			return nil, types.ErrInvalidParam
		}
		return CreateRawHashlockSendTx(&param)
	} else {
		return nil, types.ErrNotSupport
	}
	return tx, nil
}

func (hashlock *HashlockType) GetTypeMap() map[string]int32 {
	return map[string]int32{
		"Hlock":   HashlockActionLock,
		"Hsend":   HashlockActionSend,
		"Hunlock": HashlockActionUnlock,
	}
}

func (at *HashlockType) GetLogMap() map[int64]*types.LogInfo {
	return map[int64]*types.LogInfo{}
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

func (t *CoinsGetTxsByAddr) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req types.ReqAddr
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *CoinsGetTxsByAddr) ProtoToJson(reply *types.Message) (interface{}, error) {
	return reply, nil
}

func CreateRawHashlockLockTx(parm *HashlockLockTx) (*types.Transaction, error) {
	if parm == nil {
		hlog.Error("CreateRawHashlockLockTx", "parm", parm)
		return nil, types.ErrInvalidParam
	}

	v := &HashlockLock{
		Amount:        parm.Amount,
		Time:          parm.Time,
		Hash:          common.Sha256([]byte(parm.Secret)),
		ToAddress:     parm.ToAddr,
		ReturnAddress: parm.ReturnAddr,
	}
	lock := &HashlockAction{
		Ty:    HashlockActionLock,
		Value: &HashlockAction_Hlock{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(types.ExecName(HashlockX)),
		Payload: types.Encode(lock),
		Fee:     parm.Fee,
		To:      address.ExecAddress(types.ExecName(HashlockX)),
	}
	tx, err := types.FormatTx(types.ExecName(HashlockX), tx)
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

	v := &HashlockUnlock{
		Secret: []byte(parm.Secret),
	}
	unlock := &HashlockAction{
		Ty:    HashlockActionUnlock,
		Value: &HashlockAction_Hunlock{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(types.ExecName(HashlockX)),
		Payload: types.Encode(unlock),
		Fee:     parm.Fee,
		To:      address.ExecAddress(HashlockX),
	}

	tx, err := types.FormatTx(types.ExecName(HashlockX), tx)
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

	v := &HashlockSend{
		Secret: []byte(parm.Secret),
	}
	send := &HashlockAction{
		Ty:    HashlockActionSend,
		Value: &HashlockAction_Hsend{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(HashlockX),
		Payload: types.Encode(send),
		Fee:     parm.Fee,
		To:      address.ExecAddress(HashlockX),
	}
	tx, err := types.FormatTx(types.ExecName(HashlockX), tx)
	if err != nil {
		return nil, err
	}

	return tx, nil
}
