package executor

import (
	"gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

type StateDB struct {
	cache     map[string][]byte
	txcache   map[string][]byte
	intx      bool
	client    queue.Client
	stateHash []byte
	version   int64
	local     *db.SimpleMVCC
	flagMVCC  int64
}

func NewStateDB(client queue.Client, stateHash []byte, enableMVCC bool, flagMVCC int64) db.KV {
	db := &StateDB{
		cache:     make(map[string][]byte),
		txcache:   make(map[string][]byte),
		intx:      false,
		client:    client,
		stateHash: stateHash,
		version:   -1,
		local:     db.NewSimpleMVCC(NewLocalDB(client)),
	}
	if enableMVCC {
		v, err := db.local.GetVersion(stateHash)
		if err == nil && v >= 0 {
			db.version = v
		}
		db.flagMVCC = flagMVCC
	}
	return db
}

func (s *StateDB) Begin() {
	s.intx = true
	s.txcache = nil
}

func (s *StateDB) Rollback() {
	s.resetTx()
}

func (s *StateDB) Commit() {
	for k, v := range s.txcache {
		s.cache[k] = v
	}
	s.resetTx()
}

func (s *StateDB) resetTx() {
	s.intx = false
	s.txcache = nil
}

func (s *StateDB) Get(key []byte) ([]byte, error) {
	skey := string(key)
	if s.intx && s.txcache != nil {
		if value, ok := s.txcache[skey]; ok {
			return value, nil
		}
	}
	if value, ok := s.cache[skey]; ok {
		return value, nil
	}
	//mvcc 是有效的情况下，直接从mvcc中获取
	if s.version >= 0 {
		data, err := s.local.GetV(key, s.version)
		//TODO 这里需要一个标志，数据是否是从0开始同步的
		if s.flagMVCC == FlagFromZero {
			return data, err
		} else if s.flagMVCC == FlagNotFromZero {
			if err == nil {
				return data, nil
			}
		}
	}
	if s.client == nil {
		return nil, types.ErrNotFound
	}
	query := &types.StoreGet{s.stateHash, [][]byte{key}}
	msg := s.client.NewMessage("store", types.EventStoreGet, query)
	s.client.Send(msg, true)
	resp, err := s.client.Wait(msg)
	if err != nil {
		panic(err) //no happen for ever
	}
	if nil == resp.GetData().(*types.StoreReplyValue).Values {
		return nil, types.ErrNotFound
	}
	value := resp.GetData().(*types.StoreReplyValue).Values[0]
	if value == nil {
		//panic(string(key))
		return nil, types.ErrNotFound
	}
	//get 的值可以写入cache，因为没有对系统的值做修改
	s.cache[skey] = value
	return value, nil
}

func (s *StateDB) Set(key []byte, value []byte) error {
	skey := string(key)
	if s.intx {
		if s.txcache == nil {
			s.txcache = make(map[string][]byte)
		}
		s.txcache[skey] = value
	} else {
		s.cache[skey] = value
	}
	return nil
}

func (db *StateDB) BatchGet(keys [][]byte) (value [][]byte, err error) {
	panic("not support")
}

type LocalDB struct {
	db.TransactionDB
	cache  map[string][]byte
	client queue.Client
}

func NewLocalDB(client queue.Client) db.KVDB {
	return &LocalDB{cache: make(map[string][]byte), client: client}
}

func (l *LocalDB) Get(key []byte) ([]byte, error) {
	if value, ok := l.cache[string(key)]; ok {
		return value, nil
	}
	query := &types.LocalDBGet{[][]byte{key}}
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

func (l *LocalDB) Set(key []byte, value []byte) error {
	l.cache[string(key)] = value
	return nil
}

func (db *LocalDB) BatchGet(keys [][]byte) (values [][]byte, err error) {
	panic("local batch get not support")
}

//从数据库中查询数据列表，set 中的cache 更新不会影响这个list
func (l *LocalDB) List(prefix, key []byte, count, direction int32) ([][]byte, error) {
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

//从数据库中查询指定前缀的key的数量
func (l *LocalDB) PrefixCount(prefix []byte) (count int64) {
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
