package state

import (
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/common"
	"gitlab.33.cn/chain33/chain33/types"
)

// 数据状态变更接口
// 所有的数据状态变更事件实现此接口，并且封装各自的变更数据以及回滚动作
// 在调用合约时（具体的Tx执行时），会根据操作生成对应的变更对象并缓存下来
// 如果合约执行出错，会按生成顺序的倒序，依次调用变更对象的回滚接口进行数据回滚，并同步删除变更对象缓存
// 如果合约执行成功，会按生成顺序的郑旭，依次调用变更对象的数据和日志变更记录，回传给区块链
type journalEntry interface {
	undo(mdb *MemoryStateDB)
	getData(mdb *MemoryStateDB) []*types.KeyValue
	getLog(mdb *MemoryStateDB) []*types.ReceiptLog
}

// 定义变更对象序列
type journal []journalEntry

type (

	// 基础变更对象，用于封装默认操作
	baseChange struct {
	}

	// 创建合约对象变更事件
	createAccountChange struct {
		baseChange
		account *common.Address
	}

	// 自杀事件
	suicideChange struct {
		baseChange
		account     *common.Address
		prev        bool // whether account had already suicided
	}

	// nonce变更事件
	nonceChange struct {
		baseChange
		account *common.Address
		prev    uint64
	}

	// 存储状态变更事件
	storageChange struct {
		baseChange
		account       *common.Address
		key, prevalue common.Hash
	}

	// 合约代码状态变更事件
	codeChange struct {
		baseChange
		account            *common.Address
		prevcode, prevhash []byte
	}

	// 返还金额变更事件
	refundChange struct {
		baseChange
		prev uint64
	}

	// 金额变更事件
	balanceChange struct {
		baseChange
		amount int64
		addr string
		data []*types.KeyValue
		logs []*types.ReceiptLog
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
func (ch baseChange) undo(s *MemoryStateDB) {
}

func (ch baseChange) getData(s *MemoryStateDB) (kvset []*types.KeyValue) {
	return nil
}

func (ch baseChange) getLog(s *MemoryStateDB) (logs []*types.ReceiptLog) {
	return nil
}

// 创建账户对象的回滚，需要删除缓存中的账户和变更标记
func (ch createAccountChange) undo(s *MemoryStateDB) {
	delete(s.accounts, *ch.account)
	delete(s.accountsDirty, *ch.account)
	delete(s.contractsDirty, *ch.account)
}

// 创建账户对象的数据集
func (ch createAccountChange) getData(s *MemoryStateDB) (kvset []*types.KeyValue) {
	acc := s.accounts[*ch.account]
	if acc != nil {
		kvset = append(kvset, acc.GetDataKV()...)
		kvset = append(kvset, acc.GetStateKV()...)
		return kvset
	}
	return nil
}


func (ch suicideChange) undo(mdb *MemoryStateDB) {
	// 如果已经自杀过了，不处理
	if ch.prev {
		return
	}
	acc := mdb.accounts[*ch.account]
	if acc != nil {
		acc.State.Suicided = ch.prev
	}
}

func (ch suicideChange) getData(mdb *MemoryStateDB) []*types.KeyValue {
	// 如果已经自杀过了，不处理
	if ch.prev {
		return nil
	}
	acc := mdb.accounts[*ch.account]
	if acc != nil {
		return acc.GetStateKV()
	}
	return nil
}

func (ch nonceChange) undo(mdb *MemoryStateDB) {
	acc := mdb.accounts[*ch.account]
	if acc != nil {
		acc.State.Nonce = ch.prev
	}
}

func (ch nonceChange) getData(mdb *MemoryStateDB) []*types.KeyValue {
	acc := mdb.accounts[*ch.account]
	if acc != nil {
		return acc.GetStateKV()
	}
	return nil
}


func (ch codeChange) undo(mdb *MemoryStateDB) {
	acc := mdb.accounts[*ch.account]
	if acc != nil {
		acc.Data.Code = ch.prevcode
		acc.Data.CodeHash = ch.prevhash
	}
}

func (ch codeChange) getData(mdb *MemoryStateDB) (kvset []*types.KeyValue) {
	acc := mdb.accounts[*ch.account]
	if acc != nil {
		kvset = append(kvset, acc.GetDataKV()...)
		kvset = append(kvset, acc.GetStateKV()...)
		return kvset
	}
	return nil
}

func (ch storageChange) undo(mdb *MemoryStateDB) {
	acc := mdb.accounts[*ch.account]
	if acc != nil {
		acc.SetState(ch.key, ch.prevalue)
	}
}

func (ch storageChange) getData(mdb *MemoryStateDB) []*types.KeyValue {
	acc := mdb.accounts[*ch.account]
	if acc != nil {
		return acc.GetStateKV()
	}
	return nil
}

func (ch refundChange) undo(mdb *MemoryStateDB) {
	mdb.refund = ch.prev
}

func (ch addLogChange) undo(mdb *MemoryStateDB) {
	logs := mdb.logs[ch.txhash]
	if len(logs) == 1 {
		delete(mdb.logs, ch.txhash)
	} else {
		mdb.logs[ch.txhash] = logs[:len(logs)-1]
	}
	mdb.logSize--
}

func (ch addPreimageChange) undo(mdb *MemoryStateDB) {
	delete(mdb.preimages, ch.hash)
}

// 恢复成变化前的金额
// 注意，这里需要重新加载coins账户，并增加金额，而不是直接设置金额
// 避免中间其它交易修改coins账户金额引起的数据覆盖
func (ch balanceChange) undo(mdb *MemoryStateDB) {
	acc := mdb.CoinsAccount.LoadAccount(ch.addr)
	acc.Balance -= ch.amount
	mdb.CoinsAccount.SaveAccount(acc)
}

func (ch balanceChange) getData(mdb *MemoryStateDB) []*types.KeyValue {
	return ch.data
}
func (ch balanceChange) getLog(mdb *MemoryStateDB) []*types.ReceiptLog {
	return ch.logs
}