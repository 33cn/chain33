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
	if action.Ty == types.TicketActionGenesis && action.GetGenesis() != nil {
		genesis := action.GetGenesis()
		if genesis.Count <= 0 {
			return nil, types.ErrTicketCount
		}
		//new ticks
		return ticketdb.GenesisInit(n.GetDB(), tx.Hash(), n.GetAddr(), genesis, n.GetBlockTime())
	}
	//return error
	return nil, types.ErrActionNotSupport
}
