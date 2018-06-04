package state

import (
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/common"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	// 在StateDB中合约账户保存的键值有以下几种
	// 合约数据，前缀+合约地址，第一次生成合约时设置，后面不会发生变化
	ContractDataPrefix = "mavl-user.evm-data: "

	// 合约状态，前缀+合约地址，保存合约nonce以及其它数据，可变
	ContractStatePrefix = "mavl-user.evm-state: "

	// 注意，合约账户本身也可能有余额信息，这部分在CoinsAccount处理
)

// 合约账户对象
type ContractAccount struct {
	mdb *MemoryStateDB

	// 合约代码地址
	Addr string

	// 合约固定数据
	Data types.EVMContractData

	// 合约状态数据
	State types.EVMContractState
}

// 创建一个新的合约对象
// 注意，此时合约对象有可能已经存在也有可能不存在
// 需要通过LoadContract进行判断
func NewContractAccount(addr string, db *MemoryStateDB) *ContractAccount {
	if len(addr) == 0 || db == nil {
		log15.Error("NewContractAccount error, something is missing", "contract addr", addr, "db", db)
		return nil
	}
	ca := &ContractAccount{Addr: addr, mdb: db}
	ca.State.Storage = make(map[string][]byte)
	ca.Data.CreateTime = time.Now().UTC().UnixNano()
	return ca
}

// 获取状态数据
func (self *ContractAccount) GetState(key common.Hash) common.Hash {
	return common.BytesToHash(self.State.GetStorage()[key.Hex()])
}

// 设置状态数据
func (self *ContractAccount) SetState(key, value common.Hash) {
	self.mdb.addChange(storageChange{
		baseChange: baseChange{},
		account:    self.Addr,
		key:        key,
		prevalue:   self.GetState(key),
	})
	self.State.GetStorage()[key.Hex()] = value.Bytes()
	self.updateStorageHash()
}

func (self *ContractAccount) updateStorageHash() {
	var state = &types.EVMContractState{Suicided: self.State.Suicided, Nonce: self.State.Nonce}
	state.Storage = make(map[string][]byte)
	for k, v := range self.State.GetStorage() {
		state.Storage[k] = v
	}

	ret, err := proto.Marshal(state)
	if err != nil {
		log15.Error("marshal contract state data error", "error", err)
		return
	}

	self.State.StorageHash = common.ToHash(ret).Bytes()
}

// 从外部恢复合约数据
func (self *ContractAccount) resotreData(data []byte) {
	var content types.EVMContractData
	err := proto.Unmarshal(data, &content)
	if err != nil {
		log15.Error("read contract data error", self.Addr)
		return
	}

	self.Data = content
}

// 从外部恢复合约状态
func (self *ContractAccount) resotreState(data []byte) {
	var content types.EVMContractState
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
	self.mdb.addChange(codeChange{
		baseChange: baseChange{},
		account:    self.Addr,
		prevhash:   self.Data.GetCodeHash(),
		prevcode:   prevcode,
	})
	self.Data.Code = code
	self.Data.CodeHash = common.ToHash(code).Bytes()
}

func (self *ContractAccount) SetCreator(creator string) {
	if len(creator) == 0 {
		log15.Error("SetCreator error", "creator", creator)
		return
	}
	self.Data.Creator = creator
}

func (self *ContractAccount) SetExecName(execName string) {
	if len(execName) == 0 {
		log15.Error("SetExecName error", "execName", execName)
		return
	}
	self.Data.Name = execName
}

func (self *ContractAccount) SetAliasName(alias string) {
	if len(alias) == 0 {
		log15.Error("SetAliasName error", "aliasName", alias)
		return
	}
	self.Data.Alias = alias
}

func (self *ContractAccount) GetAliasName() string {
	return self.Data.Alias
}
func (self *ContractAccount) GetCreateTime() string {
	return time.Unix(0, self.Data.CreateTime).String()
}

func (self *ContractAccount) GetCreator() string {
	return self.Data.Creator
}

func (self *ContractAccount) GetExecName() string {
	return self.Data.Name
}

// 合约固定数据，包含合约代码，以及代码哈希
func (self *ContractAccount) GetDataKV() (kvSet []*types.KeyValue) {
	self.Data.Addr = self.Addr
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
	return &types.ReceiptLog{types.TyLogContractData, datas}
}

// 构建变更日志
func (self *ContractAccount) BuildStateLog() (log *types.ReceiptLog) {
	datas, err := proto.Marshal(&self.State)
	if err != nil || len(datas) == 0 {
		log15.Error("marshal contract state error!", "error", self.Addr)
		return
	}

	return &types.ReceiptLog{types.TyLogContractState, datas}
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
	self.mdb.addChange(nonceChange{
		baseChange: baseChange{},
		account:    self.Addr,
		prev:       self.State.GetNonce(),
	})
	self.State.Nonce = nonce
}

func (self *ContractAccount) GetNonce() uint64 {
	return self.State.GetNonce()
}
