package core

import (
	"gitlab.33.cn/chain33/chain33/executor"
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
	// 合约存储，键为 前缀+合约地址，值为合约存储的RLP编码的哈希
	ContractStorageHashPrefix = "evm_storage_hash: "

	// 注意，合约账户本身也有余额信息，这部分在CoinsAccount处理
)

type Code []byte
type Storage map[common.Hash]common.Hash

// 合约账户对象
type ContractAccount struct {
	// 在外部账户基础上增加特性
	types.Account

	// 是否为合约账户
	contract bool

	code     Code
	codeHash common.Hash

	storage Storage

	// 合约执行过程中的状态数据变更存储
	cachedStorage Storage
	dirtyStorage  Storage
}

func NewContractAccount(acc types.Account) *ContractAccount {
	return &ContractAccount{acc, false, nil, nil, nil, nil, nil}
}

func (self *ContractAccount) GetState(key common.Hash) common.Hash {
	return self.cachedStorage[key]
}

func (self *ContractAccount) SetState(key, value common.Hash) {
	self.cachedStorage[key] = value
}

func (self *ContractAccount) IsContract() bool {
	return self.contract
}

func (self *ContractAccount) SetContract(contract bool) {
	self.contract = contract
}

func (self *ContractAccount) LoadContract(db executor.StateDB) {

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
	self.code = code
	self.codeHash = common.BytesToHash(code)
}

func (self *ContractAccount) SubBalance(value *big.Int) {
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
