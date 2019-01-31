// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package db

import (
	"encoding/hex"
	"math/rand"
	"testing"

	"fmt"

	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	strChars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz" // 62 characters
)

func RandInt() int {
	return rand.Int()
}

func RandStr(length int) string {
	chars := []byte{}
MAIN_LOOP:
	for {
		val := rand.Int63()
		for i := 0; i < 10; i++ {
			v := int(val & 0x3f) // rightmost 6 bits
			if v >= 62 {         // only 62 characters in strChars
				val >>= 6
				continue
			} else {
				chars = append(chars, strChars[v])
				if len(chars) == length {
					break MAIN_LOOP
				}
				val >>= 6
			}
		}
	}

	return string(chars)
}

// 迭代测试
func testDBIterator(t *testing.T, db DB) {
	t.Log("test Set")
	db.Set([]byte("aaaaaa/1"), []byte("aaaaaa/1"))
	db.Set([]byte("my_key/1"), []byte("my_key/1"))
	db.Set([]byte("my_key/2"), []byte("my_key/2"))
	db.Set([]byte("my_key/3"), []byte("my_key/3"))
	db.Set([]byte("my_key/4"), []byte("my_key/4"))
	db.Set([]byte("my"), []byte("my"))
	db.Set([]byte("my_"), []byte("my_"))
	db.Set([]byte("zzzzzz/1"), []byte("zzzzzz/1"))
	b, err := hex.DecodeString("ff")
	require.NoError(t, err)
	db.Set(b, []byte("0xff"))

	t.Log("test Get")
	v, _ := db.Get([]byte("aaaaaa/1"))
	require.Equal(t, string(v), "aaaaaa/1")
	//test list:
	it0 := NewListHelper(db)
	list0 := it0.List(nil, nil, 100, 1)
	for _, v = range list0 {
		t.Log("list0", string(v))
	}
	t.Log("test PrefixScan")
	it := NewListHelper(db)
	list := it.PrefixScan(nil)
	for _, v = range list {
		t.Log("list:", string(v))
	}
	assert.Equal(t, list0, list)
	require.Equal(t, list, [][]byte{[]byte("aaaaaa/1"), []byte("my"), []byte("my_"), []byte("my_key/1"), []byte("my_key/2"), []byte("my_key/3"), []byte("my_key/4"), []byte("zzzzzz/1"), []byte("0xff")})
	t.Log("test IteratorScanFromFirst")
	list = it.IteratorScanFromFirst([]byte("my"), 2)
	/*for _, v = range list {
		t.Log(string(v))
	}*/
	require.Equal(t, list, [][]byte{[]byte("my"), []byte("my_")})

	t.Log("test IteratorScanFromLast")
	list = it.IteratorScanFromLast([]byte("my"), 100)
	/*for _, v = range list {
		t.Log(string(v))
	}*/
	require.Equal(t, list, [][]byte{[]byte("my_key/4"), []byte("my_key/3"), []byte("my_key/2"), []byte("my_key/1"), []byte("my_"), []byte("my")})

	t.Log("test IteratorScan 1")
	list = it.IteratorScan([]byte("my"), []byte("my_key/3"), 100, 1)
	/*for _, v = range list {
		t.Log(string(v))
	}*/
	require.Equal(t, list, [][]byte{[]byte("my_key/4")})

	t.Log("test IteratorScan 0")
	list = it.IteratorScan([]byte("my"), []byte("my_key/3"), 100, 0)
	/*for _, v = range list {
		t.Log(string(v))
	}*/
	require.Equal(t, list, [][]byte{[]byte("my_key/2"), []byte("my_key/1"), []byte("my_"), []byte("my")})
}

// 边界测试
func testDBBoundary2(t *testing.T, db DB) {
	a, _ := hex.DecodeString("ff")
	b, _ := hex.DecodeString("ffff")
	c, _ := hex.DecodeString("ffffff")
	d, _ := hex.DecodeString("ffffffff")
	db.Set(a, []byte("0xff"))
	db.Set(b, []byte("0xffff"))
	db.Set(c, []byte("0xffffff"))
	db.Set(d, []byte("0xffffffff"))

	values := [][]byte{[]byte("0xff"), []byte("0xffff"), []byte("0xffffff"), []byte("0xffffffff")}
	valuesReverse := [][]byte{[]byte("0xffffffff"), []byte("0xffffff"), []byte("0xffff"), []byte("0xff")}
	var v []byte
	_ = v
	it := NewListHelper(db)

	// f为prefix
	t.Log("PrefixScan")
	list := it.PrefixScan(a)
	for i, v := range list {
		t.Log(i, string(v))
	}
	require.Equal(t, list, values)

	t.Log("IteratorScanFromFirst")
	list = it.IteratorScanFromFirst(a, 2)
	for i, v := range list {
		t.Log(i, string(v))
	}
	require.Equal(t, list, values[0:2])
	t.Log("IteratorScanFromLast")
	list = it.IteratorScanFromLast(a, 100)
	for i, v := range list {
		t.Log(i, string(v))
	}
	require.Equal(t, list, valuesReverse)

	t.Log("IteratorScan 1") //seek 第二个
	list = it.IteratorScan(a, b, 100, 1)
	for i, v := range list {
		t.Log(i, string(v))
	}
	require.Equal(t, list, values[1:])

	t.Log("IteratorScan 0")
	list = it.IteratorScan(a, c, 100, 0)
	for i, v := range list {
		t.Log(i, string(v))
	}
	require.Equal(t, list, valuesReverse[1:])
}

