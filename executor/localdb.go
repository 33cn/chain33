package executor

import (
	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

// LocalDB 本地数据库，类似localdb，不加入区块链的状态。
// 数据库只读，不能落盘
// 数据的get set 主要经过 cache
// 如果需要进行list, 那么把get set 的内容加入到 后端数据库
type LocalDB struct {
	cache        *cacheDB
	txcache      *cacheDB
	keys         []string
	intx         bool
	hasbegin     bool
	kvs          []*types.KeyValue
	txid         *types.Int64
	client       queue.Client
	api          client.QueueProtocolAPI
	disableread  bool
	disablewrite bool
}

// NewLocalDB 创建一个新的LocalDB
func NewLocalDB(cli queue.Client, api client.QueueProtocolAPI, readOnly bool) db.KVDB {

	txid, err := api.LocalNew(readOnly)
	if err != nil {
		panic(err)
	}
	return &LocalDB{
		cache:   newcacheDB(1024),
		txcache: newcacheDB(32),
		txid:    txid,
		client:  cli,
		api:     api,
	}
}

// DisableRead 禁止读取LocalDB数据库
func (l *LocalDB) DisableRead() {
	l.disableread = true
}

// DisableWrite 禁止写LocalDB数据库
func (l *LocalDB) DisableWrite() {
	l.disablewrite = true
}

// EnableRead 启动读取LocalDB数据库
func (l *LocalDB) EnableRead() {
	l.disableread = false
}

// EnableWrite 启动写LocalDB数据库
func (l *LocalDB) EnableWrite() {
	l.disablewrite = false
}

func (l *LocalDB) resetTx() {
	l.intx = false
	l.txcache.Reset()
	l.keys = nil
	l.hasbegin = false
}

// StartTx reset state db keys
func (l *LocalDB) StartTx() {
	l.keys = nil
}

// GetSetKeys  get state db set keys
func (l *LocalDB) GetSetKeys() (keys []string) {
	return l.keys
}

// Begin 开始一个事务
func (l *LocalDB) Begin() {
	l.intx = true
	l.keys = nil
	l.txcache.Reset()
	l.hasbegin = false
}

func (l *LocalDB) begin() {
	err := l.api.LocalBegin(l.txid)
	if err != nil {
		panic(err)
	}
}

// 第一次save 的时候，远程做一个 begin 操作，开始事务
func (l *LocalDB) save() error {
	if l.kvs != nil {
		if !l.hasbegin {
			l.begin()
			l.hasbegin = true
		}
		param := &types.LocalDBSet{Txid: l.txid.Data}
		param.KV = l.kvs
		err := l.api.LocalSet(param)
		if err != nil {
			return err
		}
		l.kvs = nil
	}
	return nil
}

// Commit 提交一个事务
func (l *LocalDB) Commit() error {
	l.cache.Merge(l.txcache)
	err := l.save()
	if err != nil {
		return err
	}
	if l.hasbegin {
		err = l.api.LocalCommit(l.txid)
	}
	l.resetTx()
	return err
}

// Close 提交一个事务
func (l *LocalDB) Close() error {
	l.cache.Reset()
	l.resetTx()
	err := l.api.LocalClose(l.txid)
	return err

}

// ResetCache evm 使用
func (l *LocalDB) ResetCache() {
	l.cache.Reset()
	l.resetTx()
}

// Rollback 回滚修改
func (l *LocalDB) Rollback() {
	if l.hasbegin {
		err := l.api.LocalRollback(l.txid)
		if err != nil {
			panic(err)
		}
	}
	l.resetTx()
}

// Get 获取key
func (l *LocalDB) Get(key []byte) ([]byte, error) {
	if l.disableread {
		return nil, types.ErrDisableRead
	}
	if l.intx {
		//hit the txcache
		if value, incache, err := l.txcache.Get(key); incache {
			return value, err
		}
	}
	//hit the cache
	if value, incache, err := l.cache.Get(key); incache {
		return value, err
	}
	//not hit cache, query from db
	query := &types.LocalDBGet{Txid: l.txid.Data, Keys: [][]byte{key}}
	resp, err := l.api.LocalGet(query)
	if err != nil {
		panic(err) //no happen for ever
	}
	if resp.Values == nil || resp.Values[0] == nil {
		l.cache.Set(key, nil)
		return nil, types.ErrNotFound
	}
	l.cache.Set(key, resp.Values[0])
	return resp.Values[0], nil
}

// Set 获取key
func (l *LocalDB) Set(key []byte, value []byte) error {

	if l.disablewrite {
		return types.ErrDisableWrite
	}
	skey := string(key)
	if l.intx {
		l.keys = append(l.keys, skey)
		l.txcache.Set(key, value)
	} else {
		l.cache.Set(key, value)
	}
	l.kvs = append(l.kvs, &types.KeyValue{Key: key, Value: value})
	return nil
}

// List 从数据库中查询数据列表
func (l *LocalDB) List(prefix, key []byte, count, direction int32) ([][]byte, error) {
	if l.disableread {
		return nil, types.ErrDisableRead
	}
	err := l.save()
	if err != nil {
		return nil, err
	}
	query := &types.LocalDBList{Txid: l.txid.Data, Prefix: prefix, Key: key, Count: count, Direction: direction}
	resp, err := l.api.LocalList(query)
	if err != nil {
		panic(err) //no happen for ever
	}
	values := resp.Values
	if values == nil {
		//panic(string(key))
		return nil, types.ErrNotFound
	}
	return values, nil
}

// PrefixCount 从数据库中查询指定前缀的key的数量
func (l *LocalDB) PrefixCount(prefix []byte) (count int64) {
	panic("localdb not support PrefixCount")
}

type cacheDB struct {
	data map[string][]byte
}

func newcacheDB(size int) *cacheDB {
	return &cacheDB{
		data: make(map[string][]byte, size),
	}
}

// return a flag: is key is in cache
func (db *cacheDB) Get(key []byte) (value []byte, incache bool, err error) {
	if db.data == nil {
		return nil, false, types.ErrNotFound
	}
	v, ok := db.data[string(key)]
	if ok && v != nil {
		return v, true, nil
	}
	return nil, ok, types.ErrNotFound
}

func (db *cacheDB) Set(key []byte, value []byte) {
	db.data[string(key)] = value
}

func (db *cacheDB) Reset() {
	for k := range db.data {
		delete(db.data, k)
	}
}

func (db *cacheDB) Merge(db2 *cacheDB) {
	for k, v := range db2.data {
		db.data[k] = v
	}
}
