package coins

/*
coins 是一个货币的exec。内置货币的执行器。

主要提供两种操作：

EventFee -> 扣除手续费
EventTransfer -> 转移资产
*/

//package none execer for unknow execer
//all none transaction exec ok, execept nofee
//nofee transaction will not pack into block

import (
	"fmt"
	"code.aliyun.com/chain33/chain33/account"
	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/common"
	"code.aliyun.com/chain33/chain33/execs/execdrivers"
	"code.aliyun.com/chain33/chain33/types"
	log "github.com/inconshreveable/log15"
)

var clog = log.New("module", "execs.ticket")

var keyBuf [200]byte

const MAX_COIN int64 = 1e16

func init() {
	execdrivers.Register("ticket", newTicket())
}

type Ticket struct {
	db        dbm.KVDB
	height    int64
	blocktime int64
}

func newTicket() *Ticket {
	return &Ticket{}
}

func (n *Ticket) SetEnv(height, blocktime int64) {
	n.height = height
	n.blocktime = blocktime
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
			return nil, types.ErrTicketCount)
		}
		//new ticks
		return genesisReceipt(tx.Hash(), genesis), nil
	}
	//return error
	return nil, types.ErrActionNotSupport
}

func (n *Ticket) SetDB(db dbm.KVDB) {
	n.db = db
}

func genesisReceipt(hash []byte, genesis *types.TicketGenesis) *types.Receipt {
	prefix := common.ToHex(hash)
	preifx = genesis.MinerAddress + ":" + prefix + ":"
	var logs []*types.ReceiptLog
	var kv   []*types.KeyValue
	for i := 0; i < genesis.Count; i++ {
		id := prefix + fmt.Sprintf("%010d", i)
		t := ticketdb.NewTicket(id, genesis.MinerAddress, genesis.CodeWallet)
		logs = append(logs, t.GetReceiptLog())
		kv = append(kv, t.GetKVSet()...)
	}
	receipt := &types.Receipt{types.ExecOk, kv, logs}
	return receipt
}
