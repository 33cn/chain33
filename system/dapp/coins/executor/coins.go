package executor

/*
coins 是一个货币的exec。内置货币的执行器。

主要提供两种操作：
EventTransfer -> 转移资产
*/

//package none execer for unknow execer
//all none transaction exec ok, execept nofee
//nofee transaction will not pack into block

import (
	"reflect"

	"gitlab.33.cn/chain33/chain33/common/address"
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	cty "gitlab.33.cn/chain33/chain33/system/dapp/coins/types"
	"gitlab.33.cn/chain33/chain33/types"
)

//var clog = log.New("module", "execs.coins")

func Init() {
	drivers.Register(GetName(), newCoins, 0)
	InitType()
}

func GetName() string {
	return newCoins().GetName()
}

type Coins struct {
	drivers.DriverBase
}

func newCoins() drivers.Driver {
	c := &Coins{}
	c.SetChild(c)
	return c
}

func (c *Coins) GetName() string {
	return "coins"
}

func (c *Coins) CheckTx(tx *types.Transaction, index int) error {
	return nil
}

func (c *Coins) Exec_Transfer(transfer *cty.CoinsTransfer, tx *types.Transaction, index int) (*types.Receipt, error) {
	from := tx.From()
	//to 是 execs 合约地址
	if drivers.IsDriverAddress(tx.GetRealToAddr(), c.GetHeight()) {
		return c.GetCoinsAccount().TransferToExec(from, tx.GetRealToAddr(), transfer.Amount)
	}
	return c.GetCoinsAccount().Transfer(from, tx.GetRealToAddr(), transfer.Amount)
}

func (c *Coins) Exec_TransferToExec(transfer *cty.CoinsTransferToExec, tx *types.Transaction, index int) (*types.Receipt, error) {
	if c.GetHeight() < types.ForkV12TransferExec {
		return nil, types.ErrActionNotSupport
	}
	from := tx.From()
	//to 是 execs 合约地址
	if !isExecAddrMatch(transfer.ExecName, tx.GetRealToAddr()) {
		return nil, types.ErrToAddrNotSameToExecAddr
	}
	return c.GetCoinsAccount().TransferToExec(from, tx.GetRealToAddr(), transfer.Amount)
}

func (c *Coins) Exec_Withdraw(withdraw *cty.CoinsWithdraw, tx *types.Transaction, index int) (*types.Receipt, error) {
	if !types.IsMatchFork(c.GetHeight(), types.ForkV16Withdraw) {
		withdraw.ExecName = ""
	}
	from := tx.From()
	//to 是 execs 合约地址
	if drivers.IsDriverAddress(tx.GetRealToAddr(), c.GetHeight()) || isExecAddrMatch(withdraw.ExecName, tx.GetRealToAddr()) {
		return c.GetCoinsAccount().TransferWithdraw(from, tx.GetRealToAddr(), withdraw.Amount)
	}
	return nil, types.ErrActionNotSupport
}

func (c *Coins) Exec_Genesis(genesis *cty.CoinsGenesis, tx *types.Transaction, index int) (*types.Receipt, error) {
	if c.GetHeight() == 0 {
		if drivers.IsDriverAddress(tx.GetRealToAddr(), c.GetHeight()) {
			return c.GetCoinsAccount().GenesisInitExec(genesis.ReturnAddress, genesis.Amount, tx.GetRealToAddr())
		}
		return c.GetCoinsAccount().GenesisInit(tx.GetRealToAddr(), genesis.Amount)
	} else {
		return nil, types.ErrReRunGenesis
	}
}

func (c *Coins) GetPayloadValue() types.Message {
	return &cty.CoinsAction{}
}

func (c *Coins) GetTypeMap() map[string]int32 {
	return map[string]int32{
		"Transfer":       cty.CoinsActionTransfer,
		"TransferToExec": cty.CoinsActionTransferToExec,
		"Withdraw":       cty.CoinsActionWithdraw,
		"Genesis":        cty.CoinsActionGenesis,
	}
}

func (c *Coins) GetFuncMap() map[string]reflect.Method {
	return funclist
}

func isExecAddrMatch(name string, to string) bool {
	toaddr := address.ExecAddress(name)
	return toaddr == to
}

func (c *Coins) ExecLocal_Transfer(transfer *cty.CoinsTransfer, tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	kv, err := updateAddrReciver(c.GetLocalDB(), tx.GetRealToAddr(), transfer.Amount, true)
	if err != nil {
		return nil, err
	}
	return &types.LocalDBSet{KV: []*types.KeyValue{kv}}, nil
}

func (c *Coins) ExecLocal_TransferToExec(transfer *cty.CoinsTransferToExec, tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	kv, err := updateAddrReciver(c.GetLocalDB(), tx.GetRealToAddr(), transfer.Amount, true)
	if err != nil {
		return nil, err
	}
	return &types.LocalDBSet{KV: []*types.KeyValue{kv}}, nil
}

func (c *Coins) ExecLocal_Withdraw(withdraw *cty.CoinsWithdraw, tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	from := tx.From()
	kv, err := updateAddrReciver(c.GetLocalDB(), from, withdraw.Amount, true)
	if err != nil {
		return nil, err
	}
	return &types.LocalDBSet{KV: []*types.KeyValue{kv}}, nil
}

func (c *Coins) ExecLocal_Genesis(gen *cty.CoinsGenesis, tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	kv, err := updateAddrReciver(c.GetLocalDB(), tx.GetRealToAddr(), gen.Amount, true)
	if err != nil {
		return nil, err
	}
	return &types.LocalDBSet{KV: []*types.KeyValue{kv}}, nil
}

func (c *Coins) ExecDelLocal_Transfer(transfer *cty.CoinsTransfer, tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	kv, err := updateAddrReciver(c.GetLocalDB(), tx.GetRealToAddr(), transfer.Amount, false)
	if err != nil {
		return nil, err
	}
	return &types.LocalDBSet{KV: []*types.KeyValue{kv}}, nil
}

func (c *Coins) ExecDelLocal_TransferToExec(transfer *cty.CoinsTransferToExec, tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	kv, err := updateAddrReciver(c.GetLocalDB(), tx.GetRealToAddr(), transfer.Amount, false)
	if err != nil {
		return nil, err
	}
	return &types.LocalDBSet{KV: []*types.KeyValue{kv}}, nil
}

func (c *Coins) ExecDelLocal_Withdraw(withdraw *cty.CoinsWithdraw, tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	from := tx.From()
	kv, err := updateAddrReciver(c.GetLocalDB(), from, withdraw.Amount, false)
	if err != nil {
		return nil, err
	}
	return &types.LocalDBSet{KV: []*types.KeyValue{kv}}, nil
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
	} else if funcName == "GetPrefixCount" {
		var in types.ReqKey
		err := types.Decode(params, &in)
		if err != nil {
			return nil, err
		}
		return c.GetPrefixCount(&in)
	} else if funcName == "GetAddrTxsCount" {
		var in types.ReqKey
		err := types.Decode(params, &in)
		if err != nil {
			return nil, err
		}
		return c.GetAddrTxsCount(&in)
	}
	return nil, types.ErrActionNotSupport
}
