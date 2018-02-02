package ticket

/*
coins 是一个货币的exec。内置货币的执行器。

主要提供两种操作：

EventTransfer -> 转移资产
*/

//package none execer for unknow execer
//all none transaction exec ok, execept nofee
//nofee transaction will not pack into block

import (
	"fmt"

	"code.aliyun.com/chain33/chain33/execs/execdrivers"
	"code.aliyun.com/chain33/chain33/types"
	log "github.com/inconshreveable/log15"
)

var clog = log.New("module", "execs.ticket")

func init() {
	execdrivers.Register("ticket", newTicket())
	execdrivers.RegisterAddress("ticket")
}

type Ticket struct {
	execdrivers.ExecBase
}

func newTicket() *Ticket {
	t := &Ticket{}
	t.SetChild(t)
	return t
}

func (n *Ticket) GetName() string {
	return "ticket"
}

func (n *Ticket) GetActionName(tx *types.Transaction) string {
	var action types.TicketAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return "unknow"
	}
	if action.Ty == types.TicketActionGenesis && action.GetGenesis() != nil {
		return "genesis"
	} else if action.Ty == types.TicketActionOpen && action.GetTopen() != nil {
		return "open"
	} else if action.Ty == types.TicketActionClose && action.GetTclose() != nil {
		return "close"
	} else if action.Ty == types.TicketActionMiner && action.GetMiner() != nil {
		return "miner"
	}
	return "unknow"
}

func (n *Ticket) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	var action types.TicketAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return nil, err
	}
	clog.Info("exec ticket tx=", "tx=", action)
	actiondb := NewTicketAction(n.GetDB(), tx, n.GetAddr(), n.GetBlockTime(), n.GetHeight())
	if action.Ty == types.TicketActionGenesis && action.GetGenesis() != nil {
		genesis := action.GetGenesis()
		if genesis.Count <= 0 {
			return nil, types.ErrTicketCount
		}
		//new ticks
		return actiondb.GenesisInit(genesis)
	} else if action.Ty == types.TicketActionOpen && action.GetTopen() != nil {
		topen := action.GetTopen()
		if topen.Count <= 0 {
			return nil, types.ErrTicketCount
		}
		return actiondb.TicketOpen(topen)
	} else if action.Ty == types.TicketActionClose && action.GetTclose() != nil {
		tclose := action.GetTclose()
		return actiondb.TicketClose(tclose)
	} else if action.Ty == types.TicketActionMiner && action.GetMiner() != nil {
		miner := action.GetMiner()
		return actiondb.TicketMiner(miner, index)
	}
	//return error
	return nil, types.ErrActionNotSupport
}

func (n *Ticket) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := n.ExecLocalCommon(tx, receipt, index)
	if err != nil {
		return nil, err
	}
	if receipt.GetTy() != types.ExecOk {
		return set, nil
	}
	for i := 0; i < len(receipt.Logs); i++ {
		item := receipt.Logs[i]
		//这三个是ticket 的log
		if item.Ty == types.TyLogNewTicket || item.Ty == types.TyLogMinerTicket || item.Ty == types.TyLogCloseTicket {
			var ticketlog types.ReceiptTicket
			err := types.Decode(item.Log, &ticketlog)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			kv := n.saveTicket(&ticketlog)
			set.KV = append(set.KV, kv...)
		}
	}
	return set, nil
}

func (n *Ticket) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := n.ExecDelLocalCommon(tx, receipt, index)
	if err != nil {
		return nil, err
	}
	if receipt.GetTy() != types.ExecOk {
		return set, nil
	}
	for i := 0; i < len(receipt.Logs); i++ {
		item := receipt.Logs[i]
		//这三个是ticket 的log
		if item.Ty == types.TyLogNewTicket || item.Ty == types.TyLogMinerTicket || item.Ty == types.TyLogCloseTicket {
			var ticketlog types.ReceiptTicket
			err := types.Decode(item.Log, &ticketlog)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			kv := n.delTicket(&ticketlog)
			set.KV = append(set.KV, kv...)
		}
	}
	return set, nil
}

func (n *Ticket) saveTicket(ticketlog *types.ReceiptTicket) (kvs []*types.KeyValue) {
	if ticketlog.PrevStatus > 0 {
		kv := delticket(ticketlog.Addr, ticketlog.TicketId, ticketlog.PrevStatus)
		kvs = append(kvs, kv)
	}
	kvs = append(kvs, addticket(ticketlog.Addr, ticketlog.TicketId, ticketlog.Status))
	return kvs
}

func (n *Ticket) delTicket(ticketlog *types.ReceiptTicket) (kvs []*types.KeyValue) {
	if ticketlog.PrevStatus > 0 {
		kv := addticket(ticketlog.Addr, ticketlog.TicketId, ticketlog.PrevStatus)
		kvs = append(kvs, kv)
	}
	kvs = append(kvs, delticket(ticketlog.Addr, ticketlog.TicketId, ticketlog.Status))
	return kvs
}

func (n *Ticket) Query(funcname string, params []byte) (types.Message, error) {
	if funcname == "TicketInfos" {
		var info types.TicketInfos
		err := types.Decode(params, &info)
		if err != nil {
			return nil, err
		}
		return TicketInfos(n.GetDB(), &info)
	} else if funcname == "TicketList" {
		var l types.TicketList
		err := types.Decode(params, &l)
		if err != nil {
			return nil, err
		}
		return TicketList(n.GetQueryDB(), n.GetDB(), &l)
	}
	return nil, types.ErrActionNotSupport
}

func calcTicketKey(addr string, ticketId string, status int32) []byte {
	key := fmt.Sprintf("ticket-tl:%s:%d:%s", addr, status, ticketId)
	return []byte(key)
}

func calcTicketPrefix(addr string, status int32) []byte {
	key := fmt.Sprintf("ticket-tl:%s:%d", addr, status)
	return []byte(key)
}

func addticket(addr string, ticketId string, status int32) *types.KeyValue {
	kv := &types.KeyValue{}
	kv.Key = calcTicketKey(addr, ticketId, status)
	kv.Value = []byte(ticketId)
	return kv
}

func delticket(addr string, ticketId string, status int32) *types.KeyValue {
	kv := &types.KeyValue{}
	kv.Key = calcTicketKey(addr, ticketId, status)
	kv.Value = nil
	return kv
}