func testDBBoundary(t *testing.T, db DB) {
	a, _ := hex.DecodeString("0f")
	c, _ := hex.DecodeString("0fff")
	b, _ := hex.DecodeString("ff")
	d, _ := hex.DecodeString("ffff")
	db.Set(a, []byte("0x0f"))
	db.Set(c, []byte("0x0fff"))
	db.Set(b, []byte("0xff"))
	db.Set(d, []byte("0xffff"))

	var v []byte
	_ = v
	it := NewListHelper(db)

	// f为prefix
	t.Log("PrefixScan")
	list := it.PrefixScan(a)
	require.Equal(t, list, [][]byte{[]byte("0x0f"), []byte("0x0fff")})

	t.Log("IteratorScanFromFirst")
	list = it.IteratorScanFromFirst(a, 2)
	require.Equal(t, list, [][]byte{[]byte("0x0f"), []byte("0x0fff")})

	t.Log("IteratorScanFromLast")
	list = it.IteratorScanFromLast(a, 100)
	require.Equal(t, list, [][]byte{[]byte("0x0fff"), []byte("0x0f")})

	t.Log("IteratorScan 1")
	list = it.IteratorScan(a, a, 100, 1)
	require.Equal(t, list, [][]byte{[]byte("0x0fff")})

	t.Log("IteratorScan 0")
	list = it.IteratorScan(a, a, 100, 0)
	require.Equal(t, list, [][]byte(nil))

	// ff为prefix
	t.Log("PrefixScan")
	list = it.PrefixScan(b)
	require.Equal(t, list, [][]byte{[]byte("0xff"), []byte("0xffff")})

	t.Log("IteratorScanFromFirst")
	list = it.IteratorScanFromFirst(b, 2)
	require.Equal(t, list, [][]byte{[]byte("0xff"), []byte("0xffff")})

	t.Log("IteratorScanFromLast")
	list = it.IteratorScanFromLast(b, 100)
	require.Equal(t, list, [][]byte{[]byte("0xffff"), []byte("0xff")})

	t.Log("IteratorScan 1")
	list = it.IteratorScan(b, b, 100, 1)
	require.Equal(t, list, [][]byte{[]byte("0xffff")})

	t.Log("IteratorScan 0")
	list = it.IteratorScan(b, d, 100, 0)
	require.Equal(t, list, [][]byte{[]byte("0xff")})
}

func testDBIteratorDel(t *testing.T, db DB) {
	for i := 0; i < 1000; i++ {
		k := []byte(fmt.Sprintf("my_key/%010d", i))
		v := []byte(fmt.Sprintf("my_value/%010d", i))
		db.Set(k, v)
	}

	prefix := []byte("my")
	it := db.Iterator(prefix, nil, true)
	defer it.Close()
	for it.Rewind(); it.Valid(); it.Next() {
		t.Log(string(it.Key()), "*********", string(it.Value()))
		batch := db.NewBatch(true)
		batch.Delete(it.Key())
		batch.Write()
	}
}

func testBatch(t *testing.T, db DB) {
	batch := db.NewBatch(false)
	batch.Set([]byte("hello"), []byte("world"))
	err := batch.Write()
	assert.Nil(t, err)

	batch = db.NewBatch(false)
	v, err := db.Get([]byte("hello"))
	assert.Nil(t, err)
	assert.Equal(t, v, []byte("world"))

	//set and del
	batch.Set([]byte("hello1"), []byte("world"))
	batch.Set([]byte("hello2"), []byte("world"))
	batch.Set([]byte("hello3"), []byte("world"))
	batch.Set([]byte("hello4"), []byte("world"))
	batch.Set([]byte("hello5"), []byte("world"))
	batch.Delete([]byte("hello1"))
	err = batch.Write()
	assert.Nil(t, err)
	v, err = db.Get([]byte("hello1"))
	assert.Equal(t, err, types.ErrNotFound)
	assert.Nil(t, v)
}

func testTransaction(t *testing.T, db DB) {
	tx, err := db.BeginTx()
	assert.Nil(t, err)
	tx.Set([]byte("hello1"), []byte("world1"))
	value, err := tx.Get([]byte("hello1"))
	assert.Nil(t, err)
	assert.Equal(t, "world1", string(value))
	tx.Rollback()
	value, err = db.Get([]byte("hello1"))
	assert.Equal(t, types.ErrNotFound, err)
	assert.Equal(t, []byte(nil), value)

	tx, err = db.BeginTx()
	assert.Nil(t, err)
	tx.Set([]byte("hello2"), []byte("world2"))
	value, err = tx.Get([]byte("hello2"))
	assert.Nil(t, err)
	assert.Equal(t, "world2", string(value))
	err = tx.Commit()
	assert.Nil(t, err)
	value, err = db.Get([]byte("hello2"))
	assert.Nil(t, err)
	assert.Equal(t, "world2", string(value))
}
