// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package store

import (
	"os"
	"testing"

	"github.com/33cn/chain33/common/log"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

var storecfg0 = &types.Store{Name: "base_test", Driver: "leveldb", DbPath: "/tmp/base_test0", DbCache: 100}
var storecfg1 = &types.Store{Name: "base_test", Driver: "leveldb", DbPath: "/tmp/base_test1", DbCache: 100}

type storeChild struct {
}

func (s *storeChild) Set(datas *types.StoreSet, sync bool) ([]byte, error) {
	return []byte{}, nil
}

func (s *storeChild) Get(datas *types.StoreGet) [][]byte {
	return [][]byte{}
}

func (s *storeChild) MemSet(datas *types.StoreSet, sync bool) ([]byte, error) {
	return []byte{}, nil
}

func (s *storeChild) Commit(hash *types.ReqHash) ([]byte, error) {
	return []byte{}, nil
}

func (s *storeChild) MemSetUpgrade(datas *types.StoreSet, sync bool) ([]byte, error) {
	return []byte{}, nil
}

func (s *storeChild) CommitUpgrade(req *types.ReqHash) ([]byte, error) {
	return []byte{}, nil
}

func (s *storeChild) Rollback(req *types.ReqHash) ([]byte, error) {
	return []byte{}, nil
}

func (s *storeChild) Del(req *types.StoreDel) ([]byte, error) {
	return []byte{}, nil
}

func (s *storeChild) IterateRangeByStateHash(statehash []byte, start []byte, end []byte, ascending bool, fn func(key, value []byte) bool) {

}

func (s *storeChild) ProcEvent(msg *queue.Message) {}

func init() {
	log.SetLogLevel("error")
}

func TestBaseStore_NewClose(t *testing.T) {
	os.RemoveAll(storecfg0.DbPath)
	store := NewBaseStore(storecfg0)
	assert.NotNil(t, store)

	db := store.GetDB()
	assert.NotNil(t, db)

	store.Close()
}

func TestBaseStore_Queue(t *testing.T) {
	os.RemoveAll(storecfg1.DbPath)
	store := NewBaseStore(storecfg1)
	assert.NotNil(t, store)

	var q = queue.New("channel")
	store.SetQueueClient(q.Client())
	queueClinet := store.GetQueueClient()

	child := &storeChild{}
	store.SetChild(child)

	var kv []*types.KeyValue
	kv = append(kv, &types.KeyValue{Key: []byte("k1"), Value: []byte("v1")})
	kv = append(kv, &types.KeyValue{Key: []byte("k2"), Value: []byte("v2")})
	datas := &types.StoreSet{
		StateHash: EmptyRoot[:],
		KV:        kv,
	}
	set := &types.StoreSetWithSync{Storeset: datas, Sync: true}
	msg := queueClinet.NewMessage("store", types.EventStoreSet, set)
	err := queueClinet.Send(msg, true)
	assert.Nil(t, err)
	resp, err := queueClinet.Wait(msg)
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int64(types.EventStoreSetReply), resp.Ty)

	get := &types.StoreGet{StateHash: EmptyRoot[:], Keys: [][]byte{}}
	msg = queueClinet.NewMessage("store", types.EventStoreGet, get)
	err = queueClinet.Send(msg, true)
	assert.Nil(t, err)
	resp, err = queueClinet.Wait(msg)
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int64(types.EventStoreGetReply), resp.Ty)

	memset := set
	msg = queueClinet.NewMessage("store", types.EventStoreMemSet, memset)
	err = queueClinet.Send(msg, true)
	assert.Nil(t, err)
	resp, err = queueClinet.Wait(msg)
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int64(types.EventStoreSetReply), resp.Ty)

	commit := &types.ReqHash{Hash: EmptyRoot[:]}
	msg = queueClinet.NewMessage("store", types.EventStoreCommit, commit)
	err = queueClinet.Send(msg, true)
	assert.Nil(t, err)
	resp, err = queueClinet.Wait(msg)
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int64(types.EventStoreCommit), resp.Ty)

	rollback := &types.ReqHash{Hash: EmptyRoot[:]}
	msg = queueClinet.NewMessage("store", types.EventStoreRollback, rollback)
	err = queueClinet.Send(msg, true)
	assert.Nil(t, err)
	resp, err = queueClinet.Wait(msg)
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int64(types.EventStoreRollback), resp.Ty)

	totalCoins := &types.IterateRangeByStateHash{
		StateHash: EmptyRoot[:],
		Start:     []byte(""),
		End:       []byte(""),
		Count:     100,
	}
	msg = queueClinet.NewMessage("store", types.EventStoreGetTotalCoins, totalCoins)
	err = queueClinet.Send(msg, true)
	assert.Nil(t, err)
	resp, err = queueClinet.Wait(msg)
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int64(types.EventGetTotalCoinsReply), resp.Ty)

	del := &types.StoreDel{StateHash: EmptyRoot[:], Height: 0}
	msg = queueClinet.NewMessage("store", types.EventStoreDel, del)
	err = queueClinet.Send(msg, true)
	assert.Nil(t, err)
	resp, err = queueClinet.Wait(msg)
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int64(types.EventStoreDel), resp.Ty)

	list := &types.StoreList{StateHash: EmptyRoot[:]}
	msg = queueClinet.NewMessage("store", types.EventStoreList, list)
	err = queueClinet.Send(msg, true)
	assert.Nil(t, err)
	resp, err = queueClinet.Wait(msg)
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int64(types.EventStoreListReply), resp.Ty)

}

func TestSubStore(t *testing.T) {
	storelistQuery := NewStoreListQuery(&storeChild{}, &types.StoreList{StateHash: EmptyRoot[:], Mode: 1})
	ok := storelistQuery.IterateCallBack([]byte("abc"), nil)
	_ = storelistQuery.Run()
	assert.True(t, ok)

	storelistQuery = NewStoreListQuery(&storeChild{}, &types.StoreList{StateHash: EmptyRoot[:], Count: 2, Mode: 1})
	ok = storelistQuery.IterateCallBack([]byte("abc"), nil)
	assert.False(t, ok)

	storelistQuery = NewStoreListQuery(&storeChild{}, &types.StoreList{StateHash: EmptyRoot[:], Suffix: []byte("bc"), Mode: 2})
	ok = storelistQuery.IterateCallBack([]byte("abc"), nil)
	assert.True(t, ok)
}

func TestRegAndLoad(t *testing.T) {
	Reg("test", func(cfg *types.Store, sub []byte) queue.Module {
		return nil
	})

	_, err := Load("test")
	assert.NoError(t, err)

	_, err = Load("test2")
	assert.Equal(t, types.ErrNotFound, err)
}
