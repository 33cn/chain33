package hashlock

import (
	"time"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/types"
)

var clog = log.New("module", "execs.hashlock")

const minLockTime = 60

func init() {
	h := newHashlock()
	drivers.Register(h.GetName(), h, 0)
}

type Hashlock struct {
	drivers.DriverBase
}

func newHashlock() *Hashlock {
	h := &Hashlock{}
	h.SetChild(h)
	return h
}

func (h *Hashlock) GetName() string {
	return "hashlock"
}

func (h *Hashlock) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	var action types.HashlockAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return nil, err
	}

	clog.Debug("exec hashlock tx=", "tx=", action)
	actiondb := NewAction(h, tx, h.GetAddr())
	if action.Ty == types.HashlockActionLock && action.GetHlock() != nil {
		clog.Debug("hashlocklock action")
		hlock := action.GetHlock()
		if hlock.Amount <= 0 {
			clog.Warn("hashlock amount <=0")
			return nil, types.ErrHashlockAmount
		}
		if err := account.CheckAddress(hlock.ToAddress); err != nil {
			clog.Warn("hashlock checkaddress")
			return nil, err
		}
		if err := account.CheckAddress(hlock.ReturnAddress); err != nil {
			clog.Warn("hashlock checkaddress")
			return nil, err
		}
		if hlock.ReturnAddress != account.From(tx).String() {
			clog.Warn("hashlock return address")
			return nil, types.ErrHashlockReturnAddrss
		}

		if hlock.Time <= minLockTime {
			clog.Warn("exec hashlock time not enough")
			return nil, types.ErrHashlockTime
		}
		return actiondb.Hashlocklock(hlock)
	} else if action.Ty == types.HashlockActionUnlock && action.GetHunlock() != nil {
		hunlock := action.GetHunlock()
		//unlock 有两个条件： 1. 时间已经过期 2. 密码是对的，返回原来的账户
		clog.Debug("hashlockunlock action")
		return actiondb.Hashlockunlock(hunlock)
	} else if action.Ty == types.HashlockActionSend && action.GetHsend() != nil {
		hsend := action.GetHsend()
		//send 有两个条件：1. 时间没有过期 2. 密码是对的，币转移到 ToAddress
		clog.Debug("hashlocksend action")
		return actiondb.Hashlocksend(hsend)
	}

	//return error
	return nil, types.ErrActionNotSupport
}

//获取运行状态名
func (h *Hashlock) GetActionName(tx *types.Transaction) string {
	return tx.ActionName()
}

//把信息进行存储
//将运行结果内容存入本地地址

func (h *Hashlock) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	clog.Error("ExecLocal action")
	set, err := h.DriverBase.ExecLocal(tx, receipt, index)
	if err != nil {
		clog.Error("ExecLocal")
		return nil, err
	}
	if receipt.GetTy() != types.ExecOk {

		return set, nil
	}
	//执行成功
	var action types.HashlockAction
	err = types.Decode(tx.GetPayload(), &action)
	if err != nil {
		panic(err)
	}

	var kv *types.KeyValue
	if action.Ty == types.HashlockActionLock && action.GetHlock() != nil {
		hlock := action.GetHlock()
		info := types.Hashlockquery{hlock.Time, hashlockLocked, hlock.Amount, h.GetBlockTime(), 0}
		clog.Error("ExecLocal", "info", info)
		kv, err = UpdateHashReciver(h.GetLocalDB(), hlock.Hash, info)
	} else if action.Ty == types.HashlockActionUnlock && action.GetHunlock() != nil {
		hunlock := action.GetHunlock()
		info := types.Hashlockquery{0, hashlockUnlocked, 0, 0, 0}
		clog.Error("ExecLocal", "info", info)
		kv, err = UpdateHashReciver(h.GetLocalDB(), common.Sha256(hunlock.Secret), info)
	} else if action.Ty == types.HashlockActionSend && action.GetHsend() != nil {
		hsend := action.GetHsend()
		info := types.Hashlockquery{0, hashlockSent, 0, 0, 0}
		clog.Error("ExecLocal", "info", info)
		kv, err = UpdateHashReciver(h.GetLocalDB(), common.Sha256(hsend.Secret), info)
	}
	if err != nil {
		return set, nil
	}
	if kv != nil {
		set.KV = append(set.KV, kv)
	}
	return set, nil
}

func (h *Hashlock) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := h.DriverBase.ExecDelLocal(tx, receipt, index)
	if err != nil {
		return nil, err
	}
	if receipt.GetTy() != types.ExecOk {
		return set, nil
	}
	//执行成功
	var action types.HashlockAction
	err = types.Decode(tx.GetPayload(), &action)
	if err != nil {
		panic(err)
	}
	var kv *types.KeyValue
	if action.Ty == types.HashlockActionLock && action.GetHlock() != nil {
		hlock := action.GetHlock()
		info := types.Hashlockquery{hlock.Time, hashlockLocked, hlock.Amount, h.GetBlockTime(), 0}
		kv, err = UpdateHashReciver(h.GetLocalDB(), hlock.Hash, info)
	} else if action.Ty == types.HashlockActionUnlock && action.GetHunlock() != nil {
		hunlock := action.GetHunlock()
		info := types.Hashlockquery{0, hashlockUnlocked, 0, 0, 0}
		kv, err = UpdateHashReciver(h.GetLocalDB(), common.Sha256(hunlock.Secret), info)
	} else if action.Ty == types.HashlockActionSend && action.GetHsend() != nil {
		hsend := action.GetHsend()
		info := types.Hashlockquery{0, hashlockSent, 0, 0, 0}
		kv, err = UpdateHashReciver(h.GetLocalDB(), common.Sha256(hsend.Secret), info)
	}
	if err != nil {
		return set, nil
	}
	if kv != nil {
		set.KV = append(set.KV, kv)
	}
	return set, nil
}

func (h *Hashlock) Query(funcName string, hashlockID []byte) (types.Message, error) {
	if funcName == "GetHashlocKById" {
		//		currentTime := n.GetBlockTime()
		differTime := time.Now().UnixNano()/1e9 - h.GetBlockTime()
		clog.Error("Query action")
		return h.GetTxsByHashlockID(hashlockID, differTime)
	}
	return nil, types.ErrActionNotSupport
}
