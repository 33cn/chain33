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
	drivers "github.com/33cn/chain33/system/dapp"
	"github.com/33cn/chain33/types"
)

//var clog = log.New("module", "execs.coins")
var driverName = "coins"

func Init(name string, sub []byte) {
	if name != driverName {
		panic("system dapp can't be rename")
	}
	drivers.Register(driverName, newCoins, types.GetDappFork(driverName, "Enable"))
}

//初始化过程比较重量级，有很多reflact, 所以弄成全局的
func init() {
	ety := types.LoadExecutorType(driverName)
	ety.InitFuncList(types.ListMethod(&Coins{}))
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
	c.SetExecutorType(types.LoadExecutorType(driverName))
	return c
}

func (c *Coins) GetDriverName() string {
	return driverName
}

func (c *Coins) CheckTx(tx *types.Transaction, index int) error {
	return nil
}

//coins 合约 运行 ticket 合约的挖矿交易
func (c *Coins) IsFriend(myexec, writekey []byte, othertx *types.Transaction) bool {
	//step1 先判定自己合约的权限
	if !c.AllowIsSame(myexec) {
		return false
	}
	//step2 判定 othertx 的 执行器名称(只允许主链，并且是挖矿的行为)
	if string(othertx.Execer) == "ticket" && othertx.ActionName() == "miner" {
		return true
	}
	return false
}
