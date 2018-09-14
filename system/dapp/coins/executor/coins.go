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
