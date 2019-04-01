// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package store

import (
	dbm "github.com/33cn/chain33/common/db"
	clog "github.com/33cn/chain33/common/log"
	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
)

/*
模块主要的功能：

//批量写
1. EventStoreSet(stateHash, (k1,v1),(k2,v2),(k3,v3)) -> 返回 stateHash

//批量读
2. EventStoreGet(stateHash, k1,k2,k3)

*/

var slog = log.New("module", "store")

// EmptyRoot mavl树空的根hash
var EmptyRoot [32]byte

// SetLogLevel set log level
func SetLogLevel(level string) {
	clog.SetLogLevel(level)
}

// DisableLog disable log
func DisableLog() {
	slog.SetHandler(log.DiscardHandler())
}

// SubStore  store db的操作接口
type SubStore interface {
	Set(datas *types.StoreSet, sync bool) ([]byte, error)
	Get(datas *types.StoreGet) [][]byte
	MemSet(datas *types.StoreSet, sync bool) ([]byte, error)
	Commit(hash *types.ReqHash) ([]byte, error)
	Rollback(req *types.ReqHash) ([]byte, error)
	Del(req *types.StoreDel) ([]byte, error)
	IterateRangeByStateHash(statehash []byte, start []byte, end []byte, ascending bool, fn func(key, value []byte) bool)
	ProcEvent(msg *queue.Message)

	//用升级本地交易构建store
	MemSetUpgrade(datas *types.StoreSet, sync bool) ([]byte, error)
	CommitUpgrade(hash *types.ReqHash) ([]byte, error)
}

// BaseStore 基础的store结构体
type BaseStore struct {
	db      dbm.DB
	qclient queue.Client
	done    chan struct{}
	child   SubStore
}

// NewBaseStore new base store struct
func NewBaseStore(cfg *types.Store) *BaseStore {
	db := dbm.NewDB("store", cfg.Driver, cfg.DbPath, cfg.DbCache)
	db.SetCacheSize(102400)
	store := &BaseStore{db: db}
	store.done = make(chan struct{}, 1)
	slog.Info("Enter store " + cfg.Name)
	return store
}

// SetQueueClient set client queue for recv msg
func (store *BaseStore) SetQueueClient(c queue.Client) {
	store.qclient = c
	store.qclient.Sub("store")
	//recv 消息的处理
	go func() {
		for msg := range store.qclient.Recv() {
			//slog.Debug("store recv", "msg", msg)
			store.processMessage(msg)
			//slog.Debug("store process end", "msg.id", msg.Id)
		}
		store.done <- struct{}{}
	}()
}

//Wait wait for basestore ready
func (store *BaseStore) Wait() {}

