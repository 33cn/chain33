package executor

import (
	"gitlab.33.cn/chain33/chain33/common/address"
	pty "gitlab.33.cn/chain33/chain33/plugin/dapp/hashlock/types"
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	"gitlab.33.cn/chain33/chain33/types"
)

func (h *Hashlock) Exec_Hlock(hlock *pty.HashlockLock, tx *types.Transaction, index int) (*types.Receipt, error) {
	clog.Debug("hashlocklock action")
	if hlock.Amount <= 0 {
		clog.Warn("hashlock amount <=0")
		return nil, pty.ErrHashlockAmount
	}
	if err := address.CheckAddress(hlock.ToAddress); err != nil {
		clog.Warn("hashlock checkaddress")
		return nil, err
	}
	if err := address.CheckAddress(hlock.ReturnAddress); err != nil {
		clog.Warn("hashlock checkaddress")
		return nil, err
	}
	if hlock.ReturnAddress != tx.From() {
		clog.Warn("hashlock return address")
		return nil, pty.ErrHashlockReturnAddrss
	}

	if hlock.Time <= minLockTime {
		clog.Warn("exec hashlock time not enough")
		return nil, pty.ErrHashlockTime
	}
	actiondb := NewAction(h, tx, drivers.ExecAddress(string(tx.Execer)))
	return actiondb.Hashlocklock(hlock)
}

func (h *Hashlock) Exec_Hsend(transfer *pty.HashlockSend, tx *types.Transaction, index int) (*types.Receipt, error) {
	//unlock 有两个条件： 1. 时间已经过期 2. 密码是对的，返回原来的账户
	clog.Debug("hashlockunlock action")
	actiondb := NewAction(h, tx, drivers.ExecAddress(string(tx.Execer)))
	return actiondb.Hashlocksend(transfer)
}

func (h *Hashlock) Exec_Hunlock(transfer *pty.HashlockUnlock, tx *types.Transaction, index int) (*types.Receipt, error) {
	//send 有两个条件：1. 时间没有过期 2. 密码是对的，币转移到 ToAddress
	clog.Debug("hashlocksend action")
	actiondb := NewAction(h, tx, drivers.ExecAddress(string(tx.Execer)))
	return actiondb.Hashlockunlock(transfer)
}
