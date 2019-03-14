// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package db

import (
	"bytes"
	"fmt"
	"strconv"

	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/types"
)

var mvccPrefix = []byte(".-mvcc-.")
var mvccMeta = append(mvccPrefix, []byte("m.")...)
var mvccData = append(mvccPrefix, []byte("d.")...)
var mvccLast = append(mvccPrefix, []byte("l.")...)
var mvccMetaVersion = append(mvccMeta, []byte("version.")...)
var mvccMetaVersionKeyList = append(mvccMeta, []byte("versionkl.")...)

//MVCC mvcc interface
type MVCC interface {
	MVCCKV
	SetVersion(hash []byte, version int64) error
	DelVersion(hash []byte) error
	GetVersion(hash []byte) (int64, error)
	GetV(key []byte, version int64) ([]byte, error)
	SetV(key []byte, value []byte, version int64) error
	DelV(key []byte, version int64) error
	AddMVCC(kvs []*types.KeyValue, hash []byte, prevHash []byte, version int64) ([]*types.KeyValue, error)
	DelMVCC(hash []byte, version int64, strict bool) ([]*types.KeyValue, error)
	GetMaxVersion() (int64, error)
	GetVersionHash(version int64) ([]byte, error)
	//回收: 某个版本之前的所有数据
	//1. 保证有一个最新版本
	//2. 这个操作回遍历所有的key所以比较慢
	Trash(version int64) error
}

//MVCCKV only return kv when change database
type MVCCKV interface {
	GetSaveKV(key []byte, value []byte, version int64) (*types.KeyValue, error)
	GetDelKV(key []byte, version int64) (*types.KeyValue, error)
	SetVersionKV(hash []byte, version int64) ([]*types.KeyValue, error)
	DelVersionKV([]byte, int64) ([]*types.KeyValue, error)
}

//MVCCHelper impl MVCC interface
type MVCCHelper struct {
	*SimpleMVCC
	db DB
}

//SimpleMVCC kvdb
type SimpleMVCC struct {
	kvdb KVDB
}

var mvcclog = log.New("module", "db.mvcc")

//NewMVCC create MVCC object use db DB
func NewMVCC(db DB) *MVCCHelper {
	return &MVCCHelper{SimpleMVCC: NewSimpleMVCC(NewKVDB(db)), db: db}
}

//PrintAll 打印全部
func (m *MVCCHelper) PrintAll() {
	println("--meta--")
	it := m.db.Iterator(mvccMeta, nil, true)
	defer it.Close()
	for it.Rewind(); it.Valid(); it.Next() {
		if it.Error() != nil {
			mvcclog.Error("PrintAll", "error", it.Error())
			return
		}
		println(string(it.Key()), string(it.Value()))
	}

	println("--data--")
	it = m.db.Iterator(mvccData, nil, true)
	defer it.Close()
	for it.Rewind(); it.Valid(); it.Next() {
		if it.Error() != nil {
			mvcclog.Error("PrintAll", "error", it.Error())
			return
		}
		println(string(it.Key()), string(it.Value()))
	}
	println("--last--")
	it = m.db.Iterator(mvccLast, nil, true)
	defer it.Close()
	for it.Rewind(); it.Valid(); it.Next() {
		if it.Error() != nil {
			mvcclog.Error("PrintAll", "error", it.Error())
			return
		}
		println(string(it.Key()), string(it.Value()))
	}
}

//Trash del some old kv
func (m *MVCCHelper) Trash(version int64) error {
	it := m.db.Iterator(mvccData, nil, true)
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
			err := m.db.Delete(it.Key())
			if err != nil {
				mvcclog.Error("Trash Delete verson", "err", err)
			}
		}
	}
	return nil
}

