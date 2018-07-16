package db

import (
	"bytes"
	"fmt"
	"strconv"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/types"
)

var mvccPrefix = []byte(".-mvcc-.")
var mvccMeta = append(mvccPrefix, []byte("m.")...)
var mvccData = append(mvccPrefix, []byte("d.")...)

//MVCC mvcc interface
type MVCC interface {
	MVCCKV
	SetVersion(hash []byte, version int64) error
	DelVersion(hash []byte) error
	GetVersion(hash []byte) (int64, error)
	GetV(key []byte, version int64) ([]byte, error)
	SetV(key []byte, value []byte, version int64) error
	DelV(key []byte, version int64) error
	//回收: 某个版本之前的所有数据
	//1. 保证有一个最新版本
	//2. 这个操作回遍历所有的key所以比较慢
	Trash(version int64) error
}

//MVCCKV only return kv when change database
type MVCCKV interface {
	GetSaveKV(key []byte, value []byte, version int64) (*types.KeyValue, error)
	GetDelKV(key []byte, version int64) (*types.KeyValue, error)
	SetVersionKV(hash []byte, version int64) (*types.KeyValue, error)
	DelVersionKV(hash []byte) *types.KeyValue
}

//MVCCHelper impl MVCC interface
type MVCCHelper struct {
	*SimpleMVCC
	db DB
}

type SimpleMVCC struct {
	kvdb KVDB
}

var mvcclog = log.New("module", "db.mvcc")

//NewMVCC create MVCC object use db DB
func NewMVCC(db DB) *MVCCHelper {
	return &MVCCHelper{SimpleMVCC: NewSimpleMVCC(NewKVDB(db)), db: db}
}

//Trash del some old kv
func (m *MVCCHelper) Trash(version int64) error {
	it := m.db.Iterator(mvccData, true)
	defer it.Close()
	perfixkey := []byte("--.xxx.--")
	for it.Rewind(); it.Valid(); it.Next() {
		if it.Error() != nil {
			mvcclog.Error("Trash", "error", it.Error())
			return it.Error()
		}
		//如果进入一个新的key, 这个key 忽略，不删除，也就是至少保留一个
		if !bytes.HasPrefix(it.Key(), perfixkey) {
			perfixkey = cutVersion(it.Key())
			if perfixkey == nil {
				perfixkey = []byte("--.xxx.--")
			}
			continue
		}
		//第二个key
		v, err := getVersion(it.Key())
		if err != nil {
			mvcclog.Error("Trash get verson", "err", err)
			continue
		}
		if v <= version {
			m.db.Delete(it.Key())
		}
	}
	return nil
}

//DelVersion del stateHash version map
func (m *MVCCHelper) DelVersion(hash []byte) error {
	kv := m.DelVersionKV(hash)
	return m.db.Delete(kv.Key)
}

//DelV del key with version
func (m *MVCCHelper) DelV(key []byte, version int64) error {
	kv, err := m.GetDelKV(key, version)
	if err != nil {
		return err
	}
	return m.db.Delete(kv.Key)
}

func NewSimpleMVCC(db KVDB) *SimpleMVCC {
	return &SimpleMVCC{db}
}

//GetVersion get stateHash and version map
func (m *SimpleMVCC) GetVersion(hash []byte) (int64, error) {
	key := append(mvccMeta, hash...)
	value, err := m.kvdb.Get(key)
	if err != nil {
		if err == ErrNotFoundInDb {
			return 0, types.ErrNotFound
		}
		return 0, err
	}
	var data types.Int64
	err = types.Decode(value, &data)
	if err != nil {
		return 0, err
	}
	if data.GetData() < 0 {
		return 0, types.ErrVersion
	}
	return data.GetData(), nil
}

//GetSaveKV only export set key and value with version
func (m *SimpleMVCC) GetSaveKV(key []byte, value []byte, version int64) (*types.KeyValue, error) {
	k, err := getKey(key, version)
	if err != nil {
		return nil, err
	}
	return &types.KeyValue{Key: k, Value: value}, nil
}

//GetDelKV only export del key and value with version
func (m *SimpleMVCC) GetDelKV(key []byte, version int64) (*types.KeyValue, error) {
	k, err := getKey(key, version)
	if err != nil {
		return nil, err
	}
	return &types.KeyValue{Key: k}, nil
}

//SetVersionKV only export SetVersionKV key and value
func (m *SimpleMVCC) SetVersionKV(hash []byte, version int64) (*types.KeyValue, error) {
	if version < 0 {
		return nil, types.ErrVersion
	}
	key := append(mvccMeta, hash...)
	data := &types.Int64{Data: version}
	return &types.KeyValue{Key: key, Value: types.Encode(data)}, nil
}

//DelVersionKV only export del version key value
func (m *SimpleMVCC) DelVersionKV(hash []byte) *types.KeyValue {
	key := append(mvccMeta, hash...)
	return &types.KeyValue{Key: key}
}

//SetVersion set stateHash -> version map
func (m *SimpleMVCC) SetVersion(hash []byte, version int64) error {
	kv, err := m.SetVersionKV(hash, version)
	if err != nil {
		return err
	}
	return m.kvdb.Set(kv.Key, kv.Value)
}

//SetV set key and value with version
func (m *SimpleMVCC) SetV(key []byte, value []byte, version int64) error {
	kv, err := m.GetSaveKV(key, value, version)
	if err != nil {
		return err
	}
	return m.kvdb.Set(kv.Key, kv.Value)
}

//GetV get key with version
func (m *SimpleMVCC) GetV(key []byte, version int64) ([]byte, error) {
	prefix := getKeyPerfix(key)
	search, err := getKey(key, version)
	if err != nil {
		return nil, err
	}
	vals, err := m.kvdb.List(prefix, search, 1, ListSeek)
	if err != nil {
		return nil, err
	}
	k := vals[0]
	val := vals[1]
	v, err := getVersion(k)
	if err != nil {
		return nil, err
	}
	if v > version {
		return nil, types.ErrVersion
	}
	return val, nil
}

func getVersionString(key []byte) (string, error) {
	for i := len(key) - 1; i >= 0; i-- {
		if key[i] == '.' {
			return string(key[i+1:]), nil
		}
	}
	return "", types.ErrVersion
}

func cutVersion(key []byte) []byte {
	for i := len(key) - 1; i >= 0; i-- {
		if key[i] == '.' {
			d := make([]byte, i)
			copy(d, key[0:i+1])
			return d
		}
	}
	return nil
}

func getVersion(key []byte) (int64, error) {
	s, err := getVersionString(key)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(s, 10, 64)
}

func pad(version int64) []byte {
	s := fmt.Sprintf("%020d", version)
	return []byte(s)
}

func getKeyPerfix(key []byte) []byte {
	newkey := append(mvccData, key...)
	newkey = append(newkey, []byte(".")...)
	return newkey
}

func getKey(key []byte, version int64) ([]byte, error) {
	newkey := append(getKeyPerfix(key), pad(version)...)
	return newkey, nil
}
