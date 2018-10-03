package db

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.33.cn/chain33/chain33/types"
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
