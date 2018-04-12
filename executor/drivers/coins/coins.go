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
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/types"
)

var clog = log.New("module", "execs.coins")

func init() {
	n := newCoins()
	drivers.Register(n.GetName(), n, 0)
}

type Coins struct {
	drivers.DriverBase
}

func newCoins() *Coins {
	c := &Coins{}
	c.SetChild(c)
	return c
}

func (c *Coins) GetName() string {
	return "coins"
}

func (c *Coins) Clone() drivers.Driver {
	clone := &Coins{}
	clone.DriverBase = *(c.DriverBase.Clone().(*drivers.DriverBase))
	clone.SetChild(clone)
	return clone
}

func (c *Coins) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	_, err := c.DriverBase.Exec(tx, index)
	if err != nil {
		return nil, err
	}
	var action types.CoinsAction
	err = types.Decode(tx.Payload, &action)
	if err != nil {
		return nil, err
	}
	coinsAccount := c.GetCoinsAccount()
	if action.Ty == types.CoinsActionTransfer && action.GetTransfer() != nil {
		transfer := action.GetTransfer()
		from := account.From(tx).String()
		//to 是 execs 合约地址
		if c.GetExecDriver().IsDriverAddress(tx.To) {
			return coinsAccount.TransferToExec(from, tx.To, transfer.Amount)
		}

		return coinsAccount.Transfer(from, tx.To, transfer.Amount)
	} else if action.Ty == types.CoinsActionWithdraw && action.GetWithdraw() != nil {
		withdraw := action.GetWithdraw()
		from := account.PubKeyToAddress(tx.Signature.Pubkey).String()
		//to 是 execs 合约地址
		if c.GetExecDriver().IsDriverAddress(tx.To) {
			return coinsAccount.TransferWithdraw(from, tx.To, withdraw.Amount)
		}
		return nil, types.ErrActionNotSupport
	} else if action.Ty == types.CoinsActionGenesis && action.GetGenesis() != nil {
		genesis := action.GetGenesis()
		if c.GetHeight() == 0 {
			if c.GetExecDriver().IsDriverAddress(tx.To) {
				return coinsAccount.GenesisInitExec(genesis.ReturnAddress, genesis.Amount, tx.To)
			}
			return coinsAccount.GenesisInit(tx.To, genesis.Amount)
		} else {
			return nil, types.ErrReRunGenesis
		}
	} else {
		return nil, types.ErrActionNotSupport
	}
}

//0: all tx
//1: from tx
//2: to tx

func (c *Coins) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := c.DriverBase.ExecLocal(tx, receipt, index)
	if err != nil {
		return nil, err
	}
	if receipt.GetTy() != types.ExecOk {
		return set, nil
	}
	//执行成功
	var action types.CoinsAction
	err = types.Decode(tx.GetPayload(), &action)
	if err != nil {
		panic(err)
	}
	var kv *types.KeyValue
	if action.Ty == types.CoinsActionTransfer && action.GetTransfer() != nil {
		transfer := action.GetTransfer()
		kv, err = updateAddrReciver(c.GetLocalDB(), tx.To, transfer.Amount, true)
	} else if action.Ty == types.CoinsActionWithdraw && action.GetWithdraw() != nil {
		transfer := action.GetWithdraw()
		from := account.PubKeyToAddress(tx.Signature.Pubkey).String()
		kv, err = updateAddrReciver(c.GetLocalDB(), from, transfer.Amount, true)
	} else if action.Ty == types.CoinsActionGenesis && action.GetGenesis() != nil {
		gen := action.GetGenesis()
		kv, err = updateAddrReciver(c.GetLocalDB(), tx.To, gen.Amount, true)
	}
	if err != nil {
		return set, nil
	}
	if kv != nil {
		set.KV = append(set.KV, kv)
	}
	return set, nil
}

func (c *Coins) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := c.DriverBase.ExecDelLocal(tx, receipt, index)
	if err != nil {
		return nil, err
	}
	if receipt.GetTy() != types.ExecOk {
		return set, nil
	}
	//执行成功
	var action types.CoinsAction
	err = types.Decode(tx.GetPayload(), &action)
	if err != nil {
		panic(err)
	}
	var kv *types.KeyValue
	if action.Ty == types.CoinsActionTransfer && action.GetTransfer() != nil {
		transfer := action.GetTransfer()
		kv, err = updateAddrReciver(c.GetLocalDB(), tx.To, transfer.Amount, false)
	} else if action.Ty == types.CoinsActionWithdraw && action.GetWithdraw() != nil {
		transfer := action.GetWithdraw()
		from := account.PubKeyToAddress(tx.Signature.Pubkey).String()
		kv, err = updateAddrReciver(c.GetLocalDB(), from, transfer.Amount, false)
	}
	if err != nil {
		return set, nil
	}
	if kv != nil {
		set.KV = append(set.KV, kv)
	}
	return set, nil
}

func (c *Coins) GetAddrReciver(addr *types.ReqAddr) (types.Message, error) {
	reciver := types.Int64{}
	db := c.GetLocalDB()
	addrReciver, err := db.Get(calcAddrKey(addr.Addr))
	if addrReciver == nil || err != nil {
		return &reciver, types.ErrEmpty
	}
	err = types.Decode(addrReciver, &reciver)
	if err != nil {
		return &reciver, err
	}
	return &reciver, nil
}

func (c *Coins) Query(funcName string, params []byte) (types.Message, error) {
	if funcName == "GetAddrReciver" {
		var in types.ReqAddr
		err := types.Decode(params, &in)
		if err != nil {
			return nil, err
		}
		return c.GetAddrReciver(&in)
	} else if funcName == "GetTxsByAddr" {
		var in types.ReqAddr
		err := types.Decode(params, &in)
		if err != nil {
			return nil, err
		}
		return c.GetTxsByAddr(&in)
	}
	return nil, types.ErrActionNotSupport
}
