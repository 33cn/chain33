// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package db

import (
	"bytes"

	log "github.com/33cn/chain33/common/log/log15"
	"github.com/syndtr/goleveldb/leveldb/util"

	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/33cn/chain33/types"
)

var dlog = log.New("module", "db.ssdb")
var sdbBench = &SsdbBench{}

func init() {
	dbCreator := func(name string, dir string, cache int) (DB, error) {
		return NewGoSSDB(name, dir, cache)
	}
	registerDBCreator(ssDBBackendStr, dbCreator, false)
}

//SsdbBench ...
type SsdbBench struct {
	// 写次数
	writeCount int
	// 写条数
	writeNum int
	// 写耗费时间
	writeTime time.Duration
	readCount int
	readNum   int
	readTime  time.Duration
}

//SsdbNode 节点
type SsdbNode struct {
	ip   string
	port int
}

//GoSSDB db
type GoSSDB struct {
	BaseDB
	pool  *SDBPool
	nodes []*SsdbNode
}

func (bench *SsdbBench) write(num int, cost time.Duration) {
	bench.writeCount++
	bench.writeNum += num
	bench.writeTime += cost
}

func (bench *SsdbBench) read(num int, cost time.Duration) {
	bench.readCount++
	bench.readNum += num
	bench.readTime += cost
}

func (bench *SsdbBench) String() string {
	return fmt.Sprintf("SSDBBenchmark[(ReadTimes=%v, ReadRecordNum=%v, ReadCostTime=%v;) (WriteTimes=%v, WriteRecordNum=%v, WriteCostTime=%v)",
		bench.readCount, bench.readNum, bench.readTime, bench.writeCount, bench.writeNum, bench.writeTime)
}

func printSsdbBenchmark() {
	tick := time.Tick(time.Minute * 5)
	for {
		<-tick
		dlog.Info(sdbBench.String())
	}
}

// url pattern: ip:port,ip:port
func parseSsdbNode(url string) (nodes []*SsdbNode) {
	hosts := strings.Split(url, ",")
	if hosts == nil {
		dlog.Error("invalid ssdb url")
		return nil
	}
	for _, host := range hosts {
		parts := strings.Split(host, ":")
		if parts == nil || len(parts) != 2 {
			dlog.Error("invalid ssd url", "part", host)
			continue
		}
		ip := parts[0]
		port, err := strconv.Atoi(parts[1])
		if err != nil {
			dlog.Error("invalid ssd url host port", "port", parts[1])
			continue
		}
		nodes = append(nodes, &SsdbNode{ip, port})
	}
	return nodes
}

//NewGoSSDB new
func NewGoSSDB(name string, dir string, cache int) (*GoSSDB, error) {
	database := &GoSSDB{}
	database.nodes = parseSsdbNode(dir)

	if database.nodes == nil {
		dlog.Error("no valid ssdb instance exists, exit!")
		return nil, types.ErrDataBaseDamage
	}
	var err error
	database.pool, err = NewSDBPool(database.nodes)
	if err != nil {
		dlog.Error("connect to ssdb error!", "ssdb", database.nodes[0])
		return nil, types.ErrDataBaseDamage
	}

	go printSsdbBenchmark()

	return database, nil
}

//Get get
func (db *GoSSDB) Get(key []byte) ([]byte, error) {
	start := time.Now()

	value, err := db.pool.get().Get(string(key))
	if err != nil {
		//dlog.Error("Get value error", "error", err, "key", key, "keyhex", hex.EncodeToString(key), "keystr", string(key))
		return nil, err
	}
	if value == nil {
		return nil, ErrNotFoundInDb
	}

	sdbBench.read(1, time.Since(start))
	return value.Bytes(), nil
}

//Set 设置
func (db *GoSSDB) Set(key []byte, value []byte) error {
	start := time.Now()

	err := db.pool.get().Set(string(key), value)
	if err != nil {
		dlog.Error("Set", "error", err)
		return err
	}
	sdbBench.write(1, time.Since(start))
	return nil
}

//SetSync 同步
func (db *GoSSDB) SetSync(key []byte, value []byte) error {
	return db.Set(key, value)
}

//Delete 删除
func (db *GoSSDB) Delete(key []byte) error {
	start := time.Now()

	err := db.pool.get().Del(string(key))
	if err != nil {
		dlog.Error("Delete", "error", err)
		return err
	}
	sdbBench.write(1, time.Since(start))
	return nil
}

