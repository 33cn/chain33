package core

import (
	"gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/common"
	"gitlab.33.cn/chain33/chain33/types"
	"encoding/binary"
	"gitlab.33.cn/chain33/chain33/common/crypto"
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
	// 合约Nonce，键为 前缀+合约地址，值为合约Nonce
	ContractNoncePrefix = "evm_contract_nonce: "

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

	// 合约执行过程中的状态数据变更存储
	// 与合约原有状态的存储合并，因为当前是存储在同一个键值之下的，暂不分开
	storage     Storage
	storageHash common.Hash

	dirtyStorage  bool
	dirtyAccount  bool

	// 此合约是否已经自杀 （合约对象依然存在）
	suicided bool
	// 此合约是否已经被删除（合约对象依然存在）
	deleted bool
}

func NewContractAccount(acc types.Account, db *MemoryStateDB) *ContractAccount {
	ca := &ContractAccount{Account: acc}
	ca.mdb = db

	ca.storage = NewStorage()

	return ca
}

func (self *ContractAccount) GetState(key common.Hash) common.Hash {
	return self.storage[key]
}

func (self *ContractAccount) SetState(key, value common.Hash) {
	addr := common.StringToAddress(self.Addr)
	self.mdb.journal = append(self.mdb.journal, storageChange{
		account:  &addr,
		key:      key,
		prevalue: self.GetState(key),
	})

	self.storage[key] = value
	self.dirtyStorage = true
}

func (self *ContractAccount) IsContract() bool {
	return self.contract
}

func (self *ContractAccount) SetContract(contract bool) {
	self.contract = contract
}

func (self *ContractAccount) LoadContract(db db.KV) {

	content, err := db.Get(self.GetCodeKey())
	if err != nil {
		return
	}
	self.code = Code(content)
	self.contract = true

	content, err = db.Get(self.GetStorageKey())
	if err != nil {
		return
	}

	self.storage.LoadFromBytes(content)

	content, err = db.Get(self.GetCodeHashKey())
	if err != nil {
		return
	}
	// FIXME 这里考虑增加校验，再计算一次哈希
	self.codeHash = common.BytesToHash(content)


	content, err = db.Get(self.GetStorageHashKey())
	if err != nil {
		return
	}
	// FIXME 这里考虑增加校验，再计算一次哈希
	self.storageHash = common.BytesToHash(content)

	content, err = db.Get(self.GetNonceKey())
	if err != nil {
		return
	}
	self.nonce = Byte2Int(content)

	content, err = db.Get(self.GetSuicideKey())
	if err != nil {
		return
	}
	self.suicided = Byte2Bool(content)
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
	self.codeHash = common.BytesToHash(crypto.Sha256(code))
}

func Bool2Byte(value bool) byte {
	if value {
		return 1
	}
	return 0
}

func Byte2Bool(value []byte) bool {
	if len(value) != 1 || len(value) ==0{
		return false
	}
	return true
}

func Int2Byte(value uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, value)
	return b
}

func Byte2Int(value []byte) uint64 {
	return  binary.BigEndian.Uint64(value)
}

// 获取自杀相关的数据
// 合约自杀时使用
func (self *ContractAccount) getSuicideData() (kvSet []*types.KeyValue) {
	kvSet = append(kvSet, &types.KeyValue{Key: self.GetCodeKey(), Value: []byte("")})
	kvSet = append(kvSet, &types.KeyValue{Key: self.GetStorageKey(), Value: []byte("")})
	kvSet = append(kvSet, &types.KeyValue{Key: self.GetCodeHashKey(), Value: common.Hash{}.Bytes()})
	kvSet = append(kvSet, &types.KeyValue{Key: self.GetStorageHashKey(), Value: common.Hash{}.Bytes()})
	kvSet = append(kvSet, &types.KeyValue{Key: self.GetSuicideKey(), Value: []byte{Bool2Byte(self.suicided)}})
	return kvSet

}


// 获取和合约相关的数据集
// 用于合约创建时的场景
func (self *ContractAccount) GetContractData() (kvSet []*types.KeyValue) {
	kvSet = append(kvSet, self.GetChangeData()...)
	kvSet = append(kvSet, &types.KeyValue{Key: self.GetCodeKey(), Value: self.code})
	kvSet = append(kvSet, &types.KeyValue{Key: self.GetCodeHashKey(), Value: self.codeHash.Bytes()})

	return kvSet
}

// 获取和合约相关的数据集
// 用于合约创建后的调用场景，这时只会有数据和状态变更，代码不会变更
func (self *ContractAccount) GetChangeData() (kvSet []*types.KeyValue) {
	kvSet = append(kvSet, &types.KeyValue{Key: self.GetStorageKey(), Value: self.storage.ToBytes()})
	kvSet = append(kvSet, &types.KeyValue{Key: self.GetStorageHashKey(), Value: self.getStorageHash().Bytes()})
	kvSet = append(kvSet, &types.KeyValue{Key: self.GetNonceKey(), Value: Int2Byte(self.nonce)})

	return kvSet
}


func (self *ContractAccount) getStorageHash() common.Hash {
	self.storageHash = common.BytesToHash(self.storage.ToBytes())
	return self.storageHash
}

func (self *ContractAccount) GetCodeKey() []byte {
	return []byte(ContractCodePrefix + self.Addr)
}
func (self *ContractAccount) GetStorageKey() []byte {
	return []byte(ContractStoragePrefix + self.Addr)
}
func (self *ContractAccount) GetCodeHashKey() []byte {
	return []byte(ContractCodeHashPrefix + self.Addr)
}
func (self *ContractAccount) GetStorageHashKey() []byte {
	return []byte(ContractStorageHashPrefix + self.Addr)
}
func (self *ContractAccount) GetSuicideKey() []byte {
	return []byte(ContractSuicidePrefix + self.Addr)
}
func (self *ContractAccount) GetNonceKey() []byte {
	return []byte(ContractNoncePrefix + self.Addr)
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

func NewStorage() Storage {
	data := make(map[common.Hash]common.Hash)
	return Storage(data)
}