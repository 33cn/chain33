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

var clog = log.New("module", "execs.coins")

var keyBuf [200]byte

func init() {
	execdrivers.Register("coins", newCoins())
}

type Coins struct {
	db dbm.KVDB
}

func newCoins() *Coins {
	return &Coins{}
}

func (n *Coins) Exec(tx *types.Transaction) *types.Receipt {
	var action types.CoinsAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return errReceipt(err)
	}
	clog.Info("exec transaction=", "tx=", action)
	if action.Ty == types.CoinsActionTransfer {
		transfer := action.GetTransfer()
		accFrom := account.LoadAccount(n.db, account.PubKeyToAddress(tx.Signature.Pubkey).String())
		accTo := account.LoadAccount(n.db, tx.To)

		b := accFrom.GetBalance() - tx.Fee - transfer.Amount
		if b >= 0 {
			receiptBalanceFrom := &types.ReceiptBalance{accFrom.GetBalance(), b, -tx.Fee - transfer.Amount}
			accFrom.Balance = b
			tob := accTo.GetBalance() + transfer.Amount
			receiptBalanceTo := &types.ReceiptBalance{accTo.GetBalance(), tob, transfer.Amount}
			accTo.Balance = tob
			account.SaveAccount(n.db, accFrom)
			account.SaveAccount(n.db, accTo)
			return transferReceipt(accFrom, accTo, receiptBalanceFrom, receiptBalanceTo)
		} else if accFrom.GetBalance()-tx.Fee >= 0 {
			receiptBalance := &types.ReceiptBalance{accFrom.GetBalance(), accFrom.GetBalance() - tx.Fee, -tx.Fee}
			accFrom.Balance = accFrom.GetBalance() - tx.Fee
			account.SaveAccount(n.db, accFrom)
			return cutFeeReceipt(accFrom, receiptBalance)
		} else {
			return errReceipt(types.ErrNoBalance)
		}
	} else if action.Ty == types.CoinsActionGenesis {
		genesis := action.GetGenesis()
		g := account.GetGenesis(n.db)
		if g.Isrun {
			return errReceipt(types.ErrReRunGenesis)
		}
		g.Isrun = true
		account.SaveGenesis(n.db, g)
		accTo := account.LoadAccount(n.db, tx.To)
		tob := accTo.GetBalance() + genesis.Amount
		receiptBalanceTo := &types.ReceiptBalance{accTo.GetBalance(), tob, genesis.Amount}
		accTo.Balance = tob
		account.SaveAccount(n.db, accTo)
		return genesisReceipt(accTo, receiptBalanceTo, g)
	}
	return errReceipt(types.ErrActionNotSupport)
}

func (n *Coins) SetDB(db dbm.KVDB) {
	n.db = db
}

func errReceipt(err error) *types.Receipt {
	berr := err.Error()
	errlog := &types.ReceiptLog{types.TyLogErr, []byte(berr)}
	return &types.Receipt{types.ExecErr, nil, []*types.ReceiptLog{errlog}}
}

func cutFeeReceipt(acc *types.Account, receiptBalance *types.ReceiptBalance) *types.Receipt {
	feelog := &types.ReceiptLog{types.TyLogFee, types.Encode(receiptBalance)}
	return &types.Receipt{types.ExecErr, account.GetKVSet(acc), []*types.ReceiptLog{feelog}}
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
