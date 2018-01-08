package coins

/*
coins 是一个货币的exec。内置货币的执行器。

主要提供两种操作：

EventTransfer -> 转移资产
*/

//package none execer for unknow execer
//all none transaction exec ok, execept nofee
//nofee transaction will not pack into block

import (
	ticketdb "code.aliyun.com/chain33/chain33/execs/db/ticket"
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

func (n *Ticket) Exec(tx *types.Transaction) (*types.Receipt, error) {
	var action types.TicketAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return nil, err
	}
	clog.Info("exec ticket tx=", "tx=", action)
	actiondb := ticketdb.NewTicketAction(n.GetDB(), tx, n.GetAddr(), n.GetBlockTime(), n.GetHeight())
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
	}
	//return error
	return nil, types.ErrActionNotSupport
}
