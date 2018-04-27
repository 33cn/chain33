package core

import (
	"github.com/golang/protobuf/proto"
	"github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/common"
	ctypes "gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/types"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	// 在StateDB中合约账户保存的键值有以下几种
	// 合约数据，前缀+合约地址，第一次生成合约时设置，后面不会发生变化
	ContractDataPrefix = "evm_data: "

	// 合约状态，前缀+合约地址，保存合约nonce以及其它数据，可变
	ContractStatePrefix = "evm_state: "

	// 注意，合约账户本身也可能有余额信息，这部分在CoinsAccount处理
)

var emptyCodeHash = common.Hash{}

// 合约账户对象
type ContractAccount struct {
	mdb *MemoryStateDB

	// 在外部账户基础上增加特性
	Addr string

	// 合约固定数据
	Data ctypes.ContractData

	// 合约状态数据
	State ctypes.ContractState
}

// 创建一个新的合约对象
// 注意，此时合约对象有可能已经存在也有可能不存在
// 需要通过LoadContract进行判断
func NewContractAccount(addr string, db *MemoryStateDB) *ContractAccount {
	ca := &ContractAccount{Addr: addr}
	ca.mdb = db
	ca.State.Storage = make(map[string][]byte)
	return ca
}

func byte2Hash(item []byte) common.Hash {
	return common.BytesToHash(item)
}

func hash2String(item common.Hash) string {
	return string(item.Bytes())
}

// 获取状态数据
func (self *ContractAccount) GetState(key common.Hash) common.Hash {
	return byte2Hash(self.State.GetStorage()[string(key.Bytes())])
}

// 设置状态数据
func (self *ContractAccount) SetState(key, value common.Hash) {
	addr := common.StringToAddress(self.Addr)
	self.mdb.journal = append(self.mdb.journal, storageChange{
		baseChange: baseChange{},
		account:    &addr,
		key:        key,
		prevalue:   self.GetState(key),
	})

	self.State.GetStorage()[hash2String(key)] = value.Bytes()
}

// 从外部恢复合约数据
func (self *ContractAccount) resotreData(data []byte) {
	var content ctypes.ContractData
	err := proto.Unmarshal(data, &content)
	if err != nil {
		log15.Error("read contract data error", self.Addr)
		return
	}

	self.Data = content
}

// 从外部恢复合约状态
func (self *ContractAccount) resotreState(data []byte) {
	var content ctypes.ContractState
	err := proto.Unmarshal(data, &content)
	if err != nil {
		log15.Error("read contract state error", self.Addr)
		return
	}

	self.State = content

	if self.State.Storage == nil {
		self.State.Storage = make(map[string][]byte)
	}
}

// 从数据库中加载合约信息（在只有合约地址的情况下）
func (self *ContractAccount) LoadContract(db db.KV) {
	// 加载代码数据
	data, err := db.Get(self.GetDataKey())
	if err != nil {
		return
	}
	self.resotreData(data)

	// 加载状态数据
	data, err = db.Get(self.GetStateKey())
	if err != nil {
		return
	}
	self.resotreState(data)
}

// 设置合约二进制代码
// 会同步生成代码哈希
func (self *ContractAccount) SetCode(code []byte) {
	prevcode := self.Data.GetCode()
	addr := common.StringToAddress(self.Addr)
	self.mdb.journal = append(self.mdb.journal, codeChange{
		baseChange: baseChange{},
		account:    &addr,
		prevhash:   self.Data.GetCodeHash(),
		prevcode:   []byte(prevcode),
	})

	self.Data.Code = code
	self.Data.CodeHash = crypto.Sha256(code)
}

// 合约固定数据，包含合约代码，以及代码哈希
func (self *ContractAccount) GetDataKV() (kvSet []*types.KeyValue) {
	datas, err := proto.Marshal(&self.Data)
	if err != nil {
		log15.Error("marshal contract data error!", self.Addr)
		return
	}
	kvSet = append(kvSet, &types.KeyValue{Key: self.GetDataKey(), Value: datas})
	return
}

// 获取合约状态数据，包含nonce、是否自杀、存储哈希、存储数据
func (self *ContractAccount) GetStateKV() (kvSet []*types.KeyValue) {
	datas, err := proto.Marshal(&self.State)
	if err != nil {
		log15.Error("marshal contract state error!", self.Addr)
		return
	}
	kvSet = append(kvSet, &types.KeyValue{Key: self.GetStateKey(), Value: datas})
	return
}

// 构建变更日志
func (self *ContractAccount) BuildDataLog() (log *types.ReceiptLog) {
	datas, err := proto.Marshal(&self.Data)
	if err != nil {
		log15.Error("marshal contract state error!", self.Addr)
		return
	}
	return &types.ReceiptLog{ctypes.TyLogContractData, datas}
}

// 构建变更日志
func (self *ContractAccount) BuildStateLog() (log *types.ReceiptLog) {
	datas, err := proto.Marshal(&self.State)
	if err != nil {
		log15.Error("marshal contract state error!", self.Addr)
		return
	}

	return &types.ReceiptLog{ctypes.TyLogContractState, datas}
}


func (self *ContractAccount) GetDataKey() []byte {
	return []byte(ContractDataPrefix + self.Addr)
}
func (self *ContractAccount) GetStateKey() []byte {
	return []byte(ContractStatePrefix + self.Addr)
}

func (self *ContractAccount) Suicide() bool {
	self.State.Suicided = true
	return true
}

func (self *ContractAccount) HasSuicided() bool {
	return self.State.GetSuicided()
}

func (self *ContractAccount) Empty() bool {
	return self.Data.GetCodeHash() == nil || len(self.Data.GetCodeHash()) == 0
}

func (self *ContractAccount) SetNonce(nonce uint64) {
	addr := common.StringToAddress(self.Addr)

	self.mdb.journal = append(self.mdb.journal, nonceChange{
		baseChange: baseChange{},
		account:    &addr,
		prev:       self.State.GetNonce(),
	})

	self.State.Nonce = nonce
}

func (self *ContractAccount) GetNonce() uint64 {
	return self.State.GetNonce()
}
