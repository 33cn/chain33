package executor

import (
	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/ticket/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (t *Ticket) execLocal(receiptData *types.ReceiptData) (*types.LocalDBSet, error) {
	dbSet := &types.LocalDBSet{}
	if receiptData.GetTy() != types.ExecOk {
		return dbSet, nil
	}
	for _, item := range receiptData.Logs {
		//这三个是ticket 的log
		if item.Ty == ty.TyLogNewTicket || item.Ty == ty.TyLogMinerTicket || item.Ty == ty.TyLogCloseTicket {
			var ticketlog ty.ReceiptTicket
			err := types.Decode(item.Log, &ticketlog)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			kv := t.saveTicket(&ticketlog)
			dbSet.KV = append(dbSet.KV, kv...)
		} else if item.Ty == ty.TyLogTicketBind {
			var ticketlog ty.ReceiptTicketBind
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

func (t *Ticket) ExecLocal_Genesis(payload *ty.TicketGenesis, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return t.execLocal(receiptData)
}

func (t *Ticket) ExecLocal_Topen(payload *ty.TicketOpen, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return t.execLocal(receiptData)
}

func (t *Ticket) ExecLocal_Tbind(payload *ty.TicketBind, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return t.execLocal(receiptData)
}

func (t *Ticket) ExecLocal_Tclose(payload *ty.TicketClose, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return t.execLocal(receiptData)
}

func (t *Ticket) ExecLocal_Miner(payload *ty.TicketMiner, tx *types.Transaction, receiptData *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return t.execLocal(receiptData)
}
