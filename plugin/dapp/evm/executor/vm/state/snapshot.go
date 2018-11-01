package state

import (
	"sort"

	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/common"
	evmtypes "gitlab.33.cn/chain33/chain33/plugin/dapp/evm/types"
	"gitlab.33.cn/chain33/chain33/types"
)

// 数据状态变更接口
// 所有的数据状态变更事件实现此接口，并且封装各自的变更数据以及回滚动作
// 在调用合约时（具体的Tx执行时），会根据操作生成对应的变更对象并缓存下来
// 如果合约执行出错，会按生成顺序的倒序，依次调用变更对象的回滚接口进行数据回滚，并同步删除变更对象缓存
// 如果合约执行成功，会按生成顺序的郑旭，依次调用变更对象的数据和日志变更记录，回传给区块链
type DataChange interface {
	revert(mdb *MemoryStateDB)
	getData(mdb *MemoryStateDB) []*types.KeyValue
	getLog(mdb *MemoryStateDB) []*types.ReceiptLog
}

// 版本结构，包含版本号以及当前版本包含的变更对象在变更序列中的开始序号
type Snapshot struct {
	id      int
	entries []DataChange
	statedb *MemoryStateDB
}

func (ver *Snapshot) GetId() int {
	return ver.id
}

// 回滚当前版本
func (ver *Snapshot) revert() bool {
	if ver.entries == nil {
		return true
	}
	for _, entry := range ver.entries {
		entry.revert(ver.statedb)
	}
	return true
}

// 添加变更数据
func (ver *Snapshot) append(entry DataChange) {
	ver.entries = append(ver.entries, entry)
}

// 获取当前版本变更数据
func (ver *Snapshot) getData() (kvSet []*types.KeyValue, logs []*types.ReceiptLog) {
	// 获取中间的数据变更
	dataMap := make(map[string]*types.KeyValue)

	for _, entry := range ver.entries {
		items := entry.getData(ver.statedb)
		logEntry := entry.getLog(ver.statedb)
		if logEntry != nil {
			logs = append(logs, entry.getLog(ver.statedb)...)
		}

		// 执行去重操作
		for _, kv := range items {
			dataMap[string(kv.Key)] = kv
		}
	}

	// 这里也可能会引起数据顺序不一致的问题，需要修改（目前看KV的顺序不会影响哈希计算，但代码最好保证顺序一致）
	names := make([]string, 0, len(dataMap))
	for name := range dataMap {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		kvSet = append(kvSet, dataMap[name])
	}

	return kvSet, logs
}

type (

	// 基础变更对象，用于封装默认操作
	baseChange struct {
	}

	// 创建合约对象变更事件
	createAccountChange struct {
		baseChange
		account string
	}

	// 自杀事件
	suicideChange struct {
		baseChange
		account string
		prev    bool // whether account had already suicided
	}

	// nonce变更事件
	nonceChange struct {
		baseChange
		account string
		prev    uint64
	}

	// 存储状态变更事件
	storageChange struct {
		baseChange
		account       string
		key, prevalue common.Hash
	}

	// 合约代码状态变更事件
	codeChange struct {
		baseChange
		account            string
		prevcode, prevhash []byte
	}

	// 返还金额变更事件
	refundChange struct {
		baseChange
		prev uint64
	}

	// 转账事件
	// 合约转账动作不执行回滚，失败后数据不会写入区块
	transferChange struct {
		baseChange
		amount int64
		data   []*types.KeyValue
		logs   []*types.ReceiptLog
	}

	// 合约生成日志事件
	addLogChange struct {
		baseChange
		txhash common.Hash
	}

	// 合约生成sha3事件
	addPreimageChange struct {
		baseChange
		hash common.Hash
	}
)

// 在baseChang中定义三个基本操作，子对象中只需要实现必要的操作
func (ch baseChange) revert(s *MemoryStateDB) {
}

func (ch baseChange) getData(s *MemoryStateDB) (kvset []*types.KeyValue) {
	return nil
}

func (ch baseChange) getLog(s *MemoryStateDB) (logs []*types.ReceiptLog) {
	return nil
}

