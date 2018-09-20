package executor

import "gitlab.33.cn/chain33/chain33/types"

func (t *Ticket) Exec_Genesis(payload *types.TicketGenesis, tx *types.Transaction, index int) (*types.Receipt, error) {
	if payload.Count <= 0 {
		return nil, types.ErrTicketCount
	}
	actiondb := NewAction(t, tx)
	return actiondb.GenesisInit(payload)
}

func (t *Ticket) Exec_Open(payload *types.TicketOpen, tx *types.Transaction, index int) (*types.Receipt, error) {
	if payload.Count <= 0 {
		tlog.Error("topen ", "value", payload)
		return nil, types.ErrTicketCount
	}
	actiondb := NewAction(t, tx)
	return actiondb.TicketOpen(payload)
}

func (t *Ticket) Exec_Bind(payload *types.TicketBind, tx *types.Transaction, index int) (*types.Receipt, error) {
	actiondb := NewAction(t, tx)
	return actiondb.TicketBind(payload)
}

func (t *Ticket) Exec_Close(payload *types.TicketClose, tx *types.Transaction, index int) (*types.Receipt, error) {
	actiondb := NewAction(t, tx)
	return actiondb.TicketClose(payload)
}

func (t *Ticket) Exec_Miner(payload *types.TicketMiner, tx *types.Transaction, index int) (*types.Receipt, error) {
	actiondb := NewAction(t, tx)
	return actiondb.TicketMiner(payload, index)
}
