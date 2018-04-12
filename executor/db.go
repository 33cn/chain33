package executor

import (
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

type StateDB struct {
	cache     map[string][]byte
	client    queue.Client
	stateHash []byte
}

func NewStateDB(client queue.Client, stateHash []byte) *StateDB {
	return &StateDB{make(map[string][]byte), client, stateHash}
}

func (s *StateDB) Get(key []byte) ([]byte, error) {
	if value, ok := s.cache[string(key)]; ok {
		return value, nil
	}
	query := &types.StoreGet{s.stateHash, [][]byte{key}}
	msg := s.client.NewMessage("store", types.EventStoreGet, query)
	s.client.Send(msg, true)
	resp, err := s.client.Wait(msg)
	if err != nil {
		panic(err) //no happen for ever
	}
	value := resp.GetData().(*types.StoreReplyValue).Values[0]
	if value == nil {
		//panic(string(key))
		return nil, types.ErrNotFound
	}
	s.cache[string(key)] = value
	return value, nil
}

func (s *StateDB) Set(key []byte, value []byte) error {
	s.cache[string(key)] = value
	return nil
}

type LocalDB struct {
	cache  map[string][]byte
	client queue.Client
}

func NewLocalDB(client queue.Client) *LocalDB {
	return &LocalDB{make(map[string][]byte), client}
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
