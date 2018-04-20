package core

import (
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/common"
	etypes "gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/types"
	"gitlab.33.cn/chain33/chain33/types"
	"math/big"
	"sort"
	"fmt"
)

// 内存状态数据库，保存在区块操作时内部的数据变更操作
// 本数据库不会直接写文件，只会暂存变更记录
// 在区块打包完成后，这里的缓存变更数据会被清空（通过区块打包分别写入blockchain和statedb数据库）
// 在交易执行过程中，本数据库会暂存并变更，在交易执行结束后，会返回变更的数据集，返回给blockchain
// 执行器的Exec阶段会返回：交易收据、合约账户（包含合约地址、合约代码、合约存储信息）
// 执行器的ExecLocal阶段会返回：合约创建人和合约的关联信息
type MemoryStateDB struct {
	StateDB db.KV

	LocalDB *db.KVDB

	// 缓存账户对象
	accounts map[common.Address]*ContractAccount

	// 缓存账户余额变更
	accountsDirty map[common.Address]struct{}

	// 缓存合约账户对象变更
	contractsDirty map[common.Address]struct{}

	// TODO 退回资金
	refund uint64

	// 存储makeLogN指令对应的日志数据
	logs    map[common.Hash][]*types.Log
	logSize uint

	// 版本号，用于标识数据变更版本
	journal        journal
	validRevisions []revision
	reversionId int

	// 存储sha3指令对应的数据
	preimages map[common.Hash][]byte
}

type revision struct {
	id           int
	journalIndex int
}

func NewMemoryStateDB() MemoryStateDB {
	mdb := MemoryStateDB{
		accounts:       make(map[common.Address]*ContractAccount),
		accountsDirty:  make(map[common.Address]struct{}),
		contractsDirty: make(map[common.Address]struct{}),
		logs:           make(map[common.Hash][]*types.Log),
		preimages:      make(map[common.Hash][]byte),
	}
	return mdb
}

// 创建一个新的合约账户对象
func (self *MemoryStateDB) CreateAccount(addr common.Address) {
	acc := self.getAccount(addr)
	if acc == nil {
		ac := types.Account{Addr:addr.Str()}
		acc := NewContractAccount(ac, self)

		acc.SetContract(true)
		acc.LoadContract(self.StateDB)
		self.accounts[addr] = acc

		self.accountsDirty[addr] = struct{}{}
		self.contractsDirty[addr] = struct{}{}

		self.journal = append(self.journal, createAccountChange{&addr})
	}
}

func (self *MemoryStateDB) SubBalance(addr common.Address, value *big.Int) {
	acc := self.getAccount(addr)
	if acc != nil {
		acc.SubBalance(value)
		self.accountsDirty[addr] = struct{}{}
	}
}

func (self *MemoryStateDB) AddBalance(addr common.Address, value *big.Int) {
	acc := self.getAccount(addr)
	if acc != nil {
		acc.AddBalance(value)
		self.accountsDirty[addr] = struct{}{}
	}
}

func (self *MemoryStateDB) GetBalance(addr common.Address) *big.Int {
	acc := self.getAccount(addr)
	if acc != nil {
		return big.NewInt(acc.Balance)
	}
	return common.Big0
}

//TODO 目前chain33中没有保留账户的nonce信息，这里临时添加到合约账户中
func (self *MemoryStateDB) GetNonce(addr common.Address) uint64 {
	acc := self.getAccount(addr)
	if acc != nil {
		return acc.GetNonce()
	}
	return 0
}

func (self *MemoryStateDB) SetNonce(addr common.Address, nonce uint64) {
	acc := self.getAccount(addr)
	if acc != nil {
		acc.SetNonce(nonce)
	}
}

func (self *MemoryStateDB) GetCodeHash(addr common.Address) common.Hash {
	acc := self.getAccount(addr)
	if acc != nil {
		return acc.codeHash
	}
	return common.Hash{}
}

func (self *MemoryStateDB) GetCode(addr common.Address) []byte {
	acc := self.getAccount(addr)
	if acc != nil {
		return acc.code
	}
	return nil
}

func (self *MemoryStateDB) SetCode(addr common.Address, code []byte) {
	acc := self.getAccount(addr)
	if acc != nil {
		acc.SetCode(code)
		self.contractsDirty[addr] = struct{}{}
	}
}

