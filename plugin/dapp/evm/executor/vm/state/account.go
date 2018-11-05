package state

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/common"
	evmtypes "gitlab.33.cn/chain33/chain33/plugin/dapp/evm/types"
	"gitlab.33.cn/chain33/chain33/types"
)

// 合约账户对象
type ContractAccount struct {
	mdb *MemoryStateDB

	// 合约代码地址
	Addr string

	// 合约固定数据
	Data evmtypes.EVMContractData

	// 合约状态数据
	State evmtypes.EVMContractState

	// 当前的状态数据缓存
	stateCache map[string]common.Hash
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
	ca.stateCache = make(map[string]common.Hash)
	return ca
}

// 获取状态数据；
// 获取数据分为两层，一层是从当前的缓存中获取，如果获取不到，再从localdb中获取
func (self *ContractAccount) GetState(key common.Hash) common.Hash {
	// 从ForkV19开始，状态数据使用单独的KEY存储
	if types.IsDappFork(self.mdb.blockHeight, "evm", "ForkEVMState") {
		if val, ok := self.stateCache[key.Hex()]; ok {
			return val
		}
		keyStr := getStateItemKey(self.Addr, key.Hex())
		// 如果缓存中取不到数据，则只能到本地数据库中查询
		val, err := self.mdb.LocalDB.Get([]byte(keyStr))
		if err != nil {
			log15.Debug("GetState error!", "key", key, "error", err)
			return common.Hash{}
		}
		valHash := common.BytesToHash(val)
		self.stateCache[key.Hex()] = valHash
		return valHash
	} else {
		return common.BytesToHash(self.State.GetStorage()[key.Hex()])
	}
}

// 设置状态数据
func (self *ContractAccount) SetState(key, value common.Hash) {
	self.mdb.addChange(storageChange{
		baseChange: baseChange{},
		account:    self.Addr,
		key:        key,
		prevalue:   self.GetState(key),
	})
	if types.IsDappFork(self.mdb.blockHeight, "evm", "ForkEVMState") {
		self.stateCache[key.Hex()] = value
		//需要设置到localdb中，以免同一个区块中同一个合约多次调用时，状态数据丢失
		keyStr := getStateItemKey(self.Addr, key.Hex())
		self.mdb.LocalDB.Set([]byte(keyStr), value.Bytes())
	} else {
		self.State.GetStorage()[key.Hex()] = value.Bytes()
		self.updateStorageHash()
	}
}

// 从原有的存储在一个对象，将状态数据分散存储到多个KEY，保证合约可以支撑大量状态数据
func (self *ContractAccount) TransferState() {
	if len(self.State.Storage) > 0 {
		storage := self.State.Storage
		// 为了保证不会造成新、旧数据并存的情况，需要将旧的状态数据清空
		self.State.Storage = make(map[string][]byte)
		self.State.StorageHash = common.ToHash([]byte{}).Bytes()

		// 从旧的区块迁移状态数据到新的区块，模拟状态数据变更的操作
		for key, value := range storage {
			self.SetState(common.BytesToHash(common.FromHex(key)), common.BytesToHash(value))
		}
		// 更新本合约的状态数据（删除旧的map存储信息）
		self.mdb.UpdateState(self.Addr)
		return
	}
}

func (self *ContractAccount) updateStorageHash() {
	// 从ForkV20开始，状态数据使用单独KEY存储
	if types.IsDappFork(self.mdb.blockHeight, "evm", "ForkEVMState") {
		return
	}
	var state = &evmtypes.EVMContractState{Suicided: self.State.Suicided, Nonce: self.State.Nonce}
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
	var content evmtypes.EVMContractData
	err := proto.Unmarshal(data, &content)
	if err != nil {
		log15.Error("read contract data error", self.Addr)
		return
	}

	self.Data = content
}

// 从外部恢复合约状态
func (self *ContractAccount) resotreState(data []byte) {
	var content evmtypes.EVMContractState
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
		log15.Error("marshal contract data error!", "addr", self.Addr, "error", err)
		return
	}
	kvSet = append(kvSet, &types.KeyValue{Key: self.GetDataKey(), Value: datas})
	return
}

// 获取合约状态数据，包含nonce、是否自杀、存储哈希、存储数据
func (self *ContractAccount) GetStateKV() (kvSet []*types.KeyValue) {
	datas, err := proto.Marshal(&self.State)
	if err != nil {
		log15.Error("marshal contract state error!", "addr", self.Addr, "error", err)
		return
	}
	kvSet = append(kvSet, &types.KeyValue{Key: self.GetStateKey(), Value: datas})
	return
}

// 构建变更日志
func (self *ContractAccount) BuildDataLog() (log *types.ReceiptLog) {
	datas, err := proto.Marshal(&self.Data)
	if err != nil {
		log15.Error("marshal contract data error!", "addr", self.Addr, "error", err)
		return
	}
	return &types.ReceiptLog{evmtypes.TyLogContractData, datas}
}

// 构建变更日志
func (self *ContractAccount) BuildStateLog() (log *types.ReceiptLog) {
	datas, err := proto.Marshal(&self.State)
	if err != nil {
		log15.Error("marshal contract state log error!", "addr", self.Addr, "error", err)
		return
	}

	return &types.ReceiptLog{evmtypes.TyLogContractState, datas}
}

func (self *ContractAccount) GetDataKey() []byte {
	return []byte("mavl-" + evmtypes.ExecutorName + "-data: " + self.Addr)
}

func (self *ContractAccount) GetStateKey() []byte {
	return []byte("mavl-" + evmtypes.ExecutorName + "-state: " + self.Addr)
}

// 这份数据是存在LocalDB中的
func getStateItemKey(addr, key string) string {
	return fmt.Sprintf("LODB-"+evmtypes.ExecutorName+"-state:%v:%v", addr, key)
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