//DeleteSync 删除同步
func (db *GoSSDB) DeleteSync(key []byte) error {
	return db.Delete(key)
}

//Close 关闭
func (db *GoSSDB) Close() {
	db.pool.close()
}

//Print 打印
func (db *GoSSDB) Print() {
}

//Stats ...
func (db *GoSSDB) Stats() map[string]string {
	return make(map[string]string)
}

//Iterator 迭代器
func (db *GoSSDB) Iterator(itbeg []byte, itend []byte, reverse bool) Iterator {
	start := time.Now()

	var (
		keys  []string
		err   error
		begin string
		end   string
	)
	if itend == nil {
		itend = bytesPrefix(itbeg)
	}
	if bytes.Equal(itend, types.EmptyValue) {
		itend = nil
	}
	limit := util.Range{Start: itbeg, Limit: itend}
	if reverse {
		begin = string(limit.Limit)
		end = string(itbeg)
		keys, err = db.pool.get().Rkeys(begin, end, IteratorPageSize)
	} else {
		begin = string(itbeg)
		end = string(limit.Limit)
		keys, err = db.pool.get().Keys(begin, end, IteratorPageSize)
	}

	it := newSSDBIt(begin, end, itbeg, itend, []string{}, reverse, db)
	if err != nil {
		dlog.Error("get iterator error", "error", err, "keys", keys)
		return it
	}
	if len(keys) > 0 {
		it.keys = keys

		// 如果返回的数据大小刚好满足分页，则假设下一页还有数据
		if len(it.keys) == IteratorPageSize {
			it.nextPage = true
			it.tmpEnd = it.keys[IteratorPageSize-1]
		}
	}

	sdbBench.read(len(keys), time.Since(start))
	return it
}

// 因为ssdb不支持原生的迭代器模式，需要一次获取KEY；
// 为了防止匹配的KEY范围过大，这里需要进行分页，每次只取1024条KEY；
// Next方法自动进行跨页取数据
type ssDBIt struct {
	itBase
	db      *GoSSDB
	keys    []string
	index   int
	reverse bool
	// 迭代开始位置
	begin string
	// 迭代结束位置
	end string

	// 需要分页的情况下，下一页的开始位置
	//  是否有下一页数据
	tmpEnd   string
	nextPage bool

	// 当前所属的页数（从0开始）
	pageNo int
}

func newSSDBIt(begin, end string, prefix, itend []byte, keys []string, reverse bool, db *GoSSDB) *ssDBIt {
	return &ssDBIt{
		itBase:  itBase{prefix, itend, reverse},
		index:   -1,
		keys:    keys,
		reverse: reverse,
		db:      db,
	}
}

func (dbit *ssDBIt) Close() {
	dbit.keys = []string{}
}

// 获取下一页的数据
func (dbit *ssDBIt) cacheNextPage(begin string) bool {
	if dbit.initKeys(begin, dbit.end) {
		dbit.index = 0
		dbit.pageNo++
		return true
	}
	return false

}

func (dbit *ssDBIt) initKeys(begin, end string) bool {
	var (
		keys []string
		err  error
	)
	if dbit.reverse {
		keys, err = dbit.db.pool.get().Rkeys(begin, end, IteratorPageSize)
	} else {
		keys, err = dbit.db.pool.get().Keys(begin, end, IteratorPageSize)
	}
	if err != nil {
		dlog.Error("get iterator next page error", "error", err, "begin", begin, "end", dbit.end, "reverse", dbit.reverse)
		return false
	}

	if len(keys) > 0 {
		// 这里只改变keys，不改变index
		dbit.keys = keys

		// 如果返回的数据大小刚好满足分页，则假设下一页还有数据
		if len(keys) == IteratorPageSize {
			dbit.nextPage = true
			dbit.tmpEnd = dbit.keys[IteratorPageSize-1]
		} else {
			dbit.nextPage = false
		}
		return true
	}
	return false

}

func (dbit *ssDBIt) Next() bool {
	if len(dbit.keys) > dbit.index+1 {
		dbit.index++
		return true
	}
	// 如果有下一页数据，则自动抓取
	if dbit.nextPage {
		return dbit.cacheNextPage(dbit.tmpEnd)
	}
	return false

}

func (dbit *ssDBIt) checkKeyCmp(key1, key2 string, reverse bool) bool {
	if reverse {
		return strings.Compare(key1, key2) < 0
	}
	return strings.Compare(key1, key2) > 0
}

