// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package store

//store package store the world - state data
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
var EmptyRoot [32]byte

func SetLogLevel(level string) {
	clog.SetLogLevel(level)
}

func DisableLog() {
	slog.SetHandler(log.DiscardHandler())
}

type SubStore interface {
	Set(datas *types.StoreSet, sync bool) ([]byte, error)
	Get(datas *types.StoreGet) [][]byte
	MemSet(datas *types.StoreSet, sync bool) ([]byte, error)
	Commit(hash *types.ReqHash) ([]byte, error)
	Rollback(req *types.ReqHash) ([]byte, error)
	Del(req *types.StoreDel) ([]byte, error)
	IterateRangeByStateHash(statehash []byte, start []byte, end []byte, ascending bool, fn func(key, value []byte) bool)
	ProcEvent(msg queue.Message)
}

type BaseStore struct {
	db      dbm.DB
	qclient queue.Client
	done    chan struct{}
	child   SubStore
}

//driver
//dbpath
func NewBaseStore(cfg *types.Store) *BaseStore {
	db := dbm.NewDB("store", cfg.Driver, cfg.DbPath, cfg.DbCache)
	db.SetCacheSize(102400)
	store := &BaseStore{db: db}
	store.done = make(chan struct{}, 1)
	slog.Info("Enter store " + cfg.Name)
	return store
}

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

func (store *BaseStore) processMessage(msg queue.Message) {
	client := store.qclient
	if msg.Ty == types.EventStoreSet {
		datas := msg.GetData().(*types.StoreSetWithSync)
		hash, err := store.child.Set(datas.Storeset, datas.Sync)
		if err != nil {
			msg.Reply(client.NewMessage("", types.EventStoreSetReply, err))
			return
		}
		msg.Reply(client.NewMessage("", types.EventStoreSetReply, &types.ReplyHash{hash}))
	} else if msg.Ty == types.EventStoreGet {
		datas := msg.GetData().(*types.StoreGet)
		values := store.child.Get(datas)
		msg.Reply(client.NewMessage("", types.EventStoreGetReply, &types.StoreReplyValue{values}))
	} else if msg.Ty == types.EventStoreMemSet { //只是在内存中set 一下，并不改变状态
		datas := msg.GetData().(*types.StoreSetWithSync)
		hash, err := store.child.MemSet(datas.Storeset, datas.Sync)
		if err != nil {
			msg.Reply(client.NewMessage("", types.EventStoreSetReply, err))
			return
		}
		msg.Reply(client.NewMessage("", types.EventStoreSetReply, &types.ReplyHash{hash}))
	} else if msg.Ty == types.EventStoreCommit { //把内存中set 的交易 commit
		req := msg.GetData().(*types.ReqHash)
		hash, err := store.child.Commit(req)
		if hash == nil {
			msg.Reply(client.NewMessage("", types.EventStoreCommit, types.ErrHashNotFound))
			if err == types.ErrDataBaseDamage { //如果是数据库写失败，需要上报给用户
				go util.ReportErrEventToFront(slog, client, "store", "wallet", err)
			}
		} else {
			msg.Reply(client.NewMessage("", types.EventStoreCommit, &types.ReplyHash{hash}))
		}
	} else if msg.Ty == types.EventStoreRollback {
		req := msg.GetData().(*types.ReqHash)
		hash, err := store.child.Rollback(req)
		if err != nil {
			msg.Reply(client.NewMessage("", types.EventStoreRollback, types.ErrHashNotFound))
		} else {
			msg.Reply(client.NewMessage("", types.EventStoreRollback, &types.ReplyHash{hash}))
		}
	} else if msg.Ty == types.EventStoreGetTotalCoins {
		req := msg.GetData().(*types.IterateRangeByStateHash)
		resp := &types.ReplyGetTotalCoins{}
		resp.Count = req.Count
		store.child.IterateRangeByStateHash(req.StateHash, req.Start, req.End, true, resp.IterateRangeByStateHash)
		msg.Reply(client.NewMessage("", types.EventGetTotalCoinsReply, resp))
	} else if msg.Ty == types.EventStoreDel {
		req := msg.GetData().(*types.StoreDel)
		hash, err := store.child.Del(req)
		if err != nil {
			msg.Reply(client.NewMessage("", types.EventStoreDel, types.ErrHashNotFound))
		} else {
			msg.Reply(client.NewMessage("", types.EventStoreDel, &types.ReplyHash{hash}))
		}
	} else {
		store.child.ProcEvent(msg)
	}
}

func (store *BaseStore) SetChild(sub SubStore) {
	store.child = sub
}

func (store *BaseStore) Close() {
	if store.qclient != nil {
		store.qclient.Close()
		<-store.done
	}
	store.db.Close()
}

func (store *BaseStore) GetDB() dbm.DB {
	return store.db
}

func (store *BaseStore) GetQueueClient() queue.Client {
	return store.qclient
}
