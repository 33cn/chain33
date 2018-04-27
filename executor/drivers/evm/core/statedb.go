package core

import (
	"encoding/hex"
	"fmt"
	"github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/common"
	etypes "gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/types"
	"gitlab.33.cn/chain33/chain33/types"
	"math/big"
	"sort"
)

// 内存状态数据库，保存在区块操作时内部的数据变更操作
// 本数据库不会直接写文件，只会暂存变更记录
// 在区块打包完成后，这里的缓存变更数据会被清空（通过区块打包分别写入blockchain和statedb数据库）
// 在交易执行过程中，本数据库会暂存并变更，在交易执行结束后，会返回变更的数据集，返回给blockchain
// 执行器的Exec阶段会返回：交易收据、合约账户（包含合约地址、合约代码、合约存储信息）
// 执行器的ExecLocal阶段会返回：合约创建人和合约的关联信息
type MemoryStateDB struct {
	// 状态DB，从执行器框架传入
	StateDB db.KV

	// 本地DB，从执行器框架传入
	LocalDB db.KVDB

	// Coins账户操作对象，从执行器框架传入
	CoinsAccount *account.DB

	// 缓存账户对象
	accounts map[common.Address]*ContractAccount

	// 缓存账户余额变更
	accountsDirty map[common.Address]struct{}

	// 缓存合约账户对象变更
	contractsDirty map[common.Address]struct{}

	// 合约执行过程中退回的资金
	refund uint64

	// 存储makeLogN指令对应的日志数据
	logs    map[common.Hash][]*etypes.ContractLog
	logSize uint

	// 版本号，用于标识数据变更版本
	journal        journal
	validRevisions []revision
	reversionId    int

	// 存储sha3指令对应的数据，仅用于debug日志
	preimages map[common.Hash][]byte

	// 当前临时交易哈希和交易序号
	txHash  common.Hash
	txIndex int
}

// 版本结构，包含版本号以及当前版本包含的变更对象在变更序列中的开始序号
type revision struct {
	id           int
	journalIndex int
}

// 基于执行器框架的三个DB构建内存状态机对象
// 此对象的生命周期对应一个区块，在同一个区块内的多个交易执行时共享同一个DB对象
// 开始执行下一个区块时（执行器框架调用setEnv设置的区块高度发生变更时），会重新创建此DB对象
func NewMemoryStateDB(StateDB db.KV, LocalDB db.KVDB, CoinsAccount *account.DB) *MemoryStateDB {
	mdb := &MemoryStateDB{
		StateDB:        StateDB,
		LocalDB:        LocalDB,
		CoinsAccount:   CoinsAccount,
		accounts:       make(map[common.Address]*ContractAccount),
		accountsDirty:  make(map[common.Address]struct{}),
		contractsDirty: make(map[common.Address]struct{}),
		logs:           make(map[common.Hash][]*etypes.ContractLog),
		preimages:      make(map[common.Hash][]byte),
		refund:         0,
		txIndex:        0,
	}
	return mdb
}

// 每一个交易执行之前调用此方法，设置此交易的上下文信息
// 目前的上下文中包含交易哈希以及交易在区块中的序号
func (self *MemoryStateDB) Prepare(txHash common.Hash, txIndex int) {
	self.txHash = txHash
	self.txIndex = txIndex
}

// 创建一个新的合约账户对象
func (self *MemoryStateDB) CreateAccount(addr common.Address) {
	acc := self.GetAccount(addr)
	if acc == nil {
		// 新增账户后，如果未发生给合约转账的操作，合约账户的地址在coins那边是不存在的
		acc := NewContractAccount(addr.Str(), self)
		acc.LoadContract(self.StateDB)
		self.accounts[addr] = acc

		self.accountsDirty[addr] = struct{}{}
		self.contractsDirty[addr] = struct{}{}

		self.journal = append(self.journal, createAccountChange{baseChange: baseChange{}, account: &addr})
	}
}

// 调用Coins执行器的逻辑操作外部账户中的金额
func (self *MemoryStateDB) processBalance(addr common.Address, value *big.Int, withdraw bool) error {
	if self.CoinsAccount == nil {
		return NoCoinsAccount
	}

	accFrom := self.CoinsAccount.LoadAccount(addr.Str())
	amount := value.Int64()
	// 如果是取钱，需要把金额设成负，下面直接加就可以了
	if withdraw {
		amount = 0 - amount
	}
	if accFrom.GetBalance()+amount >= 0 {
		copyfrom := *accFrom
		accFrom.Balance = accFrom.GetBalance() + amount
		receiptBalance := &types.ReceiptAccountTransfer{&copyfrom, accFrom}
		self.CoinsAccount.SaveAccount(accFrom)

		feelog := &types.ReceiptLog{types.TyLogFee, types.Encode(receiptBalance)}
		feedata := self.CoinsAccount.GetKVSet(accFrom)

		self.journal = append(self.journal, balanceChange{
			baseChange: baseChange{},
			addr:       addr.Str(),
			amount:     amount,
			data:       feedata,
			logs:       []*types.ReceiptLog{feelog},
		})

		return nil
	}
	return types.ErrNoBalance
}

