package executor

import "gitlab.33.cn/chain33/chain33/types"

func (t *Ticket) execDelLocal(receiptData *types.ReceiptData) (*types.LocalDBSet, error) {
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
			kv := t.delTicket(&ticketlog)
			dbSet.KV = append(dbSet.KV, kv...)
		} else if item.Ty == types.TyLogTicketBind {
			var ticketlog types.ReceiptTicketBind
			err := types.Decode(item.Log, &ticketlog)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			kv := t.delTicketBind(&ticketlog)
			dbSet.KV = append(dbSet.KV, kv...)
		}
	}
	return dbSet, nil
}

func (t *Ticket) ExecDelLocal_Genesis(payload *types.TicketGenesis, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return nil, nil
}

func (t *Ticket) ExecDelLocal_Open(payload *types.TicketOpen, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return nil, nil
}

func (t *Ticket) ExecDelLocal_Bind(payload *types.TicketBind, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return nil, nil
}

func (t *Ticket) ExecDelLocal_Close(payload *types.TicketClose, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return nil, nil
}

func (t *Ticket) ExecDelLocal_Miner(payload *types.TicketMiner, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return nil, nil
}
