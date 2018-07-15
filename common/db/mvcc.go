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

type MVCC interface {
	SetVersion(hash []byte, version int64) error
	DelVersion(hash []byte) error
	GetVersion(hash []byte) (int64, error)
	GetV(key []byte, version int64) ([]byte, error)
	SetV(key []byte, value []byte, version int64) error
	DelV(key []byte, version int64) error
	GetSaveKV(key []byte, value []byte, version int64) *types.KeyValue
	GetDelKV(key []byte, version int64) *types.KeyValue
	//回收: 某个版本之前的所有数据
	//1. 保证有一个最新版本
	//2. 这个操作回遍历所有的key所以比较慢
	Trash(version int64) error
}

type MVCCHelper struct {
	db   DB
	list *ListHelper
}

var mvcclog = log.New("module", "db.mvcc")

func NewMVCC(db DB) *MVCCHelper {
	return &MVCCHelper{db, NewListHelper(db)}
}

func pad(version int64) []byte {
	s := fmt.Sprintf("%020d", version)
	return []byte(s)
}

func (m *MVCCHelper) SetVersion(hash []byte, version int64) error {
	if version < 0 {
		return types.ErrVersion
	}
	key := append(mvccMeta, hash...)
	data := &types.Int64{version}
	return m.db.Set(key, types.Encode(data))
}

func (m *MVCCHelper) DelVersion(hash []byte) error {
	key := append(mvccMeta, hash...)
	return m.db.Delete(key)
}

func (m *MVCCHelper) GetVersion(hash []byte) (int64, error) {
	key := append(mvccMeta, hash...)
	value, err := m.db.Get(key)
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
	return data.GetData(), nil
}

func (m *MVCCHelper) Trash(version int64) error {
	it := m.db.Iterator(mvccData, true)
	defer it.Close()
	for it.Rewind(); it.Valid(); it.Next() {
		if it.Error() != nil {
			mvcclog.Error("Trash", "error", it.Error())
			return it.Error()
		}
	}
	return nil
}

func (m *MVCCHelper) GetV(key []byte, version int64) ([]byte, error) {
	prefix := getKeyPerfix(key, version)
	it := m.db.Iterator(prefix, true)
	defer it.Close()
	search := getKey(key, version)
	it.Seek(search)
	//判断是否相等
	if !bytes.Equal(search, it.Key()) {
		it.Next()
		if !it.Valid() {
			return nil, types.ErrNotFound
		}
	}
	v, err := getVersion(it.Key())
	if err != nil {
		return nil, err
	}
	if v > version {
		return nil, types.ErrVersion
	}
	return it.ValueCopy(), nil
}

func getVersionString(key []byte) (string, error) {
	for i := len(key) - 1; i >= 0; i-- {
		if key[i] == '.' {
			return string(key[i+1:]), nil
		}
	}
	return "", types.ErrVersion
}

func getVersion(key []byte) (int64, error) {
	s, err := getVersionString(key)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(s, 10, 64)
}

//设置某个版本的信息
func (m *MVCCHelper) DelV(key []byte, version int64) error {
	kv := m.GetDelKV(key, version)
	return m.db.Delete(kv.Key)
}

func (m *MVCCHelper) SetV(key []byte, value []byte, version int64) error {
	kv := m.GetSaveKV(key, value, version)
	return m.db.Set(kv.Key, kv.Value)
}

func getKeyPerfix(key []byte, version int64) []byte {
	newkey := append(mvccData, key...)
	newkey = append(newkey, []byte(".")...)
	return newkey
}

func getKey(key []byte, version int64) []byte {
	newkey := append(getKeyPerfix(key, version), pad(version)...)
	return newkey
}

func (m *MVCCHelper) GetSaveKV(key []byte, value []byte, version int64) *types.KeyValue {
	return &types.KeyValue{Key: getKey(key, version), Value: value}
}

func (m *MVCCHelper) GetDelKV(key []byte, version int64) *types.KeyValue {
	return &types.KeyValue{Key: getKey(key, version), Value: nil}
}
