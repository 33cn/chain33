// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"encoding/hex"

	"github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

// StateDB state db for store mavl
type StateDB struct {
	cache     map[string][]byte
	txcache   map[string][]byte
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
		cache:     make(map[string][]byte),
		txcache:   make(map[string][]byte),
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
	s.keys = nil
	if types.IsFork(s.height, "ForkExecRollback") {
		s.txcache = nil
	}
}

// Rollback reset tx
func (s *StateDB) Rollback() {
	s.resetTx()
}

// Commit canche tx
func (s *StateDB) Commit() {
	for k, v := range s.txcache {
		s.cache[k] = v
	}
	s.intx = false
	s.keys = nil
	if types.IsFork(s.height, "ForkExecRollback") {
		s.resetTx()
	}
}

func (s *StateDB) resetTx() {
	s.intx = false
	s.txcache = nil
	s.keys = nil
}

// Get get value from state db
func (s *StateDB) Get(key []byte) ([]byte, error) {
	v, err := s.get(key)
	debugAccount("==get==", key, v)
	return v, err
}

func (s *StateDB) get(key []byte) ([]byte, error) {
	skey := string(key)
	if s.intx && s.txcache != nil {
		if value, ok := s.txcache[skey]; ok {
			return value, nil
		}
	}
	if value, ok := s.cache[skey]; ok {
		return value, nil
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
	s.client.Send(msg, true)
	resp, err := s.client.Wait(msg)
	if err != nil {
		panic(err) //no happen for ever
	}
	if nil == resp.GetData().(*types.StoreReplyValue).Values {
		return nil, types.ErrNotFound
	}
	value := resp.GetData().(*types.StoreReplyValue).Values[0]
	if value == nil {
		//panic(string(key))
		return nil, types.ErrNotFound
	}
	//get 的值可以写入cache，因为没有对系统的值做修改
	s.cache[skey] = value
	return value, nil
}

func debugAccount(prefix string, key []byte, value []byte) {
	//println(prefix, string(key), value)
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
}

// StartTx reset state db keys
func (s *StateDB) StartTx() {
	s.keys = nil
}

// GetSetKeys  get state db set keys
func (s *StateDB) GetSetKeys() (keys []string) {
	return s.keys
}

// Set set key value to state db
func (s *StateDB) Set(key []byte, value []byte) error {
	debugAccount("==set==", key, value)
	skey := string(key)
	if s.intx {
		if s.txcache == nil {
			s.txcache = make(map[string][]byte)
		}
		s.keys = append(s.keys, skey)
		setmap(s.txcache, skey, value)
	} else {
		setmap(s.cache, skey, value)
	}
	return nil
}

func setmap(data map[string][]byte, key string, value []byte) {
	if value == nil {
		delete(data, key)
		return
	}
	data[key] = value
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

// LocalDB local db for store key value in local
type LocalDB struct {
	db.TransactionDB
	cache  map[string][]byte
	client queue.Client
}

// NewLocalDB new local db
func NewLocalDB(client queue.Client) db.KVDB {
	return &LocalDB{cache: make(map[string][]byte), client: client}
}

// Get get value from local db
func (l *LocalDB) Get(key []byte) ([]byte, error) {
	value, err := l.get(key)
	debugAccount("==lget==", key, value)
	return value, err
}

func (l *LocalDB) get(key []byte) ([]byte, error) {
	if value, ok := l.cache[string(key)]; ok {
		return value, nil
	}
	if l.client == nil {
		return nil, types.ErrNotFound
	}
	query := &types.LocalDBGet{Keys: [][]byte{key}}
	msg := l.client.NewMessage("blockchain", types.EventLocalGet, query)
	l.client.Send(msg, true)
	resp, err := l.client.Wait(msg)

	if err != nil {
		panic(err) //no happen for ever
	}
	if nil == resp.GetData().(*types.LocalReplyValue).Values {
		return nil, types.ErrNotFound
	}
	value := resp.GetData().(*types.LocalReplyValue).Values[0]
	if value == nil {
		//panic(string(key))
		return nil, types.ErrNotFound
	}
	l.cache[string(key)] = value
	return value, nil
}

// Set set key value to local db
func (l *LocalDB) Set(key []byte, value []byte) error {
	debugAccount("==lset==", key, value)
	setmap(l.cache, string(key), value)
	return nil
}

// BatchGet batch get values from local db
func (l *LocalDB) BatchGet(keys [][]byte) (values [][]byte, err error) {
	for _, key := range keys {
		v, err := l.Get(key)
		if err != nil && err != types.ErrNotFound {
			return nil, err
		}
		values = append(values, v)
	}
	return values, nil
}

// List 从数据库中查询数据列表，set 中的cache 更新不会影响这个list
func (l *LocalDB) List(prefix, key []byte, count, direction int32) ([][]byte, error) {
	if l.client == nil {
		return nil, types.ErrNotFound
	}
	query := &types.LocalDBList{Prefix: prefix, Key: key, Count: count, Direction: direction}
	msg := l.client.NewMessage("blockchain", types.EventLocalList, query)
	l.client.Send(msg, true)
	resp, err := l.client.Wait(msg)
	if err != nil {
		panic(err) //no happen for ever
	}
	values := resp.GetData().(*types.LocalReplyValue).Values
	if values == nil {
		//panic(string(key))
		return nil, types.ErrNotFound
	}
	return values, nil
}

// PrefixCount 从数据库中查询指定前缀的key的数量
func (l *LocalDB) PrefixCount(prefix []byte) (count int64) {
	if l.client == nil {
		return 0
	}
	query := &types.ReqKey{Key: prefix}
	msg := l.client.NewMessage("blockchain", types.EventLocalPrefixCount, query)
	l.client.Send(msg, true)
	resp, err := l.client.Wait(msg)
	if err != nil {
		panic(err) //no happen for ever
	}
	count = resp.GetData().(*types.Int64).Data
	return
}
