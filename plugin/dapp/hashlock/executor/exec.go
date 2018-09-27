package executor

import (
	pty "gitlab.33.cn/chain33/chain33/plugin/dapp/hashlock/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (h *Hashlock) Exec_Hlock(transfer *pty.HashlockLock, tx *types.Transaction, index int) (*types.Receipt, error) {
	return nil, nil
}

func (h *Hashlock) Exec_Hsend(transfer *pty.HashlockSend, tx *types.Transaction, index int) (*types.Receipt, error) {
	return nil, nil
}

func (h *Hashlock) Exec_Hunlock(transfer *pty.HashlockUnlock, tx *types.Transaction, index int) (*types.Receipt, error) {
	return nil, nil
}
