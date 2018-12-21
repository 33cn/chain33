// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package table

import (
	"math"

	"github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
)

//Count 计数器
type Count struct {
	prefix  string
	name    string
	kvdb    db.KV
	num     int64
	keydata []byte
}

//NewCount 创建一个计数器
func NewCount(prefix string, name string, kvdb db.KV) *Count {
	keydata := []byte(prefix + sep + name)
	return &Count{
		prefix:  prefix,
		name:    name,
		kvdb:    kvdb,
		keydata: keydata,
		num:     math.MinInt64,
	}
}

func (c *Count) getKey() []byte {
	return c.keydata
}

//Save 保存kv
func (c *Count) Save() (kvs []*types.KeyValue, err error) {
	if c.num == math.MinInt64 {
		return nil, nil
	}
	var i types.Int64
	i.Data = c.num
	item := &types.KeyValue{Key: c.getKey(), Value: types.Encode(&i)}
	kvs = append(kvs, item)
	return
}

//Get count
func (c *Count) Get() (int64, error) {
	if c.num == math.MinInt64 {
		data, err := c.kvdb.Get(c.getKey())
		if err == types.ErrNotFound {
			c.num = 0
		} else if err != nil {
			return 0, err
		}
		var num types.Int64
		err = types.Decode(data, &num)
		if err != nil {
			return 0, err
		}
		c.num = num.Data
	}
	return c.num, nil
}

//Inc 增加1
func (c *Count) Inc() (num int64, err error) {
	c.num, err = c.Get()
	if err != nil {
		return 0, err
	}
	c.num++
	return c.num, nil
}

//Dec 减少1
func (c *Count) Dec() (num int64, err error) {
	c.num, err = c.Get()
	if err != nil {
		return 0, err
	}
	c.num--
	return c.num, nil
}

//Set 这个操作要谨慎使用
func (c *Count) Set(i int64) {
	c.num = i
}
