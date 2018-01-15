package coins

import (
	"code.aliyun.com/chain33/chain33/account"
	hashlockdb "code.aliyun.com/chain33/chain33/execs/db/hashlock"
	"code.aliyun.com/chain33/chain33/execs/execdrivers"
	"code.aliyun.com/chain33/chain33/types"
	log "github.com/inconshreveable/log15"
)

var clog = log.New("module", "execs.hashlock")

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

	clog.Info("exec hashlock tx=", "tx=", action)

	actiondb := hashlockdb.NewHashlockAction(h.GetDB(), tx, h.GetAddr(), h.GetBlockTime(), h.GetHeight())

	//暂时不实现genesis
	if action.Ty == types.HashlockActionLock && action.GetHlock() != nil {
		hlock := action.GetHlock()
		if hlock.Amount <= 0 {
			return nil, types.ErrHashlockAmount
		}
		if err := account.CheckAddress(hlock.ToAddress); err != nil {
			return nil, err
		}
		if err := account.CheckAddress(hlock.ReturnAddress); err != nil {
			return nil, err
		}
		if hlock.ReturnAddress != account.From(tx).String() {
			return nil, types.ErrHashlockReturnAddrss
		}
		//最少lock 一个小时
		if hlock.Time <= 3600 {
			return nil, types.ErrHashlockTime
		}
		return actiondb.Hashlocklock(hlock)
	} else if action.Ty == types.HashlockActionUnlock && action.GetHunlock() != nil {
		hunlock := action.GetHunlock()
		//unlock 有两个条件： 1. 时间已经过期 2. 密码是对的，返回原来的账户
		return actiondb.Hashlockunlock(hunlock)
	} else if action.Ty == types.HashlockActionSend() && action.GetHsend() != nil {
		hsend := action.GetHsend()
		//send 有两个条件：1. 时间没有过期 2. 密码是对的，币转移到 ToAddress
		return actiondb.Hashlocksend(hsend)
	}

	//return error
	return nil, types.ErrActionNotSupport
}
