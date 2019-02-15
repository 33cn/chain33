package blockchain

import (
	"sync"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

func (chain *BlockChain) procLocalDB(msgtype int64, msg queue.Message, reqnum chan struct{}) bool {
	switch msgtype {
	case types.EventLocalGet:
		go chain.processMsg(msg, reqnum, chain.localGet)
	case types.EventLocalSet:
		go chain.processMsg(msg, reqnum, chain.localSet)
	case types.EventLocalBegin:
		go chain.processMsg(msg, reqnum, chain.localBegin)
	case types.EventLocalCommit:
		go chain.processMsg(msg, reqnum, chain.localCommit)
	case types.EventLocalRollback:
		go chain.processMsg(msg, reqnum, chain.localRollback)
	case types.EventLocalList:
		go chain.processMsg(msg, reqnum, chain.localList)
	case types.EventLocalPrefixCount:
		go chain.processMsg(msg, reqnum, chain.localPrefixCount)
	case types.EventLocalNew:
		go chain.processMsg(msg, reqnum, chain.localNew)
	case types.EventLocalClose:
		go chain.processMsg(msg, reqnum, chain.localClose)
	default:
		return false
	}
	return true
}

func (chain *BlockChain) localGet(msg queue.Message) {
	keys := (msg.Data).(*types.LocalDBGet)
	if keys.Txid == 0 {
		values := chain.blockStore.Get(keys)
		msg.Reply(chain.client.NewMessage("", types.EventLocalReplyValue, values))
		return
	}
	tx, err := common.GetPointer(keys.Txid)
	if err != nil {
		msg.Reply(chain.client.NewMessage("", types.EventLocalReplyValue, err))
		return
	}
	var reply types.LocalReplyValue
	for i := 0; i < len(keys.Keys); i++ {
		key := keys.Keys[i]
		value, err := tx.(db.KVDB).Get(key)
		if err != nil {
			chainlog.Debug("localGet", "i", i, "key", string(key), "err", err)
		}
		reply.Values = append(reply.Values, value)
	}
	msg.Reply(chain.client.NewMessage("", types.EventLocalReplyValue, &reply))
}

//只允许设置 通过 transaction 来 set 信息
func (chain *BlockChain) localSet(msg queue.Message) {
	kvs := (msg.Data).(*types.LocalDBSet)
	if kvs.Txid == 0 {
		msg.Reply(chain.client.NewMessage("", types.EventLocalSet, types.ErrNotSetInTransaction))
		return
	}
	txp, err := common.GetPointer(kvs.Txid)
	if err != nil {
		msg.Reply(chain.client.NewMessage("", types.EventLocalSet, err))
		return
	}
	tx := txp.(db.KVDB)
	for i := 0; i < len(kvs.KV); i++ {
		err := tx.Set(kvs.KV[i].Key, kvs.KV[i].Value)
		if err != nil {
			chainlog.Error("localSet", "i", i, "key", string(kvs.KV[i].Key), "err", err)
		}
	}
	msg.Reply(chain.client.NewMessage("", types.EventLocalSet, nil))
}

//创建 localdb transaction
func (chain *BlockChain) localNew(msg queue.Message) {
	tx := NewLocalDB(chain.blockStore.db)
	id := common.StorePointer(tx)
	msg.Reply(chain.client.NewMessage("", types.EventLocalNew, &types.Int64{Data: id}))
}

//关闭 localdb transaction
func (chain *BlockChain) localClose(msg queue.Message) {
	id := (msg.Data).(*types.Int64).Data
	_, err := common.GetPointer(id)
	common.RemovePointer(id)
	msg.Reply(chain.client.NewMessage("", types.EventLocalClose, err))
}

func (chain *BlockChain) localBegin(msg queue.Message) {
	id := (msg.Data).(*types.Int64).Data
	tx, err := common.GetPointer(id)
	if err != nil {
		msg.Reply(chain.client.NewMessage("", types.EventLocalBegin, err))
		return
	}
	tx.(db.KVDB).Begin()
	msg.Reply(chain.client.NewMessage("", types.EventLocalBegin, nil))
}

func (chain *BlockChain) localCommit(msg queue.Message) {
	id := (msg.Data).(*types.Int64).Data
	tx, err := common.GetPointer(id)
	if err != nil {
		msg.Reply(chain.client.NewMessage("", types.EventLocalCommit, err))
		return
	}
	err = tx.(db.KVDB).Commit()
	msg.Reply(chain.client.NewMessage("", types.EventLocalCommit, err))
}

func (chain *BlockChain) localRollback(msg queue.Message) {
	id := (msg.Data).(*types.Int64).Data
	tx, err := common.GetPointer(id)
	if err != nil {
		msg.Reply(chain.client.NewMessage("", types.EventLocalRollback, err))
		return
	}
	tx.(db.KVDB).Rollback()
	msg.Reply(chain.client.NewMessage("", types.EventLocalRollback, nil))
}

func (chain *BlockChain) localList(msg queue.Message) {
	q := (msg.Data).(*types.LocalDBList)
	var values [][]byte
	if q.Txid > 0 {
		tx, err := common.GetPointer(q.Txid)
		if err != nil {
			msg.Reply(chain.client.NewMessage("", types.EventLocalReplyValue, err))
			return
		}
		values, err = tx.(db.KVDB).List(q.Prefix, q.Key, q.Count, q.Direction)
		if err != nil {
			msg.Reply(chain.client.NewMessage("", types.EventLocalReplyValue, err))
			return
		}
	} else {
		values = db.NewListHelper(chain.blockStore.db).List(q.Prefix, q.Key, q.Count, q.Direction)
	}
	msg.Reply(chain.client.NewMessage("", types.EventLocalReplyValue, &types.LocalReplyValue{Values: values}))
}

//获取指定前缀key的数量
func (chain *BlockChain) localPrefixCount(msg queue.Message) {
	Prefix := (msg.Data).(*types.ReqKey)
	counts := db.NewListHelper(chain.blockStore.db).PrefixCount(Prefix.Key)
	msg.Reply(chain.client.NewMessage("", types.EventLocalReplyValue, &types.Int64{Data: counts}))
}

// LocalDB local db for store key value in local
type LocalDB struct {
	txcache db.DB
	cache   db.DB
	maindb  db.DB
	intx    bool
	mu      sync.RWMutex
}

func newMemDB() db.DB {
	memdb, err := db.NewGoMemDB("", "", 0)
	if err != nil {
		panic(err)
	}
	return memdb
}

// NewLocalDB new local db
func NewLocalDB(maindb db.DB) db.KVDB {
	return &LocalDB{
		cache:   newMemDB(),
		txcache: newMemDB(),
		maindb:  maindb,
	}
}

// Get get value from local db
func (l *LocalDB) Get(key []byte) ([]byte, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	value, err := l.get(key)
	return value, err
}

func (l *LocalDB) get(key []byte) ([]byte, error) {
	if l.intx && l.txcache != nil {
		if value, err := l.txcache.Get(key); err == nil {
			return value, nil
		}
	}
	if value, err := l.cache.Get(key); err == nil {
		return value, nil
	}
	value, err := l.maindb.Get(key)
	if err != nil {
		return nil, err
	}
	err = l.cache.Set(key, value)
	if err != nil {
		panic(err)
	}
	return value, nil
}

// Set set key value to local db
func (l *LocalDB) Set(key []byte, value []byte) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.intx {
		if l.txcache == nil {
			l.txcache = newMemDB()
		}
		setdb(l.txcache, key, value)
	} else {
		setdb(l.cache, key, value)
	}
	return nil
}

