package executor

import (
	"gitlab.33.cn/chain33/chain33/common"
	pty "gitlab.33.cn/chain33/chain33/plugin/dapp/hashlock/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (h *Hashlock) ExecDelLocal_Hlock(hlock *pty.HashlockLock, tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	info := pty.Hashlockquery{hlock.Time, hashlockLocked, hlock.Amount, h.GetBlockTime(), 0}
	kv, err := UpdateHashReciver(h.GetLocalDB(), hlock.Hash, info)
	if err != nil {
		return nil, err
	}
	return &types.LocalDBSet{KV: []*types.KeyValue{kv}}, nil
}

func (h *Hashlock) ExecDelLocal_Hsend(hsend *pty.HashlockSend, tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	info := pty.Hashlockquery{0, hashlockSent, 0, 0, 0}
	kv, err := UpdateHashReciver(h.GetLocalDB(), common.Sha256(hsend.Secret), info)
	if err != nil {
		return nil, err
	}
	return &types.LocalDBSet{KV: []*types.KeyValue{kv}}, nil
}

func (h *Hashlock) ExecDelLocal_Hunlock(hunlock *pty.HashlockUnlock, tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	info := pty.Hashlockquery{0, hashlockUnlocked, 0, 0, 0}
	kv, err := UpdateHashReciver(h.GetLocalDB(), common.Sha256(hunlock.Secret), info)
	if err != nil {
		return nil, err
	}
	return &types.LocalDBSet{KV: []*types.KeyValue{kv}}, nil
}
