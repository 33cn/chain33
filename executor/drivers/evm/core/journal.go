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
	"math/big"

	"gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/common"
	"gitlab.33.cn/chain33/chain33/types"
)

type journalEntry interface {
	undo(kv *MemoryStateDB)
	getData(kv *MemoryStateDB) []*types.KeyValue
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
		prevbalance *big.Int
	}

	// 外部账户余额变更事件
	balanceChange struct {
		account *common.Address
		prev    int64
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


func (ch suicideChange) undo(kv *MemoryStateDB) {
	// 如果已经自杀过了，不处理
	if ch.prev {
		return
	}
	acc := kv.accounts[*ch.account]
	if acc != nil {
		acc.suicided = ch.prev

		if ch.prevbalance.Int64() > 0 {
			acc.Balance = ch.prevbalance.Int64()
		}
	}
}

func (ch suicideChange) getData(kv *MemoryStateDB) []*types.KeyValue {
	// 如果已经自杀过了，不处理
	if ch.prev {
		return nil
	}
	acc := kv.accounts[*ch.account]
	if acc != nil {

		if ch.prevbalance.Int64() > 0 {
			return acc.getCoinsData()
		}
		return nil
	}
	return nil
}

var ripemd = common.HexToAddress("0000000000000000000000000000000000000003")

func (ch touchChange) undo(s db.KV) {
	// TODO empty
}

func (ch balanceChange) undo(kv *MemoryStateDB) {
	// TODO empty
}

func (ch balanceChange) getData(kv *MemoryStateDB) []*types.KeyValue {
	acc := kv.accounts[*ch.account]
	if acc != nil {
		return acc.getCoinsData()
	}
	return nil
}

func (ch nonceChange) undo(kv *MemoryStateDB) {
	acc := kv.accounts[*ch.account]
	if acc != nil {
		acc.nonce = ch.prev
	}
}

func (ch nonceChange) getData(kv *MemoryStateDB) []*types.KeyValue {
	acc := kv.accounts[*ch.account]
	if acc != nil {
		return acc.GetContractData()
	}
	return nil
}


func (ch codeChange) undo(kv *MemoryStateDB) {
	acc := kv.accounts[*ch.account]
	if acc != nil {
		acc.code = ch.prevcode
		acc.codeHash = common.BytesToHash(ch.prevhash)
	}
}

func (ch codeChange) getData(kv *MemoryStateDB) []*types.KeyValue {
	acc := kv.accounts[*ch.account]
	if acc != nil {
		return acc.GetContractData()
	}
	return nil
}

func (ch storageChange) undo(kv *MemoryStateDB) {
	acc := kv.accounts[*ch.account]
	if acc != nil {
		acc.SetState(ch.key, ch.prevalue)
	}
}

func (ch storageChange) getData(kv *MemoryStateDB) []*types.KeyValue {
	acc := kv.accounts[*ch.account]
	if acc != nil {
		return acc.GetContractData()
	}
	return nil
}

func (ch refundChange) undo(kv *MemoryStateDB) {
	kv.refund = ch.prev
}
func (ch refundChange) getData(kv *MemoryStateDB) []*types.KeyValue {
	return nil
}

func (ch addLogChange) undo(s db.KV) {
	logs := s.logs[ch.txhash]
	if len(logs) == 1 {
		delete(s.logs, ch.txhash)
	} else {
		s.logs[ch.txhash] = logs[:len(logs)-1]
	}
	s.logSize--
}

func (ch addPreimageChange) undo(s db.KV) {
	delete(s.preimages, ch.hash)
}