// 创建账户对象的回滚，需要删除缓存中的账户和变更标记
func (ch createAccountChange) revert(s *MemoryStateDB) {
	delete(s.accounts, ch.account)
}

// 创建账户对象的数据集
func (ch createAccountChange) getData(s *MemoryStateDB) (kvset []*types.KeyValue) {
	acc := s.accounts[ch.account]
	if acc != nil {
		kvset = append(kvset, acc.GetDataKV()...)
		kvset = append(kvset, acc.GetStateKV()...)
		return kvset
	}
	return nil
}

func (ch suicideChange) revert(mdb *MemoryStateDB) {
	// 如果已经自杀过了，不处理
	if ch.prev {
		return
	}
	acc := mdb.accounts[ch.account]
	if acc != nil {
		acc.State.Suicided = ch.prev
	}
}

func (ch suicideChange) getData(mdb *MemoryStateDB) []*types.KeyValue {
	// 如果已经自杀过了，不处理
	if ch.prev {
		return nil
	}
	acc := mdb.accounts[ch.account]
	if acc != nil {
		return acc.GetStateKV()
	}
	return nil
}

func (ch nonceChange) revert(mdb *MemoryStateDB) {
	acc := mdb.accounts[ch.account]
	if acc != nil {
		acc.State.Nonce = ch.prev
	}
}

func (ch nonceChange) getData(mdb *MemoryStateDB) []*types.KeyValue {
	// nonce目前没有应用场景，而且每次调用都会变更，暂时先不写到状态数据库中
	//acc := mdb.accounts[ch.account]
	//if acc != nil {
	//	return acc.GetStateKV()
	//}
	return nil
}

func (ch codeChange) revert(mdb *MemoryStateDB) {
	acc := mdb.accounts[ch.account]
	if acc != nil {
		acc.Data.Code = ch.prevcode
		acc.Data.CodeHash = ch.prevhash
	}
}

func (ch codeChange) getData(mdb *MemoryStateDB) (kvset []*types.KeyValue) {
	acc := mdb.accounts[ch.account]
	if acc != nil {
		kvset = append(kvset, acc.GetDataKV()...)
		kvset = append(kvset, acc.GetStateKV()...)
		return kvset
	}
	return nil
}

func (ch storageChange) revert(mdb *MemoryStateDB) {
	acc := mdb.accounts[ch.account]
	if acc != nil {
		acc.SetState(ch.key, ch.prevalue)
	}
}

func (ch storageChange) getData(mdb *MemoryStateDB) []*types.KeyValue {
	acc := mdb.accounts[ch.account]
	if _, ok := mdb.stateDirty[ch.account]; ok && acc != nil {
		return acc.GetStateKV()
	}
	return nil
}

func (ch storageChange) getLog(mdb *MemoryStateDB) []*types.ReceiptLog {
	if types.IsDappFork(mdb.blockHeight, "evm", "ForkEVMState") {
		acc := mdb.accounts[ch.account]
		if acc != nil {
			currentVal := acc.GetState(ch.key)
			receipt := &evmtypes.EVMStateChangeItem{Key: getStateItemKey(ch.account, ch.key.Hex()), PreValue: ch.prevalue.Bytes(), CurrentValue: currentVal.Bytes()}
			return []*types.ReceiptLog{{Ty: evmtypes.TyLogEVMStateChangeItem, Log: types.Encode(receipt)}}
		}
	}
	return nil
}

func (ch refundChange) revert(mdb *MemoryStateDB) {
	mdb.refund = ch.prev
}

func (ch addLogChange) revert(mdb *MemoryStateDB) {
	logs := mdb.logs[ch.txhash]
	if len(logs) == 1 {
		delete(mdb.logs, ch.txhash)
	} else {
		mdb.logs[ch.txhash] = logs[:len(logs)-1]
	}
	mdb.logSize--
}

func (ch addPreimageChange) revert(mdb *MemoryStateDB) {
	delete(mdb.preimages, ch.hash)
}

func (ch transferChange) getData(mdb *MemoryStateDB) []*types.KeyValue {
	return ch.data
}
func (ch transferChange) getLog(mdb *MemoryStateDB) []*types.ReceiptLog {
	return ch.logs
}
