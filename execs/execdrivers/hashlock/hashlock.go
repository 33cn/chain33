package hashlock

import (
	"code.aliyun.com/chain33/chain33/account"
	hashlockdb "code.aliyun.com/chain33/chain33/execs/hashlockdb"
	"code.aliyun.com/chain33/chain33/execs/execdrivers"
	"code.aliyun.com/chain33/chain33/types"
	log "github.com/inconshreveable/log15"
)

var clog = log.New("module", "execs.hashlock")

const minLockTime = 60

func init() {
	execdrivers.Register("hashlock", newHashlock())
	execdrivers.RegisterAddress("hashlock")
}

type Hashlock struct {
	execdrivers.ExecBase
}

func newHashlock() *Hashlock {
	h := &Hashlock{}
	h.SetChild(h)
	return h
}

//暂时不被调用
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

	actiondb := NewHashlockAction(h.GetDB(), tx, h.GetAddr(), h.GetBlockTime(), h.GetHeight())

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
	var action types.HashlockAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return "unknow"
	}
	if err = account.CheckAddress(tx.To); err != nil {
		return "unknow"
	}
	if  action.Ty == types.HashlockActionLock && action.GetHlock() != nil {
		return "lock"
	} else if action.Ty == types.HashlockActionUnlock && action.GetHunlock() != nil {
		return "unlock"
	} else if action.Ty == types.HashlockActionSend && action.GetHsend() != nil  {
		return "send"
	} else {
		return "unknow"
	}
}
//把信息进行存储
//将运行结果内容存入本地地址

func (h *Hashlock) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := n.ExecLocalCommon(tx, receipt, index)
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
		kv, err = updateHashReciver(h.GetLocalDB(), hlock.hash, hlock.Amount, 1)
	} else if action.Ty == types.HashlockActionUnlock && action.GetHunlock() != nil {
		hunlock := action.GetHunlock()
		//from := account.PubKeyToAddress(tx.Signature.Pubkey).String()
		kv, err = updateHashReciver(h.GetLocalDB(), hunlock.hash, hunlock.Amount, 3)
	} else if action.Ty == types.HashlockActionSend && action.GetHsend() != nil {
		hsend := action.GetHsend()
		kv, err = updateHashReciver(h.GetLocalDB(), hsend.hash, hsend.Amount, 2)
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
	set, err := n.ExecDelLocalCommon(tx, receipt, index)
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
		kv, err = updateHashReciver(h.GetLocalDB(), hlock.hash, hlock.Amount, 1)
	} else if action.Ty == types.HashlockActionUnlock && action.GetHunlock() != nil {
		hunlock := action.GetHunlock()
//		from := account.PubKeyToAddress(tx.Signature.Pubkey).String()
		kv, err = updateHashReciver(h.GetLocalDB(), hunlock.hash, hunlock.Amount, 3)
	}else if action.Ty == types.HashlockActionSend && action.GetHsend() != nil {
		hsend := action.GetHsend()
		kv, err = updateHashReciver(h.GetLocalDB(), hsend.hash, hsend.Amount, 2)
	}
	if err != nil {
		return set, nil
	}
	if kv != nil {
		set.KV = append(set.KV, kv)
	}
	return set, nil
}

func (n *Hashlock) Query(funcName string, HashlockId []byte) (execs.Hashlockquery, error) {
	if funcName == "GetHashlocKById" {
		return n.GetHashReciver(HashlockId)
	}
	return nil, types.ErrActionNotSupport
}