//DelVersion del stateHash version map
func (m *MVCCHelper) DelVersion(hash []byte) error {
	version, err := m.GetVersion(hash)
	if err != nil {
		return err
	}
	kvs, err := m.DelVersionKV(hash, version)
	if err != nil {
		return err
	}
	batch := m.db.NewBatch(true)
	for _, v := range kvs {
		batch.Delete(v.Key)
	}
	return batch.Write()
}

//SetVersion set stateHash -> version map
func (m *MVCCHelper) SetVersion(hash []byte, version int64) error {
	kvlist, err := m.SetVersionKV(hash, version)
	if err != nil {
		return err
	}
	batch := m.db.NewBatch(true)
	for _, v := range kvlist {
		batch.Set(v.Key, v.Value)
	}
	return batch.Write()
}

//DelV del key with version
func (m *MVCCHelper) DelV(key []byte, version int64) error {
	kv, err := m.GetDelKV(key, version)
	if err != nil {
		return err
	}
	return m.db.Delete(kv.Key)
}

//NewSimpleMVCC new
func NewSimpleMVCC(db KVDB) *SimpleMVCC {
	return &SimpleMVCC{db}
}

//GetVersion get stateHash and version map
func (m *SimpleMVCC) GetVersion(hash []byte) (int64, error) {
	key := getVersionHashKey(hash)
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

//GetVersionHash 获取版本hash
func (m *SimpleMVCC) GetVersionHash(version int64) ([]byte, error) {
	key := getVersionKey(version)
	value, err := m.kvdb.Get(key)
	if err != nil {
		if err == ErrNotFoundInDb {
			return nil, types.ErrNotFound
		}
		return nil, err
	}
	return value, nil
}

//GetMaxVersion 获取最高版本
func (m *SimpleMVCC) GetMaxVersion() (int64, error) {
	vals, err := m.kvdb.List(mvccMetaVersion, nil, 1, ListDESC)
	if err != nil {
		return 0, err
	}
	hash := vals[0]
	return m.GetVersion(hash)
}

//GetSaveKV only export set key and value with version
func (m *SimpleMVCC) GetSaveKV(key []byte, value []byte, version int64) (*types.KeyValue, error) {
	k, err := GetKey(key, version)
	if err != nil {
		return nil, err
	}
	return &types.KeyValue{Key: k, Value: value}, nil
}

//GetDelKV only export del key and value with version
func (m *SimpleMVCC) GetDelKV(key []byte, version int64) (*types.KeyValue, error) {
	k, err := GetKey(key, version)
	if err != nil {
		return nil, err
	}
	return &types.KeyValue{Key: k}, nil
}

//GetDelKVList 获取列表
func (m *SimpleMVCC) GetDelKVList(version int64) ([]*types.KeyValue, error) {
	k := getVersionKeyListKey(version)
	data, err := m.kvdb.Get(k)
	if err != nil {
		return nil, err
	}
	var kvlist types.LocalDBSet
	err = types.Decode(data, &kvlist)
	if err != nil {
		return nil, err
	}
	return kvlist.KV, nil
}

//SetVersionKV only export SetVersionKV key and value
func (m *SimpleMVCC) SetVersionKV(hash []byte, version int64) ([]*types.KeyValue, error) {
	if version < 0 {
		return nil, types.ErrVersion
	}
	key := append(mvccMeta, hash...)
	data := &types.Int64{Data: version}
	v1 := &types.KeyValue{Key: key, Value: types.Encode(data)}

	k2 := append(mvccMetaVersion, pad(version)...)
	v2 := &types.KeyValue{Key: k2, Value: hash}
	return []*types.KeyValue{v1, v2}, nil
}

//DelVersionKV only export del version key value
func (m *SimpleMVCC) DelVersionKV(hash []byte, version int64) ([]*types.KeyValue, error) {
	kvs, err := m.SetVersionKV(hash, version)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(kvs); i++ {
		kvs[i].Value = nil
	}
	return kvs, nil
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
	prefix := GetKeyPerfix(key)
	search, err := GetKey(key, version)
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

//AddMVCC add keys in a version
//添加MVCC的规则:
//必须提供现有hash 和 prev 的hash, 而且这两个版本号相差1
func (m *SimpleMVCC) AddMVCC(kvs []*types.KeyValue, hash []byte, prevHash []byte, version int64) ([]*types.KeyValue, error) {
	if version > 0 {
		if prevHash == nil {
			return nil, types.ErrPrevVersion
		}
		prevVersion := version - 1
		v, err := m.GetVersion(prevHash)
		if err != nil {
			return nil, err
		}
		if v != prevVersion {
			return nil, types.ErrPrevVersion
		}
	}
	versionlist, err := m.SetVersionKV(hash, version)
	if err != nil {
		return nil, err
	}
	var kvlist []*types.KeyValue
	kvlist = append(kvlist, versionlist...)
	delkeys := &types.LocalDBSet{}
	for i := 0; i < len(kvs); i++ {
		//最原始的数据
		delkeys.KV = append(delkeys.KV, &types.KeyValue{Key: kvs[i].Key})
		kv, err := m.GetSaveKV(kvs[i].Key, kvs[i].Value, version)
		if err != nil {
			return nil, err
		}
		kvlist = append(kvlist, kv)
	}
	kvlist = append(kvlist, &types.KeyValue{Key: getVersionKeyListKey(version), Value: types.Encode(delkeys)})
	return kvlist, nil
}

//DelMVCC 删除某个版本
//我们目前规定删除的方法:
//1 -> 1-2-3-4-5,6 version 必须连续的增长
//2 -> del 也必须从尾部开始删除
func (m *SimpleMVCC) DelMVCC(hash []byte, version int64, strict bool) ([]*types.KeyValue, error) {
	kvs, err := m.GetDelKVList(version)
	if err != nil {
		return nil, err
	}
	return m.delMVCC(kvs, hash, version, strict)
}

func (m *SimpleMVCC) delMVCC(kvs []*types.KeyValue, hash []byte, version int64, strict bool) ([]*types.KeyValue, error) {
	if strict {
		maxv, err := m.GetMaxVersion()
		if err != nil {
			return nil, err
		}
		if maxv != version {
			return nil, types.ErrCanOnlyDelTopVersion
		}
	}
	//check hash and version is match
	vdb, err := m.GetVersion(hash)
	if err != nil {
		return nil, err
	}
	if vdb != version {
		return nil, types.ErrVersion
	}
	kv, err := m.DelVersionKV(hash, version)
	if err != nil {
		return nil, err
	}
	var kvlist []*types.KeyValue
	kvlist = append(kvlist, kv...)
	for _, v := range kvs {
		kv, err := m.GetDelKV(v.Key, version)
		if err != nil {
			return nil, err
		}
		kvlist = append(kvlist, kv)
	}
	return kvlist, nil
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

//GetKeyPerfix 获取key前缀
func GetKeyPerfix(key []byte) []byte {
	b := append([]byte{}, mvccData...)
	newkey := append(b, key...)
	newkey = append(newkey, []byte(".")...)
	return newkey
}

//GetKey 获取键
func GetKey(key []byte, version int64) ([]byte, error) {
	newkey := append(GetKeyPerfix(key), pad(version)...)
	return newkey, nil
}

func getLastKey(key []byte) []byte {
	b := append([]byte{}, mvccLast...)
	return append(b, key...)
}

func getVersionHashKey(hash []byte) []byte {
	b := append([]byte{}, mvccMeta...)
	key := append(b, hash...)
	return key
}

func getVersionKey(version int64) []byte {
	b := append([]byte{}, mvccMetaVersion...)
	key := append(b, pad(version)...)
	return key
}

func getVersionKeyListKey(version int64) []byte {
	b := append([]byte{}, mvccMetaVersionKeyList...)
	key := append(b, pad(version)...)
	return key
}
