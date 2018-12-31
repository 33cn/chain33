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
	kvs  []*types.KeyValue
	kvdb db.KV
}

//NewKVCreator 创建创建者
func NewKVCreator(kv db.KV) *KVCreator {
	return &KVCreator{kvdb: kv}
}

func (c *KVCreator) add(key, value []byte, set bool) *KVCreator {
	c.kvs = append(c.kvs, &types.KeyValue{Key: key, Value: value})
	if set {
		c.kvdb.Set(key, value)
	}
	return c
}

//Add add and set to kvdb
func (c *KVCreator) Add(key, value []byte) *KVCreator {
	return c.add(key, value, true)
}

//AddKV only add KV
func (c *KVCreator) AddKV(key, value []byte) *KVCreator {
	return c.add(key, value, false)
}

//KVList 读取所有的kv列表
func (c *KVCreator) KVList() []*types.KeyValue {
	return c.kvs
}
