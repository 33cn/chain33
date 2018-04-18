package core

import (
	"gitlab.33.cn/chain33/chain33/executor"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/common"
	etypes "gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/types"
	"math/big"
	"gitlab.33.cn/chain33/chain33/account"
)



// 内存状态数据库，保存在区块操作时内部的数据变更操作
// 本数据库不会直接写文件，只会暂存变更记录
// 在区块打包完成后，这里的缓存变更数据会被清空（通过区块打包分别写入blockchain和statedb数据库）
// 在交易执行过程中，本数据库会暂存并变更，在交易执行结束后，会返回变更的数据集，返回给blockchain
// 执行器的Exec阶段会返回：交易收据、合约账户（包含合约地址、合约代码、合约存储信息）
// 执行器的ExecLocal阶段会返回：合约创建人和合约的关联信息
type MemoryStateDB struct {

	statedb executor.StateDB

	localdb *executor.LocalDB

	// 缓存账户对象
	accounts      map[common.Address]*ContractAccount

	// 缓存账户余额变更
	accountsDirty map[common.Address]struct{}

	// 缓存合约账户对象变更
	contractsDirty map[common.Address]struct{}

	// TODO 退回资金
	refund uint64
}

// 创建一个新的合约账户对象
func (self *MemoryStateDB) CreateAccount(addr common.Address) {
	acc := &ContractAccount{}
	acc.Addr=addr.Str()
	acc.SetContract(true)
	acc.LoadContract(self.statedb)

	self.accounts[addr] = acc
	self.accountsDirty[addr]=struct{}{}
}

func (self *MemoryStateDB) SubBalance(addr common.Address, value *big.Int) {
	acc := self.getAccount(addr)
	if acc != nil{
		acc.SubBalance(value)
		self.accountsDirty[addr]= struct{}{}
	}
}

func (self *MemoryStateDB) AddBalance(addr common.Address, value *big.Int) {
	acc := self.getAccount(addr)
	if acc != nil{
		acc.AddBalance(value)
		self.accountsDirty[addr]= struct{}{}
	}
}

func (self *MemoryStateDB) GetBalance(addr common.Address) *big.Int {
	acc := self.getAccount(addr)
	if acc != nil{
		return big.NewInt(acc.Balance)
	}
	return common.Big0
}

//TODO 目前chain33中没有保留账户的nonce信息，这里暂时返回0
func (self *MemoryStateDB) GetNonce(addr common.Address) uint64 {
	return 0
}

func (self *MemoryStateDB) SetNonce(common.Address, uint64) {

}

func (self *MemoryStateDB) GetCodeHash(addr common.Address) common.Hash {
	acc := self.getAccount(addr)
	if acc != nil{
		return acc.codeHash
	}
	return common.Hash{}
}

func (self *MemoryStateDB) GetCode(addr common.Address) []byte {
	acc := self.getAccount(addr)
	if acc != nil{
		return acc.code
	}
	return nil
}

func (self *MemoryStateDB) SetCode(addr common.Address, code []byte) {
	acc := self.getAccount(addr)
	if acc != nil{
		acc.SetCode(code)
		self.contractsDirty[addr]= struct{}{}
	}
}

func (self *MemoryStateDB) GetCodeSize(addr common.Address) int {
	code := self.GetCode(addr)
	if code != nil{
		return len(code)
	}
	return 0
}

func (self *MemoryStateDB) AddRefund(gas uint64) {
	self.refund += gas
}

func (self *MemoryStateDB) GetRefund() uint64 {
	return self.refund
}

// 从缓存中获取账户或加载账户
func (self *MemoryStateDB) getAccount(addr common.Address) *ContractAccount {
	if acc,ok := self.accounts[addr]; ok{
		return acc
	}else{
		db := account.NewCoinsAccount()
		db.SetDB(&self.statedb)
		acc := db.LoadAccount(addr.Str())
		if acc != nil{
			contract := NewContractAccount(*acc)
			contract.LoadContract(self.statedb)
			return contract
		}
		return nil
	}
	return nil
}

func (self *MemoryStateDB) GetState(addr common.Address, key common.Hash) common.Hash {
	// 先从合约缓存中获取
	acc := self.getAccount(addr)
	if acc!= nil {
		return acc.GetState(key)
	}
	return common.Hash{}
}

func (self *MemoryStateDB) SetState(addr common.Address, key common.Hash, value common.Hash) {
	acc := self.getAccount(addr)
	acc.SetState(key,value)
}

func (self *MemoryStateDB) Suicide(common.Address) bool {

}

func (self *MemoryStateDB) HasSuicided(common.Address) bool {

}

func (self *MemoryStateDB) Exist(common.Address) bool {

}

func (self *MemoryStateDB) Empty(common.Address) bool {

}

func (self *MemoryStateDB) RevertToSnapshot(int) {

}

func (self *MemoryStateDB) Snapshot() int {

}

func (self *MemoryStateDB) AddLog(*etypes.Log) {

}

func (self *MemoryStateDB) AddPreimage(common.Hash, []byte) {

}

func (self *MemoryStateDB) ForEachStorage(common.Address, func(common.Hash, common.Hash) bool) {

}
