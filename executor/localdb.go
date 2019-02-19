package executor

import (
	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

//LocalDB 本地数据库，类似localdb，不加入区块链的状态。
//数据库只读，不能落盘
//数据的get set 主要经过 cache
//如果需要进行list, 那么把get set 的内容加入到 后端数据库
type LocalDB struct {
	cache   map[string][]byte
	txcache map[string][]byte
	keys    []string
	intx    bool
	kvs     []*types.KeyValue
	txid    *types.Int64
	client  queue.Client
	api     client.QueueProtocolAPI
}

//NewLocalDB 创建一个新的LocalDB
func NewLocalDB(cli queue.Client) db.KVDB {
	api, err := client.New(cli, nil)
	if err != nil {
		panic(err)
	}
	txid, err := api.LocalNew(nil)
	if err != nil {
		panic(err)
	}
	return &LocalDB{
		cache:  make(map[string][]byte),
		txid:   txid,
		client: cli,
		api:    api,
	}
}

func (l *LocalDB) resetTx() {
	l.intx = false
	l.txcache = nil
	l.keys = nil
}

//Begin 开始一个事务
func (l *LocalDB) Begin() {
	l.intx = true
	l.keys = nil
	l.txcache = nil
	err := l.api.LocalBegin(l.txid)
	if err != nil {
		panic(err)
	}
}

func (l *LocalDB) save() error {
	if l.kvs != nil {
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

//Commit 提交一个事务
func (l *LocalDB) Commit() error {
	for k, v := range l.txcache {
		l.cache[k] = v
	}
	err := l.save()
	if err != nil {
		return err
	}
	l.resetTx()
	err = l.api.LocalCommit(l.txid)
	return err
}

//Close 提交一个事务
func (l *LocalDB) Close() error {
	l.cache = nil
	l.resetTx()
	err := l.api.LocalClose(l.txid)
	return err
}

//Rollback 回滚修改
func (l *LocalDB) Rollback() {
	l.resetTx()
	err := l.api.LocalRollback(l.txid)
	if err != nil {
		panic(err)
	}
}

//Get 获取key
func (l *LocalDB) Get(key []byte) ([]byte, error) {
	skey := string(key)
	if l.intx && l.txcache != nil {
		if value, ok := l.txcache[skey]; ok {
			return value, nil
		}
	}
	if value, ok := l.cache[skey]; ok {
		if value == nil {
			return nil, types.ErrNotFound
		}
		return value, nil
	}
	query := &types.LocalDBGet{Txid: l.txid.Data, Keys: [][]byte{key}}
	resp, err := l.api.LocalGet(query)
	if err != nil {
		panic(err) //no happen for ever
	}
	if nil == resp.Values {
		l.cache[string(key)] = nil
		return nil, types.ErrNotFound
	}
	value := resp.Values[0]
	if value == nil {
		l.cache[string(key)] = nil
		return nil, types.ErrNotFound
	}
	l.cache[string(key)] = value
	return value, nil
}

//Set 获取key
func (l *LocalDB) Set(key []byte, value []byte) error {
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
	l.kvs = append(l.kvs, &types.KeyValue{Key: key, Value: value})
	return nil
}

// List 从数据库中查询数据列表
func (l *LocalDB) List(prefix, key []byte, count, direction int32) ([][]byte, error) {
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