func (self *MemoryStateDB) GetCodeSize(addr common.Address) int {
	code := self.GetCode(addr)
	if code != nil {
		return len(code)
	}
	return 0
}

func (self *MemoryStateDB) AddRefund(gas uint64) {
	self.journal = append(self.journal, refundChange{prev: self.refund})
	self.refund += gas
}

func (self *MemoryStateDB) GetRefund() uint64 {
	return self.refund
}

// 从缓存中获取账户或加载账户
func (self *MemoryStateDB) getAccount(addr common.Address) *ContractAccount {
	if acc, ok := self.accounts[addr]; ok {
		return acc
	} else {
		db := account.NewCoinsAccount()
		db.SetDB(self.StateDB)
		acc := db.LoadAccount(addr.Str())
		if acc != nil {
			contract := NewContractAccount(*acc, self)
			contract.LoadContract(self.StateDB)
			self.accounts[addr] = contract
			return contract
		}
		return nil
	}
	return nil
}

func (self *MemoryStateDB) GetState(addr common.Address, key common.Hash) common.Hash {
	// 先从合约缓存中获取
	acc := self.getAccount(addr)
	if acc != nil {
		return acc.GetState(key)
	}
	return common.Hash{}
}

func (self *MemoryStateDB) SetState(addr common.Address, key common.Hash, value common.Hash) {
	acc := self.getAccount(addr)
	if acc != nil {
		acc.SetState(key, value)
	}
}

func (self *MemoryStateDB) Suicide(addr common.Address) bool {
	acc := self.getAccount(addr)
	if acc != nil {
		self.journal = append(self.journal, suicideChange{
			account:     &addr,
			prev:        acc.suicided,
			prevbalance: big.NewInt(acc.Balance),
		})
		return acc.Suicide()
	}
	return false
}

func (self *MemoryStateDB) HasSuicided(addr common.Address) bool {
	acc := self.getAccount(addr)
	if acc != nil {
		return acc.HasSuicided()
	}
	return false
}

func (self *MemoryStateDB) Exist(addr common.Address) bool {
	return self.getAccount(addr) != nil
}

func (self *MemoryStateDB) Empty(addr common.Address) bool {
	acc := self.getAccount(addr)
	return acc == nil || acc.Empty()
}

// 将数据状态回滚到指定快照版本（中间的版本数据将会被删除）
func (self *MemoryStateDB) RevertToSnapshot(version int) {
	// 从一堆快照列表中找出本次制定版本的快照索引位置
	idx := sort.Search(len(self.validRevisions), func(i int) bool {
		return self.validRevisions[i].id >= version
	})

	// 如果版本号不对，回滚失败
	if idx == len(self.validRevisions) || self.validRevisions[idx].id != version {
		panic(fmt.Errorf("revision id %v cannot be reverted", version))
	}

	// 获取快照版本
	snapshot := self.validRevisions[idx].journalIndex

	// 执行回滚动作
	for i := len(self.journal) - 1; i >= snapshot; i-- {
		self.journal[i].undo(self)
	}
	self.journal = self.journal[:snapshot]

	// 只保留回滚版本之前的版本数据
	self.validRevisions = self.validRevisions[:idx]

}

// 对当前的数据状态打快照，并生成快照版本号，方便后面回滚数据
func (self *MemoryStateDB) Snapshot() int {
	id := self.reversionId
	self.reversionId++
	self.validRevisions = append(self.validRevisions, revision{id, len(self.journal)})
	return id
}


func (self *MemoryStateDB) AddLog(*etypes.Log) {
	//TODO 日志记录暂不实现

}

// 存储sha3指令对应的数据
func (self *MemoryStateDB) AddPreimage(hash common.Hash, data []byte) {
	// TODO  目前只存储，暂不使用
	if _, ok := self.preimages[hash]; !ok {
		pi := make([]byte, len(data))
		copy(pi, data)
		self.preimages[hash] = pi
	}
}

// FIXME 目前此方法也业务逻辑中没有地方使用到，单纯为测试实现，后继去除
func (self *MemoryStateDB) ForEachStorage(addr common.Address, cb func(common.Hash, common.Hash) bool) {
	acc := self.getAccount(addr)
	if acc == nil {
		return
	}

	// 首先遍历缓存
	for key, value := range acc.cachedStorage {
		cb(key, value)
	}

	// 再遍历原有数据
	for key, value := range acc.storage {
		cb(key, value)
	}

}
