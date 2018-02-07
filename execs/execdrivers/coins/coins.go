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

func (n *Coins) GetActionName(tx *types.Transaction) string {
	var action types.CoinsAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return "unknow"
	}
	if err = account.CheckAddress(tx.To); err != nil {
		return "unknow"
	}
	if action.Ty == types.CoinsActionTransfer && action.GetTransfer() != nil {
		return "transfer"
	} else if action.Ty == types.CoinsActionWithdraw && action.GetTransfer() != nil {
		return "withdraw"
	} else if action.Ty == types.CoinsActionGenesis && action.GetGenesis() != nil {
		return "genesis"
	} else {
		return "unknow"
	}
}

func (n *Coins) Exec(tx *types.Transaction, index int) (*types.Receipt, error) {
	_, err := n.ExecCommon(tx, index)
	if err != nil {
		return nil, err
	}
	var action types.CoinsAction
	err = types.Decode(tx.Payload, &action)
	if err != nil {
		return nil, err
	}
	if action.Ty == types.CoinsActionTransfer && action.GetTransfer() != nil {
		transfer := action.GetTransfer()
		from := account.From(tx).String()
		//to 是 execs 合约地址
		if execdrivers.IsExecAddress(tx.To) {
			return account.TransferToExec(n.GetDB(), from, tx.To, transfer.Amount)
		}
		return account.Transfer(n.GetDB(), from, tx.To, transfer.Amount)
	} else if action.Ty == types.CoinsActionWithdraw && action.GetWithdraw() != nil {
		withdraw := action.GetWithdraw()
		from := account.PubKeyToAddress(tx.Signature.Pubkey).String()
		//to 是 execs 合约地址
		if execdrivers.IsExecAddress(tx.To) {
			return account.TransferWithdraw(n.GetDB(), from, tx.To, withdraw.Amount)
		}
		return nil, types.ErrActionNotSupport
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

//0: all tx
//1: from tx
//2: to tx

func (n *Coins) ExecLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := n.ExecLocalCommon(tx, receipt, index)
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
		kv, err = updateAddrReciver(n.GetLocalDB(), tx.To, transfer.Amount, true)
	} else if action.Ty == types.CoinsActionWithdraw && action.GetWithdraw() != nil {
		transfer := action.GetWithdraw()
		from := account.PubKeyToAddress(tx.Signature.Pubkey).String()
		kv, err = updateAddrReciver(n.GetLocalDB(), from, transfer.Amount, true)
	} else if action.Ty == types.CoinsActionGenesis && action.GetGenesis() != nil {
		gen := action.GetGenesis()
		kv, err = updateAddrReciver(n.GetLocalDB(), tx.To, gen.Amount, true)
	}
	if err != nil {
		return set, nil
	}
	if kv != nil {
		set.KV = append(set.KV, kv)
	}
	return set, nil
}

func (n *Coins) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := n.ExecDelLocalCommon(tx, receipt, index)
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
		kv, err = updateAddrReciver(n.GetLocalDB(), tx.To, transfer.Amount, false)
	} else if action.Ty == types.CoinsActionWithdraw && action.GetWithdraw() != nil {
		transfer := action.GetWithdraw()
		from := account.PubKeyToAddress(tx.Signature.Pubkey).String()
		kv, err = updateAddrReciver(n.GetLocalDB(), from, transfer.Amount, false)
	}
	if err != nil {
		return set, nil
	}
	if kv != nil {
		set.KV = append(set.KV, kv)
	}
	return set, nil
}

func (n *Coins) GetAddrReciver(addr *types.ReqAddr) (types.Message, error) {
	reciver := types.Int64{}
	db := n.GetQueryDB()
	addrReciver := db.Get(calcAddrKey(addr.Addr))
	if addrReciver == nil {
		return &reciver, types.ErrEmpty
	}
	err := types.Decode(addrReciver, &reciver)
	if err != nil {
		return &reciver, err
	}
	return &reciver, nil
}

func (n *Coins) Query(funcName string, params []byte) (types.Message, error) {
	if funcName == "GetAddrReciver" {
		var in types.ReqAddr
		err := types.Decode(params, &in)
		if err != nil {
			return nil, err
		}
		return n.GetAddrReciver(&in)
	} else if funcName == "GetTxsByAddr" {
		var in types.ReqAddr
		err := types.Decode(params, &in)
		if err != nil {
			return nil, err
		}
		return n.GetTxsByAddr(&in)
	}
	return nil, types.ErrActionNotSupport
}