func (self *MemoryStateDB) SubBalance(addr common.Address, value *big.Int) {
	//借助coins执行器
	self.processBalance(addr, value, true)
}

func (self *MemoryStateDB) AddBalance(addr common.Address, value *big.Int) {
	//借助coins执行器
	self.processBalance(addr, value, false)
}

func (self *MemoryStateDB) GetBalance(addr common.Address) *big.Int {
	if self.CoinsAccount == nil {
		return common.Big0
	}
	ac := self.CoinsAccount.LoadAccount(addr.Str())
	if ac != nil {
		return big.NewInt(ac.Balance)
	}
	return common.Big0
}

//TODO 目前chain33中没有保留账户的nonce信息，这里临时添加到合约账户中
func (self *MemoryStateDB) GetNonce(addr common.Address) uint64 {
	acc := self.GetAccount(addr)
	if acc != nil {
		return acc.GetNonce()
	}
	return 0
}

func (self *MemoryStateDB) SetNonce(addr common.Address, nonce uint64) {
	acc := self.GetAccount(addr)
	if acc != nil {
		acc.SetNonce(nonce)
	}
}

func (self *MemoryStateDB) GetCodeHash(addr common.Address) common.Hash {
	acc := self.GetAccount(addr)
	if acc != nil {
		return common.BytesToHash(acc.Data.GetCodeHash())
	}
	return common.Hash{}
}

func (self *MemoryStateDB) GetCode(addr common.Address) []byte {
	acc := self.GetAccount(addr)
	if acc != nil {
		return acc.Data.GetCode()
	}
	return nil
}

func (self *MemoryStateDB) SetCode(addr common.Address, code []byte) {
	acc := self.GetAccount(addr)
	if acc != nil {
		acc.SetCode(code)
		self.contractsDirty[addr] = struct{}{}
	}
}

// 获取合约代码自身的大小
// 对应 EXTCODESIZE 操作码
func (self *MemoryStateDB) GetCodeSize(addr common.Address) int {
	code := self.GetCode(addr)
	if code != nil {
		return len(code)
	}
	return 0
}

// 合约自杀或SSTORE指令时，返还Gas
func (self *MemoryStateDB) AddRefund(gas uint64) {
	self.journal = append(self.journal, refundChange{baseChange: baseChange{}, prev: self.refund})
	self.refund += gas
}

func (self *MemoryStateDB) GetRefund() uint64 {
	return self.refund
}

// 从缓存中获取或加载合约账户
func (self *MemoryStateDB) GetAccount(addr common.Address) *ContractAccount {
	if acc, ok := self.accounts[addr]; ok {
		return acc
	} else {
		// 需要加载合约对象，根据是否存在合约代码来判断是否有合约对象
		contract := NewContractAccount(addr.Str(), self)
		contract.LoadContract(self.StateDB)
		if contract.Empty() {
			return nil
		}
		self.accounts[addr] = contract
		return contract
	}
}

// SLOAD 指令加载合约状态数据
func (self *MemoryStateDB) GetState(addr common.Address, key common.Hash) common.Hash {
	// 先从合约缓存中获取
	acc := self.GetAccount(addr)
	if acc != nil {
		return acc.GetState(key)
	}
	return common.Hash{}
}

// SSTORE 指令修改合约状态数据
func (self *MemoryStateDB) SetState(addr common.Address, key common.Hash, value common.Hash) {
	acc := self.GetAccount(addr)
	if acc != nil {
		acc.SetState(key, value)
	}
}

// SELFDESTRUCT 合约对象自杀
// 合约自杀后，合约对象依然存在，只是无法被调用，也无法恢复
func (self *MemoryStateDB) Suicide(addr common.Address) bool {
	acc := self.GetAccount(addr)
	if acc != nil {
		self.journal = append(self.journal, suicideChange{
			baseChange: baseChange{},
			account:    &addr,
			prev:       acc.State.GetSuicided(),
		})
		return acc.Suicide()
	}
	return false
}

