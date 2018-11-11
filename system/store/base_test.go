package store

import (
	"testing"

	"os"

	"github.com/stretchr/testify/assert"
	"github.com/33cn/chain33/common/log"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

var store_cfg0 = &types.Store{"base_test", "leveldb", "/tmp/base_test0", 100}
var store_cfg1 = &types.Store{"base_test", "leveldb", "/tmp/base_test1", 100}

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

func (s *storeChild) Rollback(req *types.ReqHash) ([]byte, error) {
	return []byte{}, nil
}

func (s *storeChild) Del(req *types.StoreDel) ([]byte, error) {
	return []byte{}, nil
}

func (s *storeChild) IterateRangeByStateHash(statehash []byte, start []byte, end []byte, ascending bool, fn func(key, value []byte) bool) {

}

func (s *storeChild) ProcEvent(msg queue.Message) {}

func init() {
	log.SetLogLevel("error")
}

func TestBaseStore_NewClose(t *testing.T) {
	os.RemoveAll(store_cfg0.DbPath)
	store := NewBaseStore(store_cfg0)
	assert.NotNil(t, store)

	db := store.GetDB()
	assert.NotNil(t, db)

	store.Close()
}

func TestBaseStore_Queue(t *testing.T) {
	os.RemoveAll(store_cfg1.DbPath)
	store := NewBaseStore(store_cfg1)
	assert.NotNil(t, store)

	var q = queue.New("channel")
	store.SetQueueClient(q.Client())
	queueClinet := store.GetQueueClient()

	child := &storeChild{}
	store.SetChild(child)

	var kv []*types.KeyValue
	kv = append(kv, &types.KeyValue{[]byte("k1"), []byte("v1")})
	kv = append(kv, &types.KeyValue{[]byte("k2"), []byte("v2")})
	datas := &types.StoreSet{
		EmptyRoot[:],
		kv,
		0}
	set := &types.StoreSetWithSync{datas, true}
	msg := queueClinet.NewMessage("store", types.EventStoreSet, set)
	err := queueClinet.Send(msg, true)
	assert.Nil(t, err)
	resp, err := queueClinet.Wait(msg)
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int64(types.EventStoreSetReply), resp.Ty)

	get := &types.StoreGet{EmptyRoot[:], [][]byte{}}
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

	commit := &types.ReqHash{EmptyRoot[:]}
	msg = queueClinet.NewMessage("store", types.EventStoreCommit, commit)
	err = queueClinet.Send(msg, true)
	assert.Nil(t, err)
	resp, err = queueClinet.Wait(msg)
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int64(types.EventStoreCommit), resp.Ty)

	rollback := &types.ReqHash{EmptyRoot[:]}
	msg = queueClinet.NewMessage("store", types.EventStoreRollback, rollback)
	err = queueClinet.Send(msg, true)
	assert.Nil(t, err)
	resp, err = queueClinet.Wait(msg)
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int64(types.EventStoreRollback), resp.Ty)

	totalCoins := &types.IterateRangeByStateHash{EmptyRoot[:], []byte(""), []byte(""), 100}
	msg = queueClinet.NewMessage("store", types.EventStoreGetTotalCoins, totalCoins)
	err = queueClinet.Send(msg, true)
	assert.Nil(t, err)
	resp, err = queueClinet.Wait(msg)
	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int64(types.EventGetTotalCoinsReply), resp.Ty)

}