func (store *BaseStore) processMessage(msg *queue.Message) {
	client := store.qclient
	if msg.Ty == types.EventStoreSet {
		go func() {
			datas := msg.GetData().(*types.StoreSetWithSync)
			hash, err := store.child.Set(datas.Storeset, datas.Sync)
			if err != nil {
				msg.Reply(client.NewMessage("", types.EventStoreSetReply, err))
				return
			}
			msg.Reply(client.NewMessage("", types.EventStoreSetReply, &types.ReplyHash{Hash: hash}))
		}()
	} else if msg.Ty == types.EventStoreGet {
		go func() {
			datas := msg.GetData().(*types.StoreGet)
			values := store.child.Get(datas)
			msg.Reply(client.NewMessage("", types.EventStoreGetReply, &types.StoreReplyValue{Values: values}))
		}()
	} else if msg.Ty == types.EventStoreMemSet { //只是在内存中set 一下，并不改变状态
		go func() {
			datas := msg.GetData().(*types.StoreSetWithSync)
			var hash []byte
			var err error
			if datas.Upgrade {
				hash, err = store.child.MemSetUpgrade(datas.Storeset, datas.Sync)
			} else {
				hash, err = store.child.MemSet(datas.Storeset, datas.Sync)
			}
			if err != nil {
				msg.Reply(client.NewMessage("", types.EventStoreSetReply, err))
				return
			}
			msg.Reply(client.NewMessage("", types.EventStoreSetReply, &types.ReplyHash{Hash: hash}))
		}()
	} else if msg.Ty == types.EventStoreCommit { //把内存中set 的交易 commit
		go func() {
			req := msg.GetData().(*types.ReqHash)
			var hash []byte
			var err error
			if req.Upgrade {
				hash, err = store.child.CommitUpgrade(req)
			} else {
				hash, err = store.child.Commit(req)
			}
			if hash == nil {
				msg.Reply(client.NewMessage("", types.EventStoreCommit, types.ErrHashNotFound))
				if err == types.ErrDataBaseDamage { //如果是数据库写失败，需要上报给用户
					go util.ReportErrEventToFront(slog, client, "store", "wallet", err)
				}
			} else {
				msg.Reply(client.NewMessage("", types.EventStoreCommit, &types.ReplyHash{Hash: hash}))
			}
		}()
	} else if msg.Ty == types.EventStoreRollback {
		go func() {
			req := msg.GetData().(*types.ReqHash)
			hash, err := store.child.Rollback(req)
			if err != nil {
				msg.Reply(client.NewMessage("", types.EventStoreRollback, types.ErrHashNotFound))
			} else {
				msg.Reply(client.NewMessage("", types.EventStoreRollback, &types.ReplyHash{Hash: hash}))
			}
		}()
	} else if msg.Ty == types.EventStoreGetTotalCoins {
		go func() {
			req := msg.GetData().(*types.IterateRangeByStateHash)
			resp := &types.ReplyGetTotalCoins{}
			resp.Count = req.Count
			store.child.IterateRangeByStateHash(req.StateHash, req.Start, req.End, true, resp.IterateRangeByStateHash)
			msg.Reply(client.NewMessage("", types.EventGetTotalCoinsReply, resp))
		}()
	} else if msg.Ty == types.EventStoreDel {
		go func() {
			req := msg.GetData().(*types.StoreDel)
			hash, err := store.child.Del(req)
			if err != nil {
				msg.Reply(client.NewMessage("", types.EventStoreDel, types.ErrHashNotFound))
			} else {
				msg.Reply(client.NewMessage("", types.EventStoreDel, &types.ReplyHash{Hash: hash}))
			}
		}()
	} else if msg.Ty == types.EventStoreList {
		go func() {
			req := msg.GetData().(*types.StoreList)
			query := NewStoreListQuery(store.child, req)
			msg.Reply(client.NewMessage("", types.EventStoreListReply, query.Run()))
		}()
	} else {
		go store.child.ProcEvent(msg)
	}
}

// SetChild 设置BaseStore中的子存储参数
func (store *BaseStore) SetChild(sub SubStore) {
	store.child = sub
}

// Close 关闭BaseStore 相关资源包括数据库、client等
func (store *BaseStore) Close() {
	if store.qclient != nil {
		store.qclient.Close()
		<-store.done
	}
	store.db.Close()
}

// GetDB 返回 store db
func (store *BaseStore) GetDB() dbm.DB {
	return store.db
}

// GetQueueClient 返回store模块的client
func (store *BaseStore) GetQueueClient() queue.Client {
	return store.qclient
}

// NewStoreListQuery new store list query object
func NewStoreListQuery(store SubStore, req *types.StoreList) *StorelistQuery {
	reply := &types.StoreListReply{Start: req.Start, End: req.End, Suffix: req.Suffix, Count: req.Count, Mode: req.Mode}
	return &StorelistQuery{StoreListReply: reply, req: req, store: store}
}

// StorelistQuery defines a type store list query
type StorelistQuery struct {
	store SubStore
	req   *types.StoreList
	*types.StoreListReply
}

// Run store list query
func (t *StorelistQuery) Run() *types.StoreListReply {
	t.store.IterateRangeByStateHash(t.req.StateHash, t.req.Start, t.req.End, true, t.IterateCallBack)
	return t.StoreListReply
}

// IterateCallBack store list query iterate callback
func (t *StorelistQuery) IterateCallBack(key, value []byte) bool {
	if t.Mode == 1 { //[start, end)模式
		if t.Num >= t.Count {
			t.NextKey = key
			return true
		}
		t.Num++
		t.Keys = append(t.Keys, cloneByte(key))
		t.Values = append(t.Values, cloneByte(value))
		return false
	}
	if t.Mode == 2 { //prefix + suffix模式，要对按prefix得到的数据key进行suffix的判断，符合条件的数据才是最终要的数据
		if len(key) > len(t.Suffix) {
			if string(key[len(key)-len(t.Suffix):]) == string(t.Suffix) {
				t.Num++
				t.Keys = append(t.Keys, cloneByte(key))
				t.Values = append(t.Values, cloneByte(value))
				if t.Num >= t.Count {
					t.NextKey = key
					return true
				}
				return false
			}
			return false
		}
		return false
	}
	slog.Error("StoreListReply.IterateCallBack unsupported mode", "mode", t.Mode)
	return true
}

func cloneByte(v []byte) []byte {
	value := make([]byte, len(v))
	copy(value, v)
	return value
}
