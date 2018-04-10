package db

import (
	"sync"

	"sort"
	"strings"

	log "github.com/inconshreveable/log15"
)

var mlog = log.New("module", "db.memdb")

// memdb 应该无需区分同步与异步操作

func init() {
	dbCreator := func(name string, dir string, cache int) (DB, error) {
		return NewGoMemDB(name, dir, cache)
	}
	registerDBCreator(MemDBBackendStr, dbCreator, false)
}

type GoMemDB struct {
	db   map[string][]byte
	lock sync.RWMutex
}

func NewGoMemDB(name string, dir string, cache int) (*GoMemDB, error) {

	// memdb 不需要创建文件，后续考虑增加缓存数目
	return &GoMemDB{
		db: make(map[string][]byte),
	}, nil
}

func CopyBytes(b []byte) (copiedBytes []byte) {
	if b == nil {
		return nil
	}
	copiedBytes = make([]byte, len(b))
	copy(copiedBytes, b)

	return copiedBytes
}

func (db *GoMemDB) Get(key []byte) []byte {
	db.lock.RLock()
	defer db.lock.RUnlock()

	if entry, ok := db.db[string(key)]; ok {
		return CopyBytes(entry)
	}
	return nil
}

func (db *GoMemDB) Set(key []byte, value []byte) {
	db.lock.Lock()
	defer db.lock.Unlock()

	db.db[string(key)] = CopyBytes(value)
	if db.db[string(key)] == nil {
		mlog.Error("Set", "error have no mem")
	}
}

func (db *GoMemDB) SetSync(key []byte, value []byte) {
	db.lock.Lock()
	defer db.lock.Unlock()

	db.db[string(key)] = CopyBytes(value)
	if db.db[string(key)] == nil {
		mlog.Error("Set", "error have no mem")
	}
}

func (db *GoMemDB) Delete(key []byte) {
	db.lock.Lock()
	defer db.lock.Unlock()

	delete(db.db, string(key))
}

func (db *GoMemDB) DeleteSync(key []byte) {
	db.lock.Lock()
	defer db.lock.Unlock()

	delete(db.db, string(key))
}

func (db *GoMemDB) DB() map[string][]byte {
	return db.db
}

func (db *GoMemDB) Close() {

}

func (db *GoMemDB) Print() {
	for key, value := range db.db {
		mlog.Info("Print", "key", string(key), "value", string(value))
	}
}

func (db *GoMemDB) Stats() map[string]string {
	//TODO
	return nil
}

func (db *GoMemDB) Iterator(prefix []byte, reserve bool) Iterator {

	var keys []string
	for k, _ := range db.db {
		if strings.HasPrefix(k, string(prefix)) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	var index int = 0
	return &goMemDBIt{index, keys, db, reserve, prefix}
}

type goMemDBIt struct {
	index   int      // 记录当前索引
	keys    []string // 记录所有keys值
	goMemDb *GoMemDB
	reserve bool
	prefix  []byte
}

func (dbit *goMemDBIt) Seek(key []byte) bool { //指向当前的index值

	for i, k := range dbit.keys {
		if 0 == strings.Compare(k, string(key)) {
			dbit.index = i
			return true
		}
	}
	return false
}

func (dbit *goMemDBIt) Close() {
	dbit.goMemDb.Close()
}

func (dbit *goMemDBIt) Next() bool {

	if dbit.reserve { // 反向
		dbit.index -= 1 //将当前key值指向前一个
		return true
	} else { // 正向
		dbit.index += 1 //将当前key值指向后一个
		return true
	}
}

func (dbit *goMemDBIt) Rewind() bool {

	if dbit.reserve { // 反向
		if (len(dbit.keys) > 0) && dbit.Valid() {
			dbit.index = len(dbit.keys) - 1 // 将当前key值指向最后一个
			return true
		} else {
			return false
		}
	} else { // 正向
		if dbit.Valid() {
			dbit.index = 0 // 将当前key值指向第一个
			return true
		} else {
			return false
		}
	}
}

func (dbit *goMemDBIt) Key() []byte {
	return []byte(dbit.keys[dbit.index])
}

func (dbit *goMemDBIt) Value() []byte {
	return dbit.goMemDb.Get([]byte(dbit.keys[dbit.index]))
}

func (dbit *goMemDBIt) ValueCopy() []byte {
	v := dbit.goMemDb.Get([]byte(dbit.keys[dbit.index]))
	value := make([]byte, len(v))
	copy(value, v)
	return value
}

func (dbit *goMemDBIt) Valid() bool {

	if (dbit.goMemDb == nil) && (len(dbit.keys) == 0) {
		return false
	}

	if len(dbit.keys) > dbit.index && dbit.index >= 0 {
		return true
	} else {
		return false
	}
}

func (dbit *goMemDBIt) Error() error {
	return nil
}

type kv struct{ k, v []byte }
type memBatch struct {
	db     *GoMemDB
	writes []kv
	size   int
}

func (db *GoMemDB) NewBatch(sync bool) Batch {
	return &memBatch{db: db}
}

func (b *memBatch) Set(key, value []byte) {
	b.writes = append(b.writes, kv{CopyBytes(key), CopyBytes(value)})
}

func (b *memBatch) Delete(key []byte) {
	b.writes = append(b.writes, kv{CopyBytes(key), CopyBytes(nil)})
}

func (b *memBatch) Write() {
	b.db.lock.Lock()
	defer b.db.lock.Unlock()

	for _, kv := range b.writes {
		if kv.v == nil {
			b.db.Delete(kv.k)
		} else {
			b.db.Set(kv.k, kv.v)
		}
	}
}
