// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"encoding/hex"
	"errors"

	"github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

var (
	// ErrStateDBRevertWithoutTx is returned by RevertSet when no transaction is
	// active. Undoing a write is only meaningful between Begin and Commit/Rollback:
	// outside a transaction Set writes directly to the block cache and the
	// previous value is no longer available.
	ErrStateDBRevertWithoutTx = errors.New("statedb: RevertSet called outside a transaction")
	// ErrStateDBEmptyKey is returned by RevertSet for a nil or empty state key.
	ErrStateDBEmptyKey = errors.New("statedb: RevertSet called with an empty key")
)

// StateDB state db for store mavl
type StateDB struct {
	cache     *cacheDB
	txcache   *cacheDB
	keys      []string
	intx      bool
	client    queue.Client
	stateHash []byte
	version   int64
	height    int64
	local     *db.SimpleMVCC
	opt       *StateDBOption
}

// StateDBOption state db option enable mvcc
type StateDBOption struct {
	EnableMVCC bool
	Height     int64
}

// NewStateDB new state db
func NewStateDB(client queue.Client, stateHash []byte, localdb db.KVDB, opt *StateDBOption) db.KV {
	if opt == nil {
		opt = &StateDBOption{}
	}
	db := &StateDB{
		//预分配一个单位
		cache:     newcacheDB(1024),
		txcache:   newcacheDB(32),
		keys:      make([]string, 0, 32),
		intx:      false,
		client:    client,
		stateHash: stateHash,
		height:    opt.Height,
		version:   -1,
		local:     db.NewSimpleMVCC(localdb),
		opt:       opt,
	}
	return db
}

func (s *StateDB) enableMVCC(hash []byte) {
	opt := s.opt
	if opt.EnableMVCC {
		if hash == nil {
			hash = s.stateHash
		}
		v, err := s.local.GetVersion(hash)
		if err == nil && v >= 0 {
			s.version = v
		} else if s.height > 0 {
			println("init state db", "height", s.height, "err", err.Error(), "v", v, "stateHash", hex.EncodeToString(s.stateHash))
			panic("mvcc get version error,config set enableMVCC=true, it must be synchronized from 0 height")
		}
	}
}

// Begin 开启内存事务处理
func (s *StateDB) Begin() {
	s.intx = true
	s.keys = s.keys[:0]
	types.AssertConfig(s.client)
	cfg := s.client.GetConfig()
	if cfg.IsFork(s.height, "ForkExecRollback") {
		s.txcache.Reset()
	}
}

// Rollback reset tx
func (s *StateDB) Rollback() {
	s.resetTx()
}

// ResetCache evm 使用
func (s *StateDB) ResetCache() {
	s.cache.Reset()
	s.resetTx()
}

// Commit canche tx
func (s *StateDB) Commit() error {
	s.cache.Merge(s.txcache)
	s.intx = false
	s.keys = s.keys[:0]
	types.AssertConfig(s.client)
	cfg := s.client.GetConfig()
	if cfg.IsFork(s.height, "ForkExecRollback") {
		s.resetTx()
	}
	return nil
}

func (s *StateDB) resetTx() {
	s.intx = false
	s.txcache.Reset()
	s.keys = s.keys[:0]
}

// Get get value from state db
func (s *StateDB) Get(key []byte) ([]byte, error) {
	v, err := s.get(key)
	//debugAccount("==get==", key, v)
	return v, err
}

