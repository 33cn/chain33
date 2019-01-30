package executor

import (
	"github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

// LocalDB local db for store key value in local
type LocalDB struct {
	db.TransactionDB
	cache   map[string][]byte
	txcache map[string][]byte
	keys    []string
	intx    bool
	client  queue.Client
}

// NewLocalDB new local db
func NewLocalDB(client queue.Client) db.KVDB {
	return &LocalDB{cache: make(map[string][]byte), client: client}
}

// Get get value from local db
func (l *LocalDB) Get(key []byte) ([]byte, error) {
	value, err := l.get(key)
	debugAccount("==lget==", key, value)
	return value, err
}

func (l *LocalDB) get(key []byte) ([]byte, error) {
	skey := string(key)
	if l.intx && l.txcache != nil {
		if value, ok := l.txcache[skey]; ok {
			return value, nil
		}
	}
	if value, ok := l.cache[skey]; ok {
		return value, nil
	}
	if l.client == nil {
		return nil, types.ErrNotFound
	}
	query := &types.LocalDBGet{Keys: [][]byte{key}}
	msg := l.client.NewMessage("blockchain", types.EventLocalGet, query)
	l.client.Send(msg, true)
	resp, err := l.client.Wait(msg)

	if err != nil {
		panic(err) //no happen for ever
	}
	if nil == resp.GetData().(*types.LocalReplyValue).Values {
		return nil, types.ErrNotFound
	}
	value := resp.GetData().(*types.LocalReplyValue).Values[0]
	if value == nil {
		//panic(string(key))
		return nil, types.ErrNotFound
	}
	l.cache[string(key)] = value
	return value, nil
}

// Set set key value to local db
func (l *LocalDB) Set(key []byte, value []byte) error {
	debugAccount("==lset==", key, value)
	skey := string(key)
	if l.intx {
		if l.txcache == nil {
			l.txcache = make(map[string][]byte)
		}
		l.keys = append(l.keys, skey)
		setmap(l.txcache, skey, value)
	} else {
		setmap(l.cache, skey, value)
	}
	return nil
}

// BatchGet batch get values from local db
func (l *LocalDB) BatchGet(keys [][]byte) (values [][]byte, err error) {
	for _, key := range keys {
		v, err := l.Get(key)
		if err != nil && err != types.ErrNotFound {
			return nil, err
		}
		values = append(values, v)
	}
	return values, nil
}

// List 从数据库中查询数据列表，set 中的cache 更新不会影响这个list
func (l *LocalDB) List(prefix, key []byte, count, direction int32) ([][]byte, error) {
	if l.client == nil {
		return nil, types.ErrNotFound
	}
	query := &types.LocalDBList{Prefix: prefix, Key: key, Count: count, Direction: direction}
	msg := l.client.NewMessage("blockchain", types.EventLocalList, query)
	l.client.Send(msg, true)
	resp, err := l.client.Wait(msg)
	if err != nil {
		panic(err) //no happen for ever
	}
	values := resp.GetData().(*types.LocalReplyValue).Values
	if values == nil {
		//panic(string(key))
		return nil, types.ErrNotFound
	}
	return values, nil
}

// PrefixCount 从数据库中查询指定前缀的key的数量
func (l *LocalDB) PrefixCount(prefix []byte) (count int64) {
	if l.client == nil {
		return 0
	}
	query := &types.ReqKey{Key: prefix}
	msg := l.client.NewMessage("blockchain", types.EventLocalPrefixCount, query)
	l.client.Send(msg, true)
	resp, err := l.client.Wait(msg)
	if err != nil {
		panic(err) //no happen for ever
	}
	count = resp.GetData().(*types.Int64).Data
	return
}

//Begin 开启内存事务处理
func (l *LocalDB) Begin() {
	l.intx = true
	l.keys = nil
	l.txcache = nil
}

// Rollback reset tx
func (l *LocalDB) Rollback() {
	l.resetTx()
}

// Commit canche tx
func (l *LocalDB) Commit() {
	for k, v := range l.txcache {
		l.cache[k] = v
	}
	l.resetTx()
}

func (l *LocalDB) resetTx() {
	l.intx = false
	l.txcache = nil
	l.keys = nil
}
