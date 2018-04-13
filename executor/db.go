package executor

import (
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

type StateDB struct {
	cache map[string][]byte
	db    *DataBase
}

func NewStateDB(client queue.Client, stateHash []byte) *StateDB {
	return &StateDB{make(map[string][]byte), NewDataBase(client, stateHash)}
}

func (e *StateDB) Get(key []byte) ([]byte, error) {
	if value, ok := e.cache[string(key)]; ok {
		//elog.Error("getkey", "key", string(key), "value", string(value))
		return value, nil
	}
	value, err := e.db.Get(key)
	if err != nil {
		//elog.Error("getkey", "key", string(key), "err", err)
		return nil, err
	}
	//elog.Error("getkey", "key", string(key), "value", string(value))
	e.cache[string(key)] = value
	return value, nil
}

func (e *StateDB) Set(key []byte, value []byte) error {
	//elog.Error("setkey", "key", string(key), "value", string(value))
	e.cache[string(key)] = value
	return nil
}

func (e *StateDB) List(prefix, key []byte, count, direction int32) (values [][]byte, err error) {
	return nil, types.ErrNotSupport
}

type LocalDB struct {
	cache map[string][]byte
	db    *DataBaseLocal
}

func NewLocalDB(client queue.Client) *LocalDB {
	return &LocalDB{make(map[string][]byte), NewDataBaseLocal(client)}
}

func (e *LocalDB) Get(key []byte) ([]byte, error) {
	if value, ok := e.cache[string(key)]; ok {
		//elog.Error("getkey", "key", string(key), "value", string(value))
		return value, nil
	}
	value, err := e.db.Get(key)
	if err != nil {
		//elog.Error("getkey", "key", string(key), "err", err)
		return nil, err
	}
	//elog.Error("getkey", "key", string(key), "value", string(value))
	e.cache[string(key)] = value
	return value, nil
}

func (e *LocalDB) Set(key []byte, value []byte) error {
	//elog.Error("setkey", "key", string(key), "value", string(value))
	e.cache[string(key)] = value
	return nil
}

func (e *LocalDB) List(prefix, key []byte, count, direction int32) (values [][]byte, err error) {
	return e.db.List(prefix, key, count, direction)
}

type DataBase struct {
	client    queue.Client
	stateHash []byte
}

func NewDataBase(client queue.Client, stateHash []byte) *DataBase {
	return &DataBase{client, stateHash}
}

func (db *DataBase) Get(key []byte) (value []byte, err error) {
	query := &types.StoreGet{db.stateHash, [][]byte{key}}
	msg := db.client.NewMessage("store", types.EventStoreGet, query)
	db.client.Send(msg, true)
	resp, err := db.client.Wait(msg)
	if err != nil {
		panic(err) //no happen for ever
	}
	value = resp.GetData().(*types.StoreReplyValue).Values[0]
	if value == nil {
		//panic(string(key))
		return nil, types.ErrNotFound
	}
	return value, nil
}

type DataBaseLocal struct {
	client queue.Client
}

func NewDataBaseLocal(client queue.Client) *DataBaseLocal {
	return &DataBaseLocal{client}
}

func (db *DataBaseLocal) Get(key []byte) (value []byte, err error) {
	query := &types.LocalDBGet{[][]byte{key}}
	msg := db.client.NewMessage("blockchain", types.EventLocalGet, query)
	db.client.Send(msg, true)
	resp, err := db.client.Wait(msg)
	if err != nil {
		panic(err) //no happen for ever
	}
	value = resp.GetData().(*types.LocalReplyValue).Values[0]
	if value == nil {
		//panic(string(key))
		return nil, types.ErrNotFound
	}
	return value, nil
}

//从数据库中查询数据列表，set 中的cache 更新不会影响这个list
func (db *DataBaseLocal) List(prefix, key []byte, count, direction int32) (values [][]byte, err error) {
	query := &types.LocalDBList{Prefix: prefix, Key: key, Count: count, Direction: direction}
	msg := db.client.NewMessage("blockchain", types.EventLocalList, query)
	db.client.Send(msg, true)
	resp, err := db.client.Wait(msg)
	if err != nil {
		panic(err) //no happen for ever
	}
	values = resp.GetData().(*types.LocalReplyValue).Values
	if values == nil {
		//panic(string(key))
		return nil, types.ErrNotFound
	}
	return values, nil
}
