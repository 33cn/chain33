package blockchain_test

import (
	"testing"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util/testnode"
	"github.com/stretchr/testify/assert"
)

func TestLocalDBRollback(t *testing.T) {
	mock33 := testnode.New("", nil)
	defer mock33.Close()
	//测试localdb
	api := mock33.GetAPI()
	param := &types.LocalDBGet{}
	param.Keys = append(param.Keys, []byte("hello"))
	values, err := api.LocalGet(param)
	assert.Equal(t, err, nil)
	assert.Equal(t, 1, len(values.Values))
	assert.Nil(t, values.Values[0])

	param2 := &types.LocalDBSet{}
	param2.KV = append(param2.KV, &types.KeyValue{Key: []byte("hello"), Value: []byte("world")})
	err = api.LocalSet(param2)
	assert.Equal(t, err, types.ErrNotSetInTransaction)

	//set in transaction
	id, err := api.LocalNew(nil)
	assert.Nil(t, err)
	assert.True(t, id.Data > 0)

	err = api.LocalClose(id)
	assert.Nil(t, err)

	id, err = api.LocalNew(nil)
	assert.Nil(t, err)
	assert.True(t, id.Data > 0)

	err = api.LocalBegin(id)
	assert.Nil(t, err)

	param2 = &types.LocalDBSet{Txid: id.Data}
	param2.KV = append(param2.KV, &types.KeyValue{Key: []byte("hello"), Value: []byte("world")})
	err = api.LocalSet(param2)
	assert.Nil(t, err)

	values, err = api.LocalGet(param)
	assert.Equal(t, err, nil)
	assert.Equal(t, 1, len(values.Values))
	assert.Nil(t, values.Values[0])

	param.Txid = id.Data
	values, err = api.LocalGet(param)
	assert.Equal(t, err, nil)
	assert.Equal(t, 1, len(values.Values))
	assert.Equal(t, []byte("world"), values.Values[0])

	param2 = &types.LocalDBSet{Txid: id.Data}
	param2.KV = append(param2.KV, &types.KeyValue{Key: []byte("hello2"), Value: []byte("world2")})
	err = api.LocalSet(param2)
	assert.Nil(t, err)

	param.Txid = id.Data
	param.Keys = append(param.Keys, []byte("hello2"))
	param.Keys = append(param.Keys, []byte("hello3"))

	values, err = api.LocalGet(param)
	assert.Equal(t, err, nil)
	assert.Equal(t, 3, len(values.Values))
	assert.Equal(t, []byte("world"), values.Values[0])
	assert.Equal(t, []byte("world2"), values.Values[1])
	assert.Nil(t, values.Values[2])

	list := &types.LocalDBList{
		Txid:      id.Data,
		Prefix:    []byte("hello"),
		Direction: 1,
	}
	values, err = api.LocalList(list)
	assert.Equal(t, err, nil)
	assert.Equal(t, 2, len(values.Values))
	assert.Equal(t, []byte("world"), values.Values[0])
	assert.Equal(t, []byte("world2"), values.Values[1])

	err = api.LocalRollback(id)
	assert.Nil(t, err)

	list = &types.LocalDBList{
		Txid:      id.Data,
		Prefix:    []byte("hello"),
		Direction: 1,
	}
	_, err = api.LocalList(list)
	assert.Nil(t, err)

	list = &types.LocalDBList{
		Prefix:    []byte("hello"),
		Direction: 1,
	}
	values, err = api.LocalList(list)
	assert.Equal(t, err, nil)
	assert.Equal(t, 0, len(values.Values))
}

func TestLocalDBCommit(t *testing.T) {
	mock33 := testnode.New("", nil)
	defer mock33.Close()

	//测试localdb
	api := mock33.GetAPI()
	param := &types.LocalDBGet{}
	param.Keys = append(param.Keys, []byte("hello"))
	values, err := api.LocalGet(param)
	assert.Equal(t, err, nil)
	assert.Equal(t, 1, len(values.Values))
	assert.Nil(t, values.Values[0])

	param2 := &types.LocalDBSet{}
	param2.KV = append(param2.KV, &types.KeyValue{Key: []byte("hello"), Value: []byte("world")})
	err = api.LocalSet(param2)
	assert.Equal(t, err, types.ErrNotSetInTransaction)

	//set in transaction
	id, err := api.LocalNew(nil)
	assert.Nil(t, err)
	assert.True(t, id.Data > 0)

	err = api.LocalBegin(id)
	assert.Nil(t, err)

	param2 = &types.LocalDBSet{Txid: id.Data}
	param2.KV = append(param2.KV, &types.KeyValue{Key: []byte("hello"), Value: []byte("world")})
	err = api.LocalSet(param2)
	assert.Nil(t, err)

	values, err = api.LocalGet(param)
	assert.Equal(t, err, nil)
	assert.Equal(t, 1, len(values.Values))
	assert.Nil(t, values.Values[0])

	param.Txid = id.Data
	values, err = api.LocalGet(param)
	assert.Equal(t, err, nil)
	assert.Equal(t, 1, len(values.Values))
	assert.Equal(t, []byte("world"), values.Values[0])

	param2 = &types.LocalDBSet{Txid: id.Data}
	param2.KV = append(param2.KV, &types.KeyValue{Key: []byte("hello2"), Value: []byte("world2")})
	err = api.LocalSet(param2)
	assert.Nil(t, err)

	param.Txid = id.Data
	param.Keys = append(param.Keys, []byte("hello2"))
	values, err = api.LocalGet(param)
	assert.Equal(t, err, nil)
	assert.Equal(t, 2, len(values.Values))
	assert.Equal(t, []byte("world"), values.Values[0])
	assert.Equal(t, []byte("world2"), values.Values[1])

	list := &types.LocalDBList{
		Txid:      id.Data,
		Prefix:    []byte("hello"),
		Direction: 1,
	}
	values, err = api.LocalList(list)
	assert.Equal(t, err, nil)
	assert.Equal(t, 2, len(values.Values))
	assert.Equal(t, []byte("world"), values.Values[0])
	assert.Equal(t, []byte("world2"), values.Values[1])

	err = api.LocalCommit(id)
	assert.Nil(t, err)

	err = api.LocalClose(id)
	assert.Nil(t, err)

	list = &types.LocalDBList{
		Txid:      id.Data,
		Prefix:    []byte("hello"),
		Direction: 1,
	}
	_, err = api.LocalList(list)
	assert.Equal(t, err, common.ErrPointerNotFound)

	//系统只读，无法写入数据
	list = &types.LocalDBList{
		Prefix:    []byte("hello"),
		Direction: 1,
	}
	values, err = api.LocalList(list)
	assert.Equal(t, err, nil)
	assert.Equal(t, 0, len(values.Values))
}
