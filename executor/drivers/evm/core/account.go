package core

import (
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/common"
	"gitlab.33.cn/chain33/chain33/types"
	"math/big"
)

var (
	// 在StateDB中合约账户保存的键值有以下几种
	// 合约代码，键为 前缀+合约地址，值为合约的二进制代码
	ContractCodePrefix = "evm_code: "
	// 合约代码哈希，键为 前缀+合约地址，值为合约二进制代码的哈希
	ContractCodeHashPrefix = "evm_code_hash: "
	// 合约存储，键为 前缀+合约地址，值为合约存储的RLP编码
	ContractStoragePrefix = "evm_storage: "
	// 合约存储哈希，键为 前缀+合约地址，值为合约存储的RLP编码的哈希
	ContractStorageHashPrefix = "evm_storage_hash: "
	// 合约是否自杀，键为 前缀+合约地址，值为合约是否被杀
	ContractSuicidePrefix = "evm_contract_suicide: "

	// 注意，合约账户本身也有余额信息，这部分在CoinsAccount处理
)

var emptyCodeHash = common.Hash{}

type Code []byte
type Storage map[common.Hash]common.Hash

// 合约账户对象
type ContractAccount struct {
	mdb *MemoryStateDB

	// 在外部账户基础上增加特性
	types.Account

	nonce uint64

	// 是否为合约账户
	contract bool

	code     Code
	codeHash common.Hash

	storage     Storage
	storageHash common.Hash

	// 合约执行过程中的状态数据变更存储
	cachedStorage Storage
	dirtyStorage  Storage
	dirtyAccount  bool

	// 此合约是否已经自杀 （合约对象依然存在）
	suicided bool
	// 此合约是否已经被删除（合约对象依然存在）
	deleted bool
}

func NewContractAccount(acc types.Account, db *MemoryStateDB) *ContractAccount {
	ca := &ContractAccount{Account: acc}
	ca.mdb = db
	return ca
}

func (self *ContractAccount) GetState(key common.Hash) common.Hash {
	return self.cachedStorage[key]
}

func (self *ContractAccount) SetState(key, value common.Hash) {
	addr := common.StringToAddress(self.Addr)
	self.mdb.journal = append(self.mdb.journal, storageChange{
		account:  &addr,
		key:      key,
		prevalue: self.GetState(key),
	})

	self.cachedStorage[key] = value
}

func (self *ContractAccount) IsContract() bool {
	return self.contract
}

func (self *ContractAccount) SetContract(contract bool) {
	self.contract = contract
}

func (self *ContractAccount) LoadContract(db db.KV) {

	content, err := db.Get(self.getCodeKey())
	if err != nil {
		return
	}
	self.code = Code(content)
	self.contract = true

	content, err = db.Get(self.getStorageKey())
	if err != nil {
		return
	}

	self.storage.LoadFromBytes(content)
}

func (self *ContractAccount) SetCode(code []byte) {
	prevcode := self.code
	addr := common.StringToAddress(self.Addr)
	self.mdb.journal = append(self.mdb.journal, codeChange{
		account:  &addr,
		prevhash: self.codeHash.Bytes(),
		prevcode: []byte(prevcode),
	})

	self.code = code
	self.codeHash = common.BytesToHash(code)
}

// 获取和coins相关的数据
func (self *ContractAccount) getCoinsData() []*types.KeyValue {
	coins := account.NewCoinsAccount()
	// 注意，这里不用设置DB，因为只是组装传入的数据
	return coins.GetKVSet(&self.Account)

}

// 获取自杀相关的数据
func (self *ContractAccount) getSuicideData() []*types.KeyValue {
	kvSet := make([]*types.KeyValue, 4)

	kvSet = append(kvSet, &types.KeyValue{Key: self.getCodeKey(), Value: []byte("")})
	kvSet = append(kvSet, &types.KeyValue{Key: self.getStorageKey(), Value: []byte("")})
	kvSet = append(kvSet, &types.KeyValue{Key: self.getCodeHashKey(), Value: common.Hash{}.Bytes()})
	kvSet = append(kvSet, &types.KeyValue{Key: self.getStorageHashKey(), Value: common.Hash{}.Bytes()})

	kvSet = append(kvSet, &types.KeyValue{Key: self.getSuicideKey(), Value: []byte(string(self.suicided))})

	return kvSet

}

func (self *ContractAccount) SubBalance(value *big.Int) {
	addr := common.StringToAddress(self.Addr)
	self.mdb.journal = append(self.mdb.journal, balanceChange{&addr, self.Balance})

	self.Balance -= value.Int64()
}

func (self *ContractAccount) AddBalance(value *big.Int) {
	self.Balance += value.Int64()
}

func (self *ContractAccount) getCodeKey() []byte {
	return []byte(ContractCodePrefix + self.Addr)
}
func (self *ContractAccount) getStorageKey() []byte {
	return []byte(ContractStoragePrefix + self.Addr)
}
func (self *ContractAccount) getCodeHashKey() []byte {
	return []byte(ContractCodeHashPrefix + self.Addr)
}
func (self *ContractAccount) getStorageHashKey() []byte {
	return []byte(ContractStorageHashPrefix + self.Addr)
}
func (self *ContractAccount) getSuicideKey() []byte {
	return []byte(ContractSuicidePrefix + self.Addr)
}

// 获取和合约相关的数据集
func (self *ContractAccount) GetContractData() []*types.KeyValue {
	kvSet := make([]*types.KeyValue, 4)

	kvSet = append(kvSet, &types.KeyValue{Key: self.getCodeKey(), Value: self.code})
	kvSet = append(kvSet, &types.KeyValue{Key: self.getStorageKey(), Value: self.storage.ToBytes()})
	kvSet = append(kvSet, &types.KeyValue{Key: self.getCodeHashKey(), Value: self.codeHash.Bytes()})
	kvSet = append(kvSet, &types.KeyValue{Key: self.getStorageHashKey(), Value: self.getStorageHash().Bytes()})

	return kvSet
}

func (self *ContractAccount) getStorageHash() common.Hash {
	self.storageHash = common.BytesToHash(self.storage.ToBytes())
	return self.storageHash
}

func (self *ContractAccount) Suicide() bool {
	self.suicided = true
	self.Balance = 0
	self.dirtyAccount = true
	return true
}

func (self *ContractAccount) HasSuicided() bool {
	return self.suicided
}

func (self *ContractAccount) Delete() {
	self.deleted = true
}

func (self *ContractAccount) HasDeleted() bool {
	return self.deleted
}

func (self *ContractAccount) Empty() bool {
	return self.Balance == 0 && self.codeHash == emptyCodeHash
}

func (self *ContractAccount) SetNonce(nonce uint64) {
	addr := common.StringToAddress(self.Addr)

	self.mdb.journal = append(self.mdb.journal, nonceChange{
		account: &addr,
		prev:    self.nonce,
	})

	self.nonce = nonce
}

func (self *ContractAccount) GetNonce() uint64 {
	return self.nonce
}

func (st Storage) ToBytes() (ret []byte) {
	//var content []byte
	for k, v := range st {
		ret = append(ret, k.Bytes()...)
		ret = append(ret, v.Bytes()...)
	}

	return ret
}

func (st Storage) LoadFromBytes(data []byte) {
	size := len(data) / common.HashLength
	for idx := 0; idx < size; idx += 2 {
		d := data[idx*common.HashLength : (idx+2)*common.HashLength]
		key := d[:common.HashLength]
		value := d[common.HashLength:]
		st[common.BytesToHash(key)] = common.BytesToHash(value)
	}
}
