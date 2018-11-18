// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package store

import (
	"crypto/rand"
	"fmt"
	"os"
	"testing"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/log"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"

	_ "github.com/33cn/chain33/system"
)

func init() {
	log.SetLogLevel("error")
}

func initEnv() (queue.Queue, queue.Module) {
	var q = queue.New("channel")
	cfg, sub := types.InitCfg("../cmd/chain33/chain33.test.toml")
	s := New(cfg.Store, sub.Store)
	s.SetQueueClient(q.Client())
	return q, s
}

func set(client queue.Client, hash, key, value []byte) ([]byte, error) {
	kv := &types.KeyValue{Key: key, Value: value}
	set := &types.StoreSet{}
	set.StateHash = hash
	set.KV = append(set.KV, kv)
	setwithsync := &types.StoreSetWithSync{Storeset: set, Sync: true}

	msg := client.NewMessage("store", types.EventStoreSet, setwithsync)
	client.Send(msg, true)
	msg, err := client.Wait(msg)
	if err != nil {
		return nil, err
	}
	return msg.GetData().(*types.ReplyHash).GetHash(), nil
}

func setmem(client queue.Client, hash, key, value []byte) ([]byte, error) {
	kv := &types.KeyValue{Key: key, Value: value}
	set := &types.StoreSet{}
	set.StateHash = hash
	set.KV = append(set.KV, kv)
	storeset := &types.StoreSetWithSync{Storeset: set, Sync: true}
	msg := client.NewMessage("store", types.EventStoreMemSet, storeset)
	client.Send(msg, true)
	msg, err := client.Wait(msg)
	if err != nil {
		return nil, err
	}
	return msg.GetData().(*types.ReplyHash).GetHash(), nil
}

func get(client queue.Client, hash, key []byte) ([]byte, error) {
	query := &types.StoreGet{StateHash: hash, Keys: [][]byte{key}}
	msg := client.NewMessage("store", types.EventStoreGet, query)
	client.Send(msg, true)
	msg, err := client.Wait(msg)
	if err != nil {
		return nil, err
	}
	values := msg.GetData().(*types.StoreReplyValue).GetValues()
	return values[0], nil
}

func commit(client queue.Client, hash []byte) ([]byte, error) {
	req := &types.ReqHash{Hash: hash}
	msg := client.NewMessage("store", types.EventStoreCommit, req)
	client.Send(msg, true)
	msg, err := client.Wait(msg)
	if err != nil {
		return nil, err
	}
	hash = msg.GetData().(*types.ReplyHash).GetHash()
	return hash, nil
}

func rollback(client queue.Client, hash []byte) ([]byte, error) {
	req := &types.ReqHash{Hash: hash}
	msg := client.NewMessage("store", types.EventStoreRollback, req)
	client.Send(msg, true)
	msg, err := client.Wait(msg)
	if err != nil {
		return nil, err
	}
	hash = msg.GetData().(*types.ReplyHash).GetHash()
	return hash, nil
}

func TestGetAndSet(t *testing.T) {
	q, s := initEnv()
	client := q.Client()
	var stateHash [32]byte
	//先set一个数
	key := []byte("hello")
	value := []byte("world")

	hash, err := set(client, stateHash[:], key, value)
	if err != nil {
		t.Error(err)
		return
	}

	value2, err := get(client, hash, key)
	if err != nil {
		t.Error(err)
		return
	}
	if string(value2) != string(value) {
		t.Errorf("values not match")
		return
	}
	s.Close()
}

func randstr() string {
	var hash [16]byte
	_, err := rand.Read(hash[:])
	if err != nil {
		panic(err)
	}
	return common.ToHex(hash[:])
}

func TestGetAndSetCommitAndRollback(t *testing.T) {
	q, s := initEnv()
	client := q.Client()
	var stateHash [32]byte
	//先set一个数
	key := []byte("hello" + randstr())
	value := []byte("world")

	hash, err := setmem(client, stateHash[:], key, value)
	if err != nil {
		t.Error(err)
		return
	}

	value2, err := get(client, hash, key)
	if err != nil {
		t.Error(err)
		return
	}
	if string(value2) != string(value) {
		t.Errorf("values not match %s %s %x", string(value2), string(value), hash)
		return
	}

	rollback(client, hash)
	value2, err = get(client, hash, key)
	if err != nil {
		t.Error(err)
		return
	}
	if len(value2) != 0 {
		t.Error(err)
		return
	}

	hash, err = setmem(client, stateHash[:], key, value)
	if err != nil {
		t.Error(err)
		return
	}

	commit(client, hash)

	value2, err = get(client, hash, key)
	if err != nil {
		t.Error(err)
		return
	}
	if string(value2) != string(value) {
		t.Errorf("values not match [%s] [%s] %x", string(value2), string(value), hash)
		return
	}

	s.Close()
}

func BenchmarkGetKey(b *testing.B) {
	q, s := initEnv()
	client := q.Client()
	var stateHash [32]byte
	hash := stateHash[:]
	var err error
	for i := 0; i < 1000; i++ {
		key := []byte(fmt.Sprintf("%020d", i))
		value := []byte(fmt.Sprintf("%020d", i))
		hash, err = set(client, hash, key, value)
		if err != nil {
			b.Error(err)
			return
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := []byte(fmt.Sprintf("%020d", i%1000))
		value := fmt.Sprintf("%020d", i%1000)
		value2, err := get(client, hash, key)
		if err != nil {
			b.Error(err)
			return
		}
		if string(value2) != value {
			b.Error(err)
			return
		}
	}
	s.Close()
}

func BenchmarkSetKeyOneByOne(b *testing.B) {
	q, s := initEnv()
	client := q.Client()
	var stateHash [32]byte
	hash := stateHash[:]
	var err error
	for i := 0; i < b.N; i++ {
		key := []byte(fmt.Sprintf("%020d", i))
		value := []byte(fmt.Sprintf("%020d", i))
		hash, err = set(client, hash, key, value)
		if err != nil {
			b.Error(err)
			return
		}
	}
	s.Close()
}

func BenchmarkSetKey1000(b *testing.B) {
	q, s := initEnv()
	client := q.Client()
	var stateHash [32]byte
	hash := stateHash[:]
	set := &types.StoreSet{}

	for i := 0; i < b.N; i++ {
		key := []byte(fmt.Sprintf("%020d", i))
		value := []byte(fmt.Sprintf("%020d", i))
		kv := &types.KeyValue{Key: key, Value: value}
		if i%1000 == 0 {
			set = &types.StoreSet{}
			set.StateHash = hash
		}
		set.KV = append(set.KV, kv)

		if i > 0 && i%1000 == 0 {
			setwithsync := &types.StoreSetWithSync{Storeset: set, Sync: true}
			msg := client.NewMessage("store", types.EventStoreSet, setwithsync)
			client.Send(msg, true)
			msg, err := client.Wait(msg)
			if err != nil {
				b.Error(err)
				return
			}
			hash = msg.GetData().(*types.ReplyHash).GetHash()
		}
	}
	s.Close()
}

var storecfg1 = &types.Store{Name: "mavl", Driver: "leveldb", DbPath: "/tmp/store_test1", DbCache: 100}

func TestNewMavl(t *testing.T) {
	os.RemoveAll(storecfg1.DbPath)
	store := New(storecfg1, nil)
	assert.NotNil(t, store)
}