// 判断此合约对象是否已经自杀
// 自杀的合约对象是不允许调用的
func (self *MemoryStateDB) HasSuicided(addr common.Address) bool {
	acc := self.GetAccount(addr)
	if acc != nil {
		return acc.HasSuicided()
	}
	return false
}

// 判断合约对象是否存在
func (self *MemoryStateDB) Exist(addr common.Address) bool {
	return self.GetAccount(addr) != nil
}

// 判断合约对象是否为空
func (self *MemoryStateDB) Empty(addr common.Address) bool {
	acc := self.GetAccount(addr)
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

// 获取最后一次成功的快照版本号
func (self *MemoryStateDB) GetLastSnapshot() int {
	return self.reversionId - 1
}

// 获取合约对象的变更日志
func (self *MemoryStateDB) GetReceiptLogs(addr common.Address, created bool) (logs []*types.ReceiptLog) {
	acc := self.GetAccount(addr)
	if acc != nil {
		logs = append(logs, acc.BuildStateLog())
		if created {
			logs = append(logs, acc.BuildDataLog())
		}
		return
	}
	return
}

// 获取本次操作所引起的状态数据变更
func (self *MemoryStateDB) GetChangedData(version int) (kvSet []*types.KeyValue, logs []*types.ReceiptLog) {
	// 从一堆快照列表中找出本次制定版本的快照索引位置
	idx := sort.Search(len(self.validRevisions), func(i int) bool {
		return self.validRevisions[i].id >= version
	})

	// 如果版本号不对，操作失败
	if idx == len(self.validRevisions) || self.validRevisions[idx].id != version {
		panic(fmt.Errorf("revision id %v cannot be reverted", version))
	}

	// 获取快照版本
	snapshot := self.validRevisions[idx].journalIndex

	// 获取中间的数据变更
	dataMap := make(map[string]*types.KeyValue)
	//for i := len(self.journal) - 1; i >= snapshot; i-- {
	for i := snapshot; i < len(self.journal); i++ {
		items := self.journal[i].getData(self)
		logs = append(logs, self.journal[i].getLog(self)...)

		// 执行去重操作
		if items != nil {
			for _, kv := range items {
				dataMap[string(kv.Key)] = kv
			}
		}
	}
	for _, value := range dataMap {
		kvSet = append(kvSet, value)
	}

	return kvSet, logs
}

// 借助coins执行器进行转账相关操作
func (self *MemoryStateDB) CanTransfer(addr common.Address, amount *big.Int) bool {
	if self.CoinsAccount == nil {
		return false
	}
	err := self.CoinsAccount.CheckTransfer(addr.Str(), "", amount.Int64())
	if err != nil {
		return false
	}
	return true
}

// 借助coins执行器进行转账相关操作
func (self *MemoryStateDB) Transfer(sender, recipient common.Address, amount *big.Int) {
	if self.CoinsAccount != nil {
		self.CoinsAccount.Transfer(sender.Str(), recipient.Str(), amount.Int64())
	}
}

// LOG0-4 指令对应的具体操作
// 生成对应的日志信息，目前这些生成的日志信息会在合约执行后打印到日志文件中
func (self *MemoryStateDB) AddLog(log *etypes.ContractLog) {
	self.journal = append(self.journal, addLogChange{txhash: self.txHash})
	log.TxHash = self.txHash
	log.Index = int(self.logSize)
	self.logs[self.txHash] = append(self.logs[self.txHash], log)
	self.logSize++
}

// 存储sha3指令对应的数据
func (self *MemoryStateDB) AddPreimage(hash common.Hash, data []byte) {
	// 目前只用于打印日志
	if _, ok := self.preimages[hash]; !ok {
		self.journal = append(self.journal, addPreimageChange{hash: hash})
		pi := make([]byte, len(data))
		copy(pi, data)
		self.preimages[hash] = pi
	}
}

// 本合约执行完毕之后打印合约生成的日志（如果有）
// 这里不保证当前区块可以打包成功，只是在执行区块中的交易时，如果交易执行成功，就会打印合约日志
func (self *MemoryStateDB) PrintLogs() {
	items, _ := self.logs[self.txHash]
	if items != nil {
		for _, item := range items {
			log15.Debug(item.String())
		}
	}
}

// 打印本区块内生成的preimages日志
func (self *MemoryStateDB) WritePreimages(number int64) {
	for k, v := range self.preimages {
		log15.Debug("Contract preimages ", "key:", k.Str(), "value:", hex.EncodeToString(v), "block height:", number)
	}
}
