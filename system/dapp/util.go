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
	rollbackkvs  []*types.KeyValue
}

//NewKVCreator 创建创建者
//注意: 自动回滚可能会严重影响系统性能
func NewKVCreator(kv db.KV, autorollback bool) *KVCreator {
	return &KVCreator{kvdb: kv, autorollback: autorollback}
}

func (c *KVCreator) add(key, value []byte, set bool) *KVCreator {
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
		c.kvdb.Set(key, value)
	}
	return c
}

//Get 从KV中获取 value
func (c *KVCreator) Get(key []byte) ([]byte, error) {
	return c.kvdb.Get(key)
}

//Add add and set to kvdb
func (c *KVCreator) Add(key, value []byte) *KVCreator {
	return c.add(key, value, true)
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

//RollbackLog rollback log
func (c *KVCreator) RollbackLog() *types.ReceiptLog {
	data := types.Encode(&types.LocalDBSet{KV: c.rollbackkvs})
	return &types.ReceiptLog{Ty: types.TyLogRollback, Log: data}
}

//ParseRollback 解析rollback的数据
func (c *KVCreator) ParseRollback(log *types.ReceiptLog) ([]*types.KeyValue, error) {
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
	return append(logs, c.RollbackLog())
}