// List 从数据库中查询数据列表，set 中的cache 更新不会影响这个list
func (l *LocalDB) List(prefix, key []byte, count, direction int32) ([][]byte, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	dblist := make([]db.IteratorDB, 0)
	if l.txcache != nil {
		dblist = append(dblist, l.txcache)
	}
	if l.cache != nil {
		dblist = append(dblist, l.cache)
	}
	if l.maindb != nil {
		dblist = append(dblist, l.maindb)
	}
	mergedb := db.NewMergedIteratorDB(dblist)
	it := db.NewListHelper(mergedb)
	return it.List(prefix, key, count, direction), nil
}

// PrefixCount 从数据库中查询指定前缀的key的数量
func (l *LocalDB) PrefixCount(prefix []byte) (count int64) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	dblist := make([]db.IteratorDB, 0)
	if l.txcache != nil {
		dblist = append(dblist, l.txcache)
	}
	if l.cache != nil {
		dblist = append(dblist, l.cache)
	}
	if l.maindb != nil {
		dblist = append(dblist, l.maindb)
	}
	mergedb := db.NewMergedIteratorDB(dblist)
	it := db.NewListHelper(mergedb)
	return it.PrefixCount(prefix)
}

//Begin 开启内存事务处理
func (l *LocalDB) Begin() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.intx = true
	l.txcache = nil
}

// Rollback reset tx
func (l *LocalDB) Rollback() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.resetTx()
}

// Commit canche tx
func (l *LocalDB) Commit() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.txcache == nil {
		l.resetTx()
		return nil
	}
	it := l.txcache.Iterator(nil, nil, false)
	for it.Next() {
		l.cache.Set(it.Key(), it.Value())
	}
	l.resetTx()
	return nil
}

func (l *LocalDB) resetTx() {
	l.intx = false
	l.txcache = nil
}

func setdb(d db.DB, key []byte, value []byte) {
	if value == nil {
		err := d.Delete(key)
		if err != nil {
			return
		}
	} else {
		err := d.Set(key, value)
		if err != nil {
			panic(err)
		}
	}
}
