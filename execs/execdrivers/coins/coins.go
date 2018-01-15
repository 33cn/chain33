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
	"code.aliyun.com/chain33/chain33/execs/execdrivers"
	"code.aliyun.com/chain33/chain33/types"
	log "github.com/inconshreveable/log15"
)

var clog = log.New("module", "execs.coins")

func init() {
	execdrivers.Register("coins", newCoins())
}

type Coins struct {
	execdrivers.ExecBase
}

func newCoins() *Coins {
	c := &Coins{}
	c.SetChild(c)
	return c
}

func (n *Coins) GetName() string {
	return "coins"
}

func (n *Coins) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
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
		from := account.From(tx).String()
		//to 是 execs 合约地址
		if execdrivers.IsExecAddress(tx.To) {
			return account.TransferToExec(n.GetDB(), from, tx.To, transfer.Amount)
		}
		return account.Transfer(n.GetDB(), from, tx.To, transfer.Amount)
	} else if action.Ty == types.CoinsActionGenesis && action.GetGenesis() != nil {
		genesis := action.GetGenesis()
		if n.GetHeight() == 0 {
			if execdrivers.IsExecAddress(tx.To) {
				return account.GenesisInitExec(n.GetDB(), genesis.ReturnAddress, genesis.Amount, tx.To)
			}
			return account.GenesisInit(n.GetDB(), tx.To, genesis.Amount)
		} else {
			return nil, types.ErrReRunGenesis
		}
	} else {
		return nil, types.ErrActionNotSupport
	}
}
