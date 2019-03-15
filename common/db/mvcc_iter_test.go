// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package db

import (
	"io/ioutil"
	"testing"

	"fmt"
	"os"

	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

func getMVCCIter() *MVCCIter {
	dir, err := ioutil.TempDir("", "goleveldb")
	if err != nil {
		panic(err)
	}
	leveldb, err := NewGoLevelDB("goleveldb", dir, 128)
	if err != nil {
		panic(err)
	}
	return NewMVCCIter(leveldb)
}

func KeyValueList(kvlist ...[2]string) (kvs []*types.KeyValue) {
	for _, list := range kvlist {
		kvs = append(kvs, &types.KeyValue{Key: []byte(list[0]), Value: []byte(list[1])})
	}
	return kvs
}

func saveKVList(db DB, kvlist []*types.KeyValue) {
	for _, v := range kvlist {
		if v.Value == nil {
			db.Delete(v.Key)
		} else {
			db.Set(v.Key, v.Value)
		}
	}
}

func TestAddDelMVCCIter(t *testing.T) {
	dir, err := ioutil.TempDir("", "goleveldb")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录

	m := getMVCCIter()
	defer m.db.Close()
	kvlist, err := m.AddMVCC(KeyValueList([2]string{"0.0", "0/0"}, [2]string{"0.1", "0/1"}), hashN(0), nil, 0)
	assert.Nil(t, err)
	saveKVList(m.db, kvlist)

	kvlist, err = m.AddMVCC(KeyValueList([2]string{"0.0", "0/1"}, [2]string{"1.1", "1/1"}), hashN(1), hashN(0), 1)
	assert.Nil(t, err)
	saveKVList(m.db, kvlist)

	kvlist, err = m.AddMVCC(KeyValueList([2]string{"0.0", "0/2"}, [2]string{"2.1", "2/1"}), hashN(2), hashN(1), 2)
	assert.Nil(t, err)
	saveKVList(m.db, kvlist)

	listhelper := NewListHelper(m)
	values := listhelper.List(nil, nil, 100, 1)
	assert.Equal(t, "0/2", string(values[0]))
	assert.Equal(t, "0/1", string(values[1]))
	assert.Equal(t, "1/1", string(values[2]))
	assert.Equal(t, "2/1", string(values[3]))
	//m.PrintAll()
	//删除最新版本
	kvlist, err = m.DelMVCC(hashN(2), 2, true)
	assert.Nil(t, err)
	saveKVList(m.db, kvlist)
	values = listhelper.List(nil, nil, 100, 1)
	assert.Equal(t, "0/1", string(values[0]))
	assert.Equal(t, "0/1", string(values[1]))
	assert.Equal(t, "1/1", string(values[2]))
	//m.PrintAll()
}

func TestGetAllCoinsMVCCIter(t *testing.T) {
	dir, err := ioutil.TempDir("", "goleveldb")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录

	m := getMVCCIter()
	defer m.db.Close()
	kvlist, err := m.AddMVCC(KeyValueList([2]string{"mavl-coins-bty-exec-16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp", "1"}, [2]string{"mavl-coins-bty-16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTq", "2"}), hashN(0), nil, 0)
	assert.Nil(t, err)
	saveKVList(m.db, kvlist)

	kvlist, err = m.AddMVCC(KeyValueList([2]string{"mavl-coins-bty-16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp", "3"}, [2]string{"mavl-coins-bty-16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTs", "4"}), hashN(1), hashN(0), 1)
	assert.Nil(t, err)
	saveKVList(m.db, kvlist)

	kvlist, err = m.AddMVCC(KeyValueList([2]string{"mavl-coins-bty-exec-16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp", "5"}, [2]string{"mavl-coins-bty-16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTt", "6"}), hashN(2), hashN(1), 2)
	assert.Nil(t, err)
	saveKVList(m.db, kvlist)

	listhelper := NewListHelper(m)
	fmt.Println("---case 1-1----")

	values := listhelper.List([]byte("mavl-coins-bty-"), nil, 100, 1)
	assert.Equal(t, "3", string(values[0]))
	assert.Equal(t, "2", string(values[1]))
	assert.Equal(t, "4", string(values[2]))
	assert.Equal(t, "6", string(values[3]))
	assert.Equal(t, "5", string(values[4]))

	for i := 0; i < len(values); i++ {
		fmt.Println(string(values[i]))
	}

	fmt.Println("---case 1-2----")
	var matchValues [][]byte
	listhelper.IteratorCallback([]byte("mavl-coins-bty-"), nil, 0, 1, func(key, value []byte) bool {
		matchValues = append(matchValues, value)
		return false
	})
	values = matchValues
	assert.Equal(t, "3", string(values[0]))
	assert.Equal(t, "2", string(values[1]))
	assert.Equal(t, "4", string(values[2]))
	assert.Equal(t, "6", string(values[3]))
	assert.Equal(t, "5", string(values[4]))

	for i := 0; i < len(values); i++ {
		fmt.Println(string(values[i]))
	}
	fmt.Println("---case 2-1----")
	//m.PrintAll()
	//删除最新版本
	kvlist, err = m.DelMVCC(hashN(2), 2, true)
	assert.Nil(t, err)
	saveKVList(m.db, kvlist)
	values = listhelper.List(nil, nil, 0, 1)
	assert.Equal(t, 4, len(values))
	assert.Equal(t, "3", string(values[0]))
	assert.Equal(t, "2", string(values[1]))
	assert.Equal(t, "4", string(values[2]))
	assert.Equal(t, "1", string(values[3]))

	for i := 0; i < len(values); i++ {
		fmt.Println(string(values[i]))
	}
	//m.PrintAll()
	fmt.Println("---case 2-2----")
	matchValues = nil
	listhelper.IteratorCallback(([]byte("mavl-coins-bty-")), []byte("mavl-coins-bty-exec-"), 0, 1, func(key, value []byte) bool {
		matchValues = append(matchValues, value)
		return false
	})
	values = matchValues

	for i := 0; i < len(values); i++ {
		fmt.Println(string(values[i]))
	}
	assert.Equal(t, 3, len(values))
	assert.Equal(t, "3", string(values[0]))
	assert.Equal(t, "2", string(values[1]))
	assert.Equal(t, "4", string(values[2]))

	fmt.Println("---case 2-3----")
	matchValues = nil
	listhelper.IteratorCallback(([]byte("mavl-coins-bty-16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTq")), []byte("mavl-coins-bty-exec-"), 0, 1, func(key, value []byte) bool {
		matchValues = append(matchValues, value)
		return false
	})
	values = matchValues

	for i := 0; i < len(values); i++ {
		fmt.Println(string(values[i]))
	}
	assert.Equal(t, 2, len(values))
	//assert.Equal(t, "3", string(values[0]))
	assert.Equal(t, "2", string(values[0]))
	assert.Equal(t, "4", string(values[1]))
}

func TestSimpleMVCCLocalDB(t *testing.T) {
	//use leveldb
	//use localdb
	db3, dir := newGoLevelDB(t)
	defer os.RemoveAll(dir) // clean up
	kvdb := NewLocalDB(db3)
	m := NewSimpleMVCC(kvdb)

	kvlist, err := m.AddMVCC(KeyValueList([2]string{"mavl-coins-bty-exec-16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp", "1"}, [2]string{"mavl-coins-bty-16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTq", "2"}), hashN(0), nil, 0)
	assert.Nil(t, err)
	setKVList(kvdb, kvlist)

	kvlist, err = m.AddMVCC(KeyValueList([2]string{"mavl-coins-bty-16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp", "3"}, [2]string{"mavl-coins-bty-16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTs", "4"}), hashN(1), hashN(0), 1)
	assert.Nil(t, err)
	setKVList(kvdb, kvlist)

	kvlist, err = m.AddMVCC(KeyValueList([2]string{"mavl-coins-bty-exec-16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp", "5"}, [2]string{"mavl-coins-bty-16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTt", "6"}), hashN(2), hashN(1), 2)
	assert.Nil(t, err)
	setKVList(kvdb, kvlist)

	values, err := kvdb.List([]byte(".-mvcc-.d.mavl-coins-bty-"), nil, 100, 1)
	assert.Nil(t, err)
	assert.Equal(t, 6, len(values))
	assert.Equal(t, "3", string(values[0]))
	assert.Equal(t, "2", string(values[1]))
	assert.Equal(t, "4", string(values[2]))
	assert.Equal(t, "6", string(values[3]))
	assert.Equal(t, "1", string(values[4]))
	assert.Equal(t, "5", string(values[5]))

	v, err := m.GetV([]byte("mavl-coins-bty-exec-16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp"), 0)
	assert.Nil(t, err)
	assert.Equal(t, "1", string(v))

	v, err = m.GetV([]byte("mavl-coins-bty-exec-16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp"), 1)
	assert.Nil(t, err)
	assert.Equal(t, "1", string(v))

	v, err = m.GetV([]byte("mavl-coins-bty-exec-16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp"), 2)
	assert.Nil(t, err)
	assert.Equal(t, "5", string(v))

	v, err = m.GetV([]byte("mavl-coins-bty-exec-16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp"), 3)
	assert.Nil(t, err)
	assert.Equal(t, "5", string(v))

}

func setKVList(db KVDB, kvlist []*types.KeyValue) {
	for _, v := range kvlist {
		db.Set(v.Key, v.Value)
	}
}
