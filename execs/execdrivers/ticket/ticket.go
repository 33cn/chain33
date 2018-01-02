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
	db dbm.KVDB
}

func newTicket() *Ticket {
	return &Ticket{}
}

func (n *Coins) Exec(tx *types.Transaction) *types.Receipt {
	var action types.CoinsAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		goto cutFee //尽量收取手续费
	}
	if err = account.CheckAddress(tx.To); err != nil {
		goto cutFee
	}
	clog.Info("exec transaction=", "tx=", action)
	if action.Ty == types.CoinsActionTransfer && action.GetTransfer() != nil {
		transfer := action.GetTransfer()
		if transfer.Amount <= 0 || transfer.Amount >= MAX_COIN {
			return types.NewErrReceipt(types.ErrAmount)
		}
		accFrom := account.LoadAccount(n.db, account.PubKeyToAddress(tx.Signature.Pubkey).String())
		accTo := account.LoadAccount(n.db, tx.To)
		b := accFrom.GetBalance() - tx.Fee - transfer.Amount
		clog.Info("balance ", "from", accFrom.GetBalance(), "fee", tx.Fee, "amount", transfer.Amount)
		if b >= 0 {
			receiptBalanceFrom := &types.ReceiptBalance{accFrom.GetBalance(), b, -tx.Fee - transfer.Amount}
			accFrom.Balance = b
			tob := accTo.GetBalance() + transfer.Amount
			receiptBalanceTo := &types.ReceiptBalance{accTo.GetBalance(), tob, transfer.Amount}
			accTo.Balance = tob
			account.SaveAccount(n.db, accFrom)
			account.SaveAccount(n.db, accTo)
			return transferReceipt(accFrom, accTo, receiptBalanceFrom, receiptBalanceTo)
		}
	} else if action.Ty == types.CoinsActionGenesis && action.GetGenesis() != nil {
		genesis := action.GetGenesis()
		g := account.GetGenesis(n.db)
		if !g.Isrun {
			g.Isrun = true
			account.SaveGenesis(n.db, g)
			accTo := account.LoadAccount(n.db, tx.To)
			tob := accTo.GetBalance() + genesis.Amount
			receiptBalanceTo := &types.ReceiptBalance{accTo.GetBalance(), tob, genesis.Amount}
			accTo.Balance = tob
			account.SaveAccount(n.db, accTo)
			return genesisReceipt(accTo, receiptBalanceTo, g)
		}
	}

cutFee:
	accFrom := account.LoadAccount(n.db, account.PubKeyToAddress(tx.Signature.Pubkey).String())
	if accFrom.GetBalance()-tx.Fee >= 0 {
		receiptBalance := &types.ReceiptBalance{accFrom.GetBalance(), accFrom.GetBalance() - tx.Fee, -tx.Fee}
		accFrom.Balance = accFrom.GetBalance() - tx.Fee
		account.SaveAccount(n.db, accFrom)
		return cutFeeReceipt(accFrom, receiptBalance)
	}
	//return error
	return types.NewErrReceipt(types.ErrActionNotSupport)
}

func (n *Coins) SetDB(db dbm.KVDB) {
	n.db = db
}

//only pack the transaction, exec is error.
func cutFeeReceipt(acc *types.Account, receiptBalance *types.ReceiptBalance) *types.Receipt {
	feelog := &types.ReceiptLog{types.TyLogFee, types.Encode(receiptBalance)}
	return &types.Receipt{types.ExecPack, account.GetKVSet(acc), []*types.ReceiptLog{feelog}}
}

func transferReceipt(accFrom, accTo *types.Account, receiptBalanceFrom, receiptBalanceTo *types.ReceiptBalance) *types.Receipt {
	log1 := &types.ReceiptLog{types.TyLogTransfer, types.Encode(receiptBalanceFrom)}
	log2 := &types.ReceiptLog{types.TyLogTransfer, types.Encode(receiptBalanceTo)}
	kv := account.GetKVSet(accFrom)
	kv = append(kv, account.GetKVSet(accTo)...)
	return &types.Receipt{types.ExecOk, kv, []*types.ReceiptLog{log1, log2}}
}

func genesisReceipt(accTo *types.Account, receiptBalanceTo *types.ReceiptBalance, g *types.Genesis) *types.Receipt {
	log1 := &types.ReceiptLog{types.TyLogGenesis, nil}
	log2 := &types.ReceiptLog{types.TyLogTransfer, types.Encode(receiptBalanceTo)}
	kv := account.GetGenesisKVSet(g)
	kv = append(kv, account.GetKVSet(accTo)...)
	return &types.Receipt{types.ExecOk, kv, []*types.ReceiptLog{log1, log2}}
}
