package store

//store package store the world - state data
import (
	"errors"

	"code.aliyun.com/chain33/chain33/common"
	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/common/mavl"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
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
var zeroHash [32]byte
var ErrHashNotFound = errors.New("ErrHashNotFound")

func SetLogLevel(level string) {
	common.SetLogLevel(level)
}

func DisableLog() {
	slog.SetHandler(log.DiscardHandler())
}

type Store struct {
	db      dbm.DB
	qclient queue.IClient
	trees   map[string]*mavl.MAVLTree
}

//driver
//dbpath
func New(cfg *types.Store) *Store {
	db := dbm.NewDB("store", cfg.Driver, cfg.DbPath)
	store := &Store{db: db}
	store.trees = make(map[string]*mavl.MAVLTree)
	return store
}

func (store *Store) Close() {
	store.db.Close()
	slog.Info("store module closed")
}

func (store *Store) SetQueue(q *queue.Queue) {
	store.qclient = q.GetClient()
	client := store.qclient
	client.Sub("store")

	//recv 消息的处理
	go func() {
		for msg := range client.Recv() {
			slog.Info("stroe recv", "msg", msg)
			if msg.Ty == types.EventStoreSet {
				datas := msg.GetData().(*types.StoreSet)
				hash := mavl.SetKVPair(store.db, datas)
				//mavl.PrintTreeLeaf(store.db, hash)
				msg.Reply(client.NewMessage("", types.EventStoreSetReply, &types.ReplyHash{hash}))
			} else if msg.Ty == types.EventStoreGet {
				datas := msg.GetData().(*types.StoreGet)
				tree, ok := store.trees[string(datas.StateHash)]
				var err error
				values := make([][]byte, len(datas.Keys))
				if !ok {
					tree = mavl.NewMAVLTree(store.db)
					err = tree.Load(datas.StateHash)
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
				msg.Reply(client.NewMessage("", types.EventStoreSetReply, &types.ReplyHash{hash}))
			} else if msg.Ty == types.EventStoreCommit { //把内存中set 的交易 commit
				hash := msg.GetData().(*types.ReqHash)
				tree, ok := store.trees[string(hash.Hash)]
				if !ok {
					slog.Error("store commit", "err", ErrHashNotFound)
					msg.Reply(client.NewMessage("", types.EventStoreCommit, ErrHashNotFound))
				}
				tree.Save()
				delete(store.trees, string(hash.Hash))
				msg.Reply(client.NewMessage("", types.EventStoreCommit, &types.ReplyHash{hash.Hash}))
			} else if msg.Ty == types.EventStoreRollback {
				hash := msg.GetData().(*types.ReqHash)
				_, ok := store.trees[string(hash.Hash)]
				if !ok {
					slog.Error("store rollback", "err", ErrHashNotFound)
					msg.Reply(client.NewMessage("", types.EventStoreRollback, ErrHashNotFound))
				}
				delete(store.trees, string(hash.Hash))
				msg.Reply(client.NewMessage("", types.EventStoreRollback, &types.ReplyHash{hash.Hash}))
			}
		}
	}()
}