func (s *StateDB) get(key []byte) ([]byte, error) {

	if s.intx {
		if value, exist, err := s.txcache.Get(key); exist {
			return value, err
		}
	}
	if value, exist, err := s.cache.Get(key); exist {
		return value, err
	}
	//mvcc 是有效的情况下，直接从mvcc中获取
	if s.version >= 0 {
		data, err := s.local.GetV(key, s.version)
		//TODO 这里需要一个标志，数据是否是从0开始同步的
		return data, err
	}
	if s.client == nil {
		return nil, types.ErrNotFound
	}
	query := &types.StoreGet{StateHash: s.stateHash, Keys: [][]byte{key}}
	msg := s.client.NewMessage("store", types.EventStoreGet, query)
	err := s.client.Send(msg, true)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Wait(msg)
	if err != nil {
		panic(err) //no happen for ever
	}
	defer s.client.FreeMessage(msg, resp)
	if nil == resp.GetData().(*types.StoreReplyValue).Values {
		return nil, types.ErrNotFound
	}
	value := resp.GetData().(*types.StoreReplyValue).Values[0]
	if value == nil {
		//panic(string(key))
		return nil, types.ErrNotFound
	}
	//get 的值可以写入cache，因为没有对系统的值做修改
	s.cache.Set(key, value)
	return value, nil
}

//func debugAccount(prefix string, key []byte, value []byte) {
//println(prefix, string(key), string(value))
/*
	if !types.Debug {
		return
	}
	var msg types.Account
	err := types.Decode(value, &msg)
	if err == nil {
		elog.Info(prefix, "key", string(key), "value", msg)
	}
*/
//}

// StartTx reset state db keys
func (s *StateDB) StartTx() {
	s.keys = s.keys[:0]
}

// GetSetKeys  get state db set keys
func (s *StateDB) GetSetKeys() (keys []string) {
	return s.keys
}

// Set set key value to state db
func (s *StateDB) Set(key []byte, value []byte) error {
	//debugAccount("==set==", key, value)
	if s.intx {
		s.keys = append(s.keys, string(key))
		s.txcache.Set(key, value)
	} else {
		s.cache.Set(key, value)
	}
	return nil
}

// RevertSet undoes one write that the current transaction performed on key and
// restores the value the key had before that write.
//
// The invariant it maintains is the one the executor relies on: every key in
// GetSetKeys must also appear in the transaction receipt (see checkKV in
// execenv.go). A reverted write must therefore stop being reported unless some
// other write to the same key in this transaction survives:
//
//   - exactly one recorded occurrence of key is dropped from the GetSetKeys
//     list, so duplicates left by repeated Set calls are handled one write at
//     a time;
//   - the transaction cache is set to originalValue, so reads later in the same
//     transaction observe the restored value without touching the block cache.
//
// Writes must be reverted in LIFO order: originalValue is the value observed
// before the reverted write, so undoing an older write while a newer one
// survives would not describe the value left by the newest surviving write.
//
// A nil originalValue means the key did not exist before the write. The tx
// cache then holds the nil sentinel: cacheDB.Get reports such an entry as
// types.ErrNotFound, so the key reads as absent, and because its last recorded
// occurrence was removed it no longer appears in GetSetKeys or in the receipt.
//
// originalValue is supplied by the caller (the pre-write state it captured)
// rather than read from storage, because earlier writes of this transaction
// live only in the transaction cache. A key with no recorded write in this
// transaction is a no-op, making revert paths idempotent. A nil/empty key is
// rejected with ErrStateDBEmptyKey and calling outside a transaction with
// ErrStateDBRevertWithoutTx; no input causes a panic.
func (s *StateDB) RevertSet(key []byte, originalValue []byte) error {
	if len(key) == 0 {
		return ErrStateDBEmptyKey
	}
	if !s.intx {
		return ErrStateDBRevertWithoutTx
	}
	if !s.removeLastSetKey(string(key)) {
		return nil
	}
	s.txcache.Set(key, originalValue)
	return nil
}

// removeLastSetKey drops the most recent recorded write of key from the
// transaction key list and reports whether one was found.
func (s *StateDB) removeLastSetKey(key string) bool {
	for i := len(s.keys) - 1; i >= 0; i-- {
		if s.keys[i] == key {
			s.keys = append(s.keys[:i], s.keys[i+1:]...)
			return true
		}
	}
	return false
}

// BatchGet batch get keys from state db
func (s *StateDB) BatchGet(keys [][]byte) (values [][]byte, err error) {
	for _, key := range keys {
		v, err := s.Get(key)
		if err != nil && err != types.ErrNotFound {
			return nil, err
		}
		values = append(values, v)
	}
	return values, nil
}