func (dbit *ssDBIt) findInPage(key string) int {
	pos := -1
	for i, v := range dbit.keys {
		if i < dbit.index {
			continue
		}
		if dbit.checkKeyCmp(key, v, dbit.reverse) {
			continue
		} else {
			pos = i
			break
		}
	}
	return pos
}

func (dbit *ssDBIt) Seek(key []byte) bool {
	keyStr := string(key)
	pos := dbit.findInPage(keyStr)

	// 如果第一页已经找到，不会走入此逻辑
	for pos == -1 && dbit.nextPage {
		if dbit.cacheNextPage(dbit.tmpEnd) {
			pos = dbit.findInPage(keyStr)
		} else {
			break
		}
	}

	dbit.index = pos
	return dbit.Valid()
}

func (dbit *ssDBIt) Rewind() bool {
	// 目前代码的Rewind调用都是在第一页，正常情况下走不到else分支；
	// 但为了代码健壮性考虑，这里增加对else分支的处理
	if dbit.pageNo == 0 {
		dbit.index = 0
		return true
	}

	// 当数据取到第N页的情况时，Rewind需要返回到第一页第一条
	if dbit.initKeys(dbit.begin, dbit.end) {
		dbit.index = 0
		dbit.pageNo = 0
		return true
	}
	return false

}

func (dbit *ssDBIt) Key() []byte {
	if dbit.index >= 0 && dbit.index < len(dbit.keys) {
		return []byte(dbit.keys[dbit.index])
	}
	return nil

}
func (dbit *ssDBIt) Value() []byte {
	key := dbit.keys[dbit.index]
	value, err := dbit.db.Get([]byte(key))

	if err != nil {
		dlog.Error("get iterator value error", "key", key, "error", err)
		return nil
	}
	return value
}

func (dbit *ssDBIt) Error() error {
	return nil
}

func (dbit *ssDBIt) ValueCopy() []byte {
	v := dbit.Value()
	value := make([]byte, len(v))
	copy(value, v)
	return value
}

func (dbit *ssDBIt) Valid() bool {
	start := time.Now()
	if dbit.index < 0 {
		return false
	}
	if len(dbit.keys) <= dbit.index {
		return false
	}
	key := dbit.keys[dbit.index]
	sdbBench.read(1, time.Since(start))
	return dbit.checkKey([]byte(key))
}

type ssDBBatch struct {
	db       *GoSSDB
	batchset map[string][]byte
	batchdel map[string]bool
	size     int
}

//NewBatch new
func (db *GoSSDB) NewBatch(sync bool) Batch {
	return &ssDBBatch{db: db, batchset: make(map[string][]byte), batchdel: make(map[string]bool)}
}

func (db *ssDBBatch) Set(key, value []byte) {
	db.batchset[string(key)] = value
	delete(db.batchdel, string(key))
	db.size += len(value)
	db.size += len(key)
}

func (db *ssDBBatch) Delete(key []byte) {
	db.batchset[string(key)] = []byte{}
	delete(db.batchset, string(key))
	db.batchdel[string(key)] = true
	db.size += len(key)
}

// 注意本方法的实现逻辑，因为ssdb没有提供删除和更新同时进行的批量操作；
// 所以这里先执行更新操作（删除的KEY在这里会将VALUE设置为空）；
// 然后再执行删除操作；
// 这样即使中间执行出错，也不会导致删除结果未写入的情况（值已经被置空）；
func (db *ssDBBatch) Write() error {
	start := time.Now()

	if len(db.batchset) > 0 {
		err := db.db.pool.get().MultiSet(db.batchset)
		if err != nil {
			dlog.Error("Write (multi_set)", "error", err)
			return err
		}
	}

	if len(db.batchdel) > 0 {
		var dkeys []string
		for k := range db.batchdel {
			dkeys = append(dkeys, k)
		}
		err := db.db.pool.get().MultiDel(dkeys...)
		if err != nil {
			dlog.Error("Write (multi_del)", "error", err)
			return err
		}
	}

	sdbBench.write(len(db.batchset)+len(db.batchdel), time.Since(start))
	return nil
}

func (db *ssDBBatch) ValueSize() int {
	return db.size
}

//ValueLen  batch数量
func (db *ssDBBatch) ValueLen() int {
	return len(db.batchset)
}

func (db *ssDBBatch) Reset() {
	db.batchset = make(map[string][]byte)
	db.batchdel = make(map[string]bool)
	db.size = 0
}
