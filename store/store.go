package store

//store package store the world - state data
import (
	"code.aliyun.com/chain33/chain33/common"
	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/common/mavl"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
	lru "github.com/hashicorp/golang-lru"
	log "github.com/inconshreveable/log15"
)

/*
模块主要的功能：

//批量写
1. EventStoreSet(stateHash, (k1,v1),(k2,v2),(k3,v3)) -> 返回 stateHash

//批量读
2. EventStoreGet(stateHash, k1,k2,k3)

*/

var slog = log.New("module", "store")

func SetLogLevel(level string) {
	common.SetLogLevel(level)
}

func DisableLog() {
	slog.SetHandler(log.DiscardHandler())
}

type Store struct {
	db     dbm.DB
	client queue.Client
	done   chan struct{}
	trees  map[string]*mavl.MAVLTree
	cache  *lru.Cache
}

//driver
//dbpath
func New(cfg *types.Store) *Store {
	db := dbm.NewDB("store", cfg.Driver, cfg.DbPath, 256)
	store := &Store{db: db}
	store.trees = make(map[string]*mavl.MAVLTree)
	store.done = make(chan struct{}, 1)
	store.cache, _ = lru.New(10)
	return store
}

func (store *Store) Close() {
	store.client.Close()
	<-store.done
	store.db.Close()
	slog.Info("store module closed")
}

func (store *Store) SetQueueClient(client queue.Client) {
	store.client = client
	client.Sub("store")

	//recv 消息的处理
	go func() {
		for msg := range store.client.Recv() {
			slog.Debug("stroe recv", "msg", msg)
			store.processMessage(msg)
		}
		store.done <- struct{}{}
	}()
}

func (store *Store) processMessage(msg queue.Message) {
	client := store.client
	if msg.Ty == types.EventStoreSet {
		datas := msg.GetData().(*types.StoreSet)
		hash := mavl.SetKVPair(store.db, datas)
		msg.Reply(client.NewMessage("", types.EventStoreSetReply, &types.ReplyHash{hash}))
	} else if msg.Ty == types.EventStoreGet {
		var tree *mavl.MAVLTree
		var err error
		datas := msg.GetData().(*types.StoreGet)
		values := make([][]byte, len(datas.Keys))
		search := string(datas.StateHash)
		if data, ok := store.cache.Get(search); ok {
			tree = data.(*mavl.MAVLTree)
		} else if data, ok := store.trees[search]; ok {
			tree = data
		} else {
			tree = mavl.NewMAVLTree(store.db)
			err = tree.Load(datas.StateHash)
			if err == nil {
				store.cache.Add(search, tree)
			}
			slog.Debug("store get tree", "err", err)
		}
		if err == nil {
			for i := 0; i < len(datas.Keys); i++ {
				_, value, exit := tree.Get(datas.Keys[i])
				if exit {
					values[i] = value
				}
			}
		}
		msg.Reply(client.NewMessage("", types.EventStoreGetReply, &types.StoreReplyValue{values}))
	} else if msg.Ty == types.EventStoreMemSet { //只是在内存中set 一下，并不改变状态
		datas := msg.GetData().(*types.StoreSet)
		tree := mavl.NewMAVLTree(store.db)
		tree.Load(datas.StateHash)
		for i := 0; i < len(datas.KV); i++ {
			tree.Set(datas.KV[i].Key, datas.KV[i].Value)
		}
		hash := tree.Hash()
		store.trees[string(hash)] = tree
		if len(store.trees) > 100 {
			slog.Error("too many trees in cache")
		}
		msg.Reply(client.NewMessage("", types.EventStoreSetReply, &types.ReplyHash{hash}))
	} else if msg.Ty == types.EventStoreCommit { //把内存中set 的交易 commit
		hash := msg.GetData().(*types.ReqHash)
		tree, ok := store.trees[string(hash.Hash)]
		if !ok {
			slog.Error("store commit", "err", types.ErrHashNotFound)
			msg.Reply(client.NewMessage("", types.EventStoreCommit, types.ErrHashNotFound))
			return
		}
		tree.Save()
		delete(store.trees, string(hash.Hash))
		msg.Reply(client.NewMessage("", types.EventStoreCommit, &types.ReplyHash{hash.Hash}))
	} else if msg.Ty == types.EventStoreRollback {
		hash := msg.GetData().(*types.ReqHash)
		_, ok := store.trees[string(hash.Hash)]
		if !ok {
			slog.Error("store rollback", "err", types.ErrHashNotFound)
			msg.Reply(client.NewMessage("", types.EventStoreRollback, types.ErrHashNotFound))
			return
		}
		delete(store.trees, string(hash.Hash))
		msg.Reply(client.NewMessage("", types.EventStoreRollback, &types.ReplyHash{hash.Hash}))
	}
}
