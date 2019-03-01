// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dapp

import (
	"fmt"

	"github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
)

// HeightIndexStr height and index format string
func HeightIndexStr(height, index int64) string {
	v := height*types.MaxTxsPerBlock + index
	return fmt.Sprintf("%018d", v)
}

//KVCreator 创建KV的辅助工具
type KVCreator struct {
	kvs          []*types.KeyValue
	kvdb         db.KV
	autorollback bool
	prefix       []byte
	rollbackkey  []byte
	rollbackkvs  []*types.KeyValue
}

//NewKVCreator 创建创建者
//注意: 自动回滚可能会严重影响系统性能
func NewKVCreator(kv db.KV, prefix []byte, rollbackkey []byte) *KVCreator {
	return &KVCreator{
		kvdb:         kv,
		prefix:       prefix,
		rollbackkey:  rollbackkey,
		autorollback: rollbackkey != nil,
	}
}

func (c *KVCreator) addPrefix(key []byte) []byte {
	newkey := append([]byte{}, c.prefix...)
	return append(newkey, key...)
}

func (c *KVCreator) add(key, value []byte, set bool) *KVCreator {
	if c.prefix != nil {
		key = c.addPrefix(key)
	}
	return c.addnoprefix(key, value, set)
}

func (c *KVCreator) addnoprefix(key, value []byte, set bool) *KVCreator {
	if c.autorollback {
		prev, err := c.kvdb.Get(key)
		//数据库发生错误，直接panic (再执行器中会recover)
		if err != nil && err != types.ErrNotFound {
			panic(err)
		}
		if value == nil { //del
			//不需要做任何处理, 也不用加入 kvs
			if err == types.ErrNotFound {
				return c
			}
			rb := &types.KeyValue{Key: key, Value: prev}
			c.rollbackkvs = append(c.rollbackkvs, rb)
		} else {
			if err == types.ErrNotFound { //add
				rb := &types.KeyValue{Key: key}
				c.rollbackkvs = append(c.rollbackkvs, rb)
			} else { //update
				rb := &types.KeyValue{Key: key, Value: prev}
				c.rollbackkvs = append(c.rollbackkvs, rb)
			}
		}
	}
	c.kvs = append(c.kvs, &types.KeyValue{Key: key, Value: value})
	if set {
		err := c.kvdb.Set(key, value)
		if err != nil {
			panic(err)
		}
	}
	return c
}

//Get 从KV中获取 value
func (c *KVCreator) Get(key []byte) ([]byte, error) {
	if c.prefix != nil {
		newkey := c.addPrefix(key)
		return c.kvdb.Get(newkey)
	}
	return c.kvdb.Get(key)
}

//GetNoPrefix 从KV中获取 value, 不自动添加前缀
func (c *KVCreator) GetNoPrefix(key []byte) ([]byte, error) {
	return c.kvdb.Get(key)
}

//Add add and set to kvdb
func (c *KVCreator) Add(key, value []byte) *KVCreator {
	return c.add(key, value, true)
}

//AddNoPrefix 不自动添加prefix
func (c *KVCreator) AddNoPrefix(key, value []byte) *KVCreator {
	return c.addnoprefix(key, value, true)
}

//AddListNoPrefix only add KVList
func (c *KVCreator) AddListNoPrefix(list []*types.KeyValue) *KVCreator {
	for _, kv := range list {
		c.addnoprefix(kv.Key, kv.Value, true)
	}
	return c
}

//AddList only add KVList
func (c *KVCreator) AddList(list []*types.KeyValue) *KVCreator {
	for _, kv := range list {
		c.add(kv.Key, kv.Value, true)
	}
	return c
}

//AddKVOnly only add KV(can't auto rollback)
func (c *KVCreator) AddKVOnly(key, value []byte) *KVCreator {
	if c.autorollback {
		panic("autorollback open, AddKVOnly not allow")
	}
	return c.add(key, value, false)
}

//AddKVListOnly only add KVList (can't auto rollback)
func (c *KVCreator) AddKVListOnly(list []*types.KeyValue) *KVCreator {
	if c.autorollback {
		panic("autorollback open, AddKVListOnly not allow")
	}
	for _, kv := range list {
		c.add(kv.Key, kv.Value, false)
	}
	return c
}

//KVList 读取所有的kv列表
func (c *KVCreator) KVList() []*types.KeyValue {
	return c.kvs
}

//AddRollbackKV 添加回滚数据到 KV
func (c *KVCreator) AddRollbackKV() {
	v := types.Encode(c.rollbackLog())
	c.kvs = append(c.kvs, &types.KeyValue{Key: c.rollbackkey, Value: v})
}

//DelRollbackKV 删除rollback kv
func (c *KVCreator) DelRollbackKV() {
	c.kvs = append(c.kvs, &types.KeyValue{Key: c.rollbackkey})
}

//GetRollbackKVList 获取 rollback 到 Key and Vaue
func (c *KVCreator) GetRollbackKVList() ([]*types.KeyValue, error) {
	data, err := c.kvdb.Get(c.rollbackkey)
	if err != nil {
		return nil, err
	}
	var rollbacklog types.ReceiptLog
	err = types.Decode(data, &rollbacklog)
	if err != nil {
		return nil, err
	}
	kvs, err := c.parseRollback(&rollbacklog)
	if err != nil {
		return nil, err
	}
	//reverse kvs
	for left, right := 0, len(kvs)-1; left < right; left, right = left+1, right-1 {
		kvs[left], kvs[right] = kvs[right], kvs[left]
	}
	return kvs, nil
}

//rollbackLog rollback log
func (c *KVCreator) rollbackLog() *types.ReceiptLog {
	data := types.Encode(&types.LocalDBSet{KV: c.rollbackkvs})
	return &types.ReceiptLog{Ty: types.TyLogRollback, Log: data}
}

//ParseRollback 解析rollback的数据
func (c *KVCreator) parseRollback(log *types.ReceiptLog) ([]*types.KeyValue, error) {
	var data types.LocalDBSet
	if log.Ty != types.TyLogRollback {
		return nil, types.ErrInvalidParam
	}
	err := types.Decode(log.Log, &data)
	if err != nil {
		return nil, err
	}
	return data.KV, nil
}

//AddToLogs add not empty log to logs
func (c *KVCreator) AddToLogs(logs []*types.ReceiptLog) []*types.ReceiptLog {
	if len(c.rollbackkvs) == 0 {
		return logs
	}
	return append(logs, c.rollbackLog())
}
