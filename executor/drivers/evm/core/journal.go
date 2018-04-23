// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/common"
	"gitlab.33.cn/chain33/chain33/types"
)

type journalEntry interface {
	undo(mdb *MemoryStateDB)
	getData(mdb *MemoryStateDB) []*types.KeyValue
}

type journal []journalEntry

type (
	// 创建合约对象变更事件
	createAccountChange struct {
		account *common.Address
	}

	// 自杀事件
	suicideChange struct {
		account     *common.Address
		prev        bool // whether account had already suicided
	}

	// nonce变更事件
	nonceChange struct {
		account *common.Address
		prev    uint64
	}

	// 存储状态变更事件
	storageChange struct {
		account       *common.Address
		key, prevalue common.Hash
	}

	// 合约代码状态变更事件
	codeChange struct {
		account            *common.Address
		prevcode, prevhash []byte
	}

	// 返还金额变更事件
	refundChange struct {
		prev uint64
	}


	addLogChange struct {
		txhash common.Hash
	}

	addPreimageChange struct {
		hash common.Hash
	}
)

// 创建账户对象的回滚，需要删除缓存中的账户和变更标记
func (ch createAccountChange) undo(s *MemoryStateDB) {
	delete(s.accounts, *ch.account)
	delete(s.accountsDirty, *ch.account)
	delete(s.contractsDirty, *ch.account)
}

// 创建账户对象的数据集
func (ch createAccountChange) getData(s *MemoryStateDB) []*types.KeyValue {
	acc := s.accounts[*ch.account]
	if acc != nil {
		return acc.GetContractData()
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
		acc.suicided = ch.prev
	}
}

func (ch suicideChange) getData(mdb *MemoryStateDB) []*types.KeyValue {
	// 如果已经自杀过了，不处理
	if ch.prev {
		return nil
	}
	acc := mdb.accounts[*ch.account]
	if acc != nil {
		return acc.getSuicideData()
	}
	return nil
}

func (ch nonceChange) undo(mdb *MemoryStateDB) {
	acc := mdb.accounts[*ch.account]
	if acc != nil {
		acc.nonce = ch.prev
	}
}

func (ch nonceChange) getData(mdb *MemoryStateDB) []*types.KeyValue {
	acc := mdb.accounts[*ch.account]
	if acc != nil {
		return acc.GetChangeData()
	}
	return nil
}


func (ch codeChange) undo(mdb *MemoryStateDB) {
	acc := mdb.accounts[*ch.account]
	if acc != nil {
		acc.code = ch.prevcode
		acc.codeHash = common.BytesToHash(ch.prevhash)
	}
}

func (ch codeChange) getData(mdb *MemoryStateDB) []*types.KeyValue {
	acc := mdb.accounts[*ch.account]
	if acc != nil {
		return acc.GetContractData()
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
		return acc.GetChangeData()
	}
	return nil
}

func (ch refundChange) undo(mdb *MemoryStateDB) {
	mdb.refund = ch.prev
}
func (ch refundChange) getData(mdb *MemoryStateDB) []*types.KeyValue {
	return nil
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
func (ch addLogChange) getData(mdb *MemoryStateDB) []*types.KeyValue {
	return nil
}


func (ch addPreimageChange) undo(mdb *MemoryStateDB) {
	delete(mdb.preimages, ch.hash)
}
func (ch addPreimageChange) getData(mdb *MemoryStateDB) []*types.KeyValue {
	return nil
}