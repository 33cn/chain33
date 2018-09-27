package executor

import "gitlab.33.cn/chain33/chain33/types"

func (t *Ticket) execLocal(receiptData *types.ReceiptData) (*types.LocalDBSet, error) {
	dbSet := &types.LocalDBSet{}
	if receiptData.GetTy() != types.ExecOk {
		return dbSet, nil
	}
	for _, item := range receiptData.Logs {
		//这三个是ticket 的log
		if item.Ty == types.TyLogNewTicket || item.Ty == types.TyLogMinerTicket || item.Ty == types.TyLogCloseTicket {
			var ticketlog types.ReceiptTicket
			err := types.Decode(item.Log, &ticketlog)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			kv := t.saveTicket(&ticketlog)
			dbSet.KV = append(dbSet.KV, kv...)
		} else if item.Ty == types.TyLogTicketBind {
			var ticketlog types.ReceiptTicketBind
			err := types.Decode(item.Log, &ticketlog)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			kv := t.saveTicketBind(&ticketlog)
			dbSet.KV = append(dbSet.KV, kv...)
		}
	}
	return dbSet, nil
}

func (t *Ticket) ExecLocal_Genesis(payload *types.TicketGenesis, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return t.execLocal(receiptData)
}

func (t *Ticket) ExecLocal_Open(payload *types.TicketOpen, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return t.execLocal(receiptData)
}

func (t *Ticket) ExecLocal_Bind(payload *types.TicketBind, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return t.execLocal(receiptData)
}

func (t *Ticket) ExecLocal_Close(payload *types.TicketClose, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return t.execLocal(receiptData)
}

func (t *Ticket) ExecLocal_Miner(payload *types.TicketMiner, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return t.execLocal(receiptData)
}
