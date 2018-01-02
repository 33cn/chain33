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
	"code.aliyun.com/chain33/chain33/account"
	dbm "code.aliyun.com/chain33/chain33/common/db"
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
	}
	//return error
	return types.ErrActionNotSupport
}

func (n *Ticket) SetDB(db dbm.KVDB) {
	n.db = db
}

//only pack the transaction, exec is error.
func cutFeeReceipt(acc *types.Account, receiptBalance *types.ReceiptBalance) *types.Receipt {
	feelog := &types.ReceiptLog{types.TyLogFee, types.Encode(receiptBalance)}
	return &types.Receipt{types.ExecPack, account.GetKVSet(acc), []*types.ReceiptLog{feelog}}
}

func genesisReceipt(accTo *types.Account, receiptBalanceTo *types.ReceiptBalance, g *types.Genesis) *types.Receipt {
	log1 := &types.ReceiptLog{types.TyLogGenesis, nil}
	log2 := &types.ReceiptLog{types.TyLogTransfer, types.Encode(receiptBalanceTo)}
	kv := account.GetGenesisKVSet(g)
	kv = append(kv, account.GetKVSet(accTo)...)
	return &types.Receipt{types.ExecOk, kv, []*types.ReceiptLog{log1, log2}}
}
