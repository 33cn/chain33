package drivers

//store package store the world - state data
import (
	log "github.com/inconshreveable/log15"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	clog "gitlab.33.cn/chain33/chain33/common/log"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
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
	clog.SetLogLevel(level)
}

func DisableLog() {
	slog.SetHandler(log.DiscardHandler())
}

type SubStore interface {
	Set(datas *types.StoreSet, sync bool) []byte
	Get(datas *types.StoreGet) [][]byte
	MemSet(datas *types.StoreSet, sync bool) []byte
	Commit(hash *types.ReqHash) []byte
	Rollback(req *types.ReqHash) []byte
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
	db := dbm.NewDB("store", cfg.Driver, cfg.DbPath, 128)
	store := &BaseStore{db: db}
	store.done = make(chan struct{}, 1)
	slog.Info("Enter store " + cfg.GetName())
	return store
}

func (store *BaseStore) SetQueueClient(c queue.Client) {
	store.qclient = c
	store.qclient.Sub("store")

	//recv 消息的处理
	go func() {
		for msg := range store.qclient.Recv() {
			slog.Debug("store recv", "msg", msg)
			store.processMessage(msg)
		}
		store.done <- struct{}{}
	}()
}

func (store *BaseStore) processMessage(msg queue.Message) {
	client := store.qclient
	if msg.Ty == types.EventStoreSet {
		datas := msg.GetData().(*types.StoreSetWithSync)
		hash := store.child.Set(datas.Storeset, datas.Sync)
		msg.Reply(client.NewMessage("", types.EventStoreSetReply, &types.ReplyHash{hash}))
	} else if msg.Ty == types.EventStoreGet {
		datas := msg.GetData().(*types.StoreGet)
		values := store.child.Get(datas)
		msg.Reply(client.NewMessage("", types.EventStoreGetReply, &types.StoreReplyValue{values}))
	} else if msg.Ty == types.EventStoreMemSet { //只是在内存中set 一下，并不改变状态
		datas := msg.GetData().(*types.StoreSetWithSync)
		hash := store.child.MemSet(datas.Storeset, datas.Sync)
		msg.Reply(client.NewMessage("", types.EventStoreSetReply, &types.ReplyHash{hash}))
	} else if msg.Ty == types.EventStoreCommit { //把内存中set 的交易 commit
		req := msg.GetData().(*types.ReqHash)
		hash := store.child.Commit(req)
		if hash == nil {
			msg.Reply(client.NewMessage("", types.EventStoreCommit, types.ErrHashNotFound))
		} else {
			msg.Reply(client.NewMessage("", types.EventStoreCommit, &types.ReplyHash{hash}))
		}
	} else if msg.Ty == types.EventStoreRollback {
		req := msg.GetData().(*types.ReqHash)
		hash := store.child.Rollback(req)
		if hash == nil {
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
	} else {
		store.child.ProcEvent(msg)
	}
}

func (store *BaseStore) SetChild(sub SubStore) {
	store.child = sub
}

func (store *BaseStore) Close() {
	store.qclient.Close()
	<-store.done
	store.db.Close()
}

func (store *BaseStore) GetDB() dbm.DB {
	return store.db
}

func (store *BaseStore) GetQueueClient() queue.Client {
	return store.qclient
}
