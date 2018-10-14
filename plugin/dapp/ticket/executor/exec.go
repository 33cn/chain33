package executor

import (
	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/ticket/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (t *Ticket) Exec_Genesis(payload *ty.TicketGenesis, tx *types.Transaction, index int) (*types.Receipt, error) {
	if payload.Count <= 0 {
		return nil, types.ErrTicketCount
	}
	actiondb := NewAction(t, tx)
	return actiondb.GenesisInit(payload)
}

func (t *Ticket) Exec_Open(payload *ty.TicketOpen, tx *types.Transaction, index int) (*types.Receipt, error) {
	if payload.Count <= 0 {
		tlog.Error("topen ", "value", payload)
		return nil, types.ErrTicketCount
	}
	actiondb := NewAction(t, tx)
	return actiondb.TicketOpen(payload)
}

func (t *Ticket) Exec_Bind(payload *ty.TicketBind, tx *types.Transaction, index int) (*types.Receipt, error) {
	actiondb := NewAction(t, tx)
	return actiondb.TicketBind(payload)
}

func (t *Ticket) Exec_Close(payload *ty.TicketClose, tx *types.Transaction, index int) (*types.Receipt, error) {
	actiondb := NewAction(t, tx)
	return actiondb.TicketClose(payload)
}

func (t *Ticket) Exec_Miner(payload *ty.TicketMiner, tx *types.Transaction, index int) (*types.Receipt, error) {
	actiondb := NewAction(t, tx)
	return actiondb.TicketMiner(payload, index)
}
