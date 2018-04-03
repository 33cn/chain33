package token

import (
	"fmt"
	"gitlab.33.cn/chain33/chain33/account"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/types"
)

func (t *token) ExecTransWithdraw(accountDB *account.AccountDB, tx *types.Transaction, action *types.TokenAction, index int) (*types.Receipt, error) {
	_, err := t.DriverBase.Exec(tx, index)
	if err != nil {
		return nil, err
	}

	if (action.Ty == types.ActionTransfer) && action.GetTransfer() != nil {
		transfer := action.GetTransfer()
		from := account.From(tx).String()
		//to 是 execs 合约地址
		if t.GetExecDriver().IsDriverAddress(tx.To) {
			return accountDB.TransferToExec(from, tx.To, transfer.Amount)
		}
		return accountDB.Transfer(from, tx.To, transfer.Amount)
	} else if (action.Ty == types.ActionWithdraw) && action.GetWithdraw() != nil {
		withdraw := action.GetWithdraw()
		from := account.PubKeyToAddress(tx.Signature.Pubkey).String()
		//to 是 execs 合约地址
		if t.GetExecDriver().IsDriverAddress(tx.To) {
			return accountDB.TransferWithdraw(from, tx.To, withdraw.Amount)
		}
		return nil, types.ErrActionNotSupport
	} else if (action.Ty == types.ActionGenesis) && action.GetGenesis() != nil {
		genesis := action.GetGenesis()
		if t.GetHeight() == 0 {
			if t.GetExecDriver().IsDriverAddress(tx.To) {
				return accountDB.GenesisInitExec(genesis.ReturnAddress, genesis.Amount, tx.To)
			}
			return accountDB.GenesisInit(tx.To, genesis.Amount)
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

func (t *token) ExecLocalTransWithdraw(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := t.DriverBase.ExecLocal(tx, receipt, index)
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
	if action.Ty == types.ActionTransfer && action.GetTransfer() != nil {
		transfer := action.GetTransfer()
		kv, err = updateAddrReciver(t.GetLocalDB(), transfer.Cointoken, tx.To, transfer.Amount, true)
	} else if action.Ty == types.ActionWithdraw && action.GetWithdraw() != nil {
		withdraw := action.GetWithdraw()
		from := account.PubKeyToAddress(tx.Signature.Pubkey).String()
		kv, err = updateAddrReciver(t.GetLocalDB(), withdraw.Cointoken, from, withdraw.Amount, true)
	} else if action.Ty == types.ActionGenesis && action.GetGenesis() != nil {
		gen := action.GetGenesis()
		kv, err = updateAddrReciver(t.GetLocalDB(), "token", tx.To, gen.Amount, true)
	}
	if err != nil {
		return set, nil
	}
	if kv != nil {
		set.KV = append(set.KV, kv)
	}
	return set, nil
}

func (t *token) ExecDelLocalLocalTransWithdraw(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set, err := t.DriverBase.ExecDelLocal(tx, receipt, index)
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
	if action.Ty == types.ActionTransfer && action.GetTransfer() != nil {
		transfer := action.GetTransfer()
		kv, err = updateAddrReciver(t.GetLocalDB(), transfer.Cointoken, tx.To, transfer.Amount, false)
	} else if action.Ty == types.ActionWithdraw && action.GetWithdraw() != nil {
		withdraw := action.GetWithdraw()
		from := account.PubKeyToAddress(tx.Signature.Pubkey).String()
		kv, err = updateAddrReciver(t.GetLocalDB(), withdraw.Cointoken, from, withdraw.Amount, false)
	}
	if err != nil {
		return set, nil
	}
	if kv != nil {
		set.KV = append(set.KV, kv)
	}
	return set, nil
}

//存储地址上收币的信息
func calcAddrKey(token string, addr string) []byte {
	return []byte(fmt.Sprintf("token:%s-Addr:%s", token, addr))
}

func getAddrReciverKV(token string, addr string, reciverAmount int64) *types.KeyValue {
	reciver := &types.Int64{reciverAmount}
	amountbytes := types.Encode(reciver)
	kv := &types.KeyValue{calcAddrKey(token, addr), amountbytes}
	return kv
}

func getAddrReciver(db dbm.KVDB, token string, addr string) (int64, error) {
	reciver := types.Int64{}
	addrReciver, err := db.Get(calcAddrKey(token, addr))
	if err != nil && err != types.ErrNotFound {
		return 0, err
	}
	if len(addrReciver) == 0 {
		return 0, nil
	}
	err = types.Decode(addrReciver, &reciver)
	if err != nil {
		return 0, err
	}
	return reciver.Data, nil
}

func setAddrReciver(db dbm.KVDB, token string, addr string, reciverAmount int64) error {
	kv := getAddrReciverKV(addr, token, reciverAmount)
	return db.Set(kv.Key, kv.Value)
}

func updateAddrReciver(cachedb dbm.KVDB, token string, addr string, amount int64, isadd bool) (*types.KeyValue, error) {
	recv, err := getAddrReciver(cachedb, token, addr)
	if err != nil && err != types.ErrNotFound {
		return nil, err
	}
	if isadd {
		recv += amount
	} else {
		recv -= amount
	}
	setAddrReciver(cachedb, token, addr, recv)
	//keyvalue
	return getAddrReciverKV(token, addr, recv), nil
}
