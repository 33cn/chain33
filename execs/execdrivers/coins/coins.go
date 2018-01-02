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
	"code.aliyun.com/chain33/chain33/account"
	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/execs/execdrivers"
	"code.aliyun.com/chain33/chain33/types"
	log "github.com/inconshreveable/log15"
)

var clog = log.New("module", "execs.coins")

var keyBuf [200]byte

const MAX_COIN int64 = 1e16

func init() {
	execdrivers.Register("coins", newCoins())
}

type Coins struct {
	db        dbm.KVDB
	height    int64
	blocktime int64
}

func newCoins() *Coins {
	return &Coins{}
}

func (n *Coins) SetEnv(height, blocktime int64) {
	n.height = height
	n.blocktime = blocktime
}

func (n *Coins) Exec(tx *types.Transaction) (*types.Receipt, error) {
	var action types.CoinsAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return nil, err
	}
	if err = account.CheckAddress(tx.To); err != nil {
		return nil, err
	}
	clog.Info("exec transaction=", "tx=", action)
	if action.Ty == types.CoinsActionTransfer && action.GetTransfer() != nil {
		transfer := action.GetTransfer()
		if transfer.Amount <= 0 || transfer.Amount >= MAX_COIN {
			return nil, types.ErrAmount
		}
		accFrom := account.LoadAccount(n.db, account.PubKeyToAddress(tx.Signature.Pubkey).String())
		accTo := account.LoadAccount(n.db, tx.To)
		b := accFrom.GetBalance() - transfer.Amount
		clog.Info("balance ", "from", accFrom.GetBalance(), "amount", transfer.Amount)
		if b >= 0 {
			receiptBalanceFrom := &types.ReceiptBalance{accFrom.GetBalance(), b, -transfer.Amount}
			accFrom.Balance = b
			tob := accTo.GetBalance() + transfer.Amount
			receiptBalanceTo := &types.ReceiptBalance{accTo.GetBalance(), tob, transfer.Amount}
			accTo.Balance = tob
			account.SaveAccount(n.db, accFrom)
			account.SaveAccount(n.db, accTo)
			return transferReceipt(accFrom, accTo, receiptBalanceFrom, receiptBalanceTo), nil
		} else {
			return nil, types.ErrNoBalance
		}
	} else if action.Ty == types.CoinsActionGenesis && action.GetGenesis() != nil {
		genesis := action.GetGenesis()
		if n.height == 0 {
			g := &types.Genesis{}
			g.Isrun = true
			account.SaveGenesis(n.db, g)
			accTo := account.LoadAccount(n.db, tx.To)
			tob := accTo.GetBalance() + genesis.Amount
			receiptBalanceTo := &types.ReceiptBalance{accTo.GetBalance(), tob, genesis.Amount}
			accTo.Balance = tob
			account.SaveAccount(n.db, accTo)
			return genesisReceipt(accTo, receiptBalanceTo, g), nil
		} else {
			return nil, types.ErrReRunGenesis
		}
	} else {
		return nil, types.ErrActionNotSupport
	}
}

func (n *Coins) SetDB(db dbm.KVDB) {
	n.db = db
}

//only pack the transaction, exec is error.

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
