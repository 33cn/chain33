package db

import (
	"bytes"
	log "github.com/inconshreveable/log15"
	"github.com/syndtr/goleveldb/leveldb/util"

	"encoding/hex"
	"fmt"
	"github.com/pkg/errors"
	"github.com/seefan/gossdb"
	"github.com/seefan/gossdb/ssdb"
	"gitlab.33.cn/chain33/chain33/types"
	"strconv"
	"strings"
	"time"
)

var dlog = log.New("module", "db.ssdb")
var SsdbPoolError = errors.New("get client from pool error")
var benchmark = &SsdbBench{}

func init() {
	dbCreator := func(name string, dir string, cache int) (DB, error) {
		return NewGoSSDB(name, dir, cache)
	}
	registerDBCreator(ssDBBackendStr, dbCreator, false)

	go printSsdbBenchmark()
}

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
type SsdbNode struct {
	ip   string
	port int
}
type GoSSDB struct {
	TransactionDB
	//pool  *gossdb.Connectors
	pool  *gossdb.Client
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
	for {
		time.Sleep(time.Minute)
		dlog.Info(benchmark.String())
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

func NewGoSSDB(name string, dir string, cache int) (*GoSSDB, error) {
	database := &GoSSDB{}
	database.nodes = parseSsdbNode(dir)
	if database.nodes == nil {
		dlog.Error("no valid ssdb instance exists, exit!")
		return nil, types.ErrDataBaseDamage
	}

	var err error
	//node := database.nodes[0]
	//pool, err := gossdb.NewPool(&conf.Config{
	//	Host:         node.ip,
	//	Port:         node.port,
	//	RetryEnabled: true,
	//})
	//if err != nil {
	//	dlog.Error("create ssdb pool error, exit!", "error", err)
	//	return nil, types.ErrDataBaseDamage
	//}
	if err = ssdb.Start(); err != nil {
		dlog.Error("start ssdb error, exit!", "error", err)
		return nil, types.ErrDataBaseDamage
	}

	database.pool, err = ssdb.Client()
	if err != nil {
		dlog.Error("get ssdb client error, exit!", "error", err)
		return nil, types.ErrDataBaseDamage
	}
	return database, nil
}

func (db *GoSSDB) newClient() *gossdb.Client {
	//client, err := db.pool.NewClient()
	//if err != nil {
	//	dlog.Error("can not get connection from pool", "error", err)
	//	return nil
	//}
	//return client

	return db.pool
}

func (db *GoSSDB) Get(key []byte) ([]byte, error) {
	start := time.Now()
	client := db.newClient()
	if client == nil {
		return nil, SsdbPoolError
	}
	defer client.Close()

	value, err := client.Get(string(key))
	if err != nil {
		dlog.Error("Get value error", "error", err, "key", key, "keyhex", hex.EncodeToString(key), "keystr", string(key))
		return nil, err
	}
	if value == "" {
		return nil, ErrNotFoundInDb
	}

	// 空值特殊处理
	binVal := value.Bytes()
	if bytes.Equal(binVal, types.EmptyValue) {
		return []byte{}, nil
	}

	benchmark.read(1, time.Since(start))
	return binVal, nil
}

func (db *GoSSDB) Set(key []byte, value []byte) error {
	start := time.Now()
	if len(value) == 0 {
		value = types.EmptyValue
	}

	client := db.newClient()
	if client == nil {
		return SsdbPoolError
	}
	defer client.Close()

	err := client.Set(string(key), value)
	if err != nil {
		llog.Error("Set", "error", err)
		return err
	}
	benchmark.write(1, time.Since(start))
	return nil
}

func (db *GoSSDB) SetSync(key []byte, value []byte) error {
	return db.Set(key, value)
}

func (db *GoSSDB) Delete(key []byte) error {
	start := time.Now()
	client := db.newClient()
	if client == nil {
		return SsdbPoolError
	}
	defer client.Close()
	err := client.Del(string(key))
	if err != nil {
		llog.Error("Delete", "error", err)
		return err
	}
	benchmark.write(1, time.Since(start))
	return nil
}

func (db *GoSSDB) DeleteSync(key []byte) error {
	return db.Delete(key)
}

func (db *GoSSDB) Close() {
	db.pool.Close()
}

func (db *GoSSDB) Print() {
	//dlog.Info(db.pool.Info())
}

func (db *GoSSDB) Stats() map[string]string {
	return make(map[string]string)
}

func (db *GoSSDB) Iterator(prefix []byte, reverse bool) Iterator {
	start := time.Now()
	client := db.newClient()
	if client == nil {
		return newSSDBIt(prefix, []string{}, reverse, db)
	}
	defer client.Close()

	var (
		keys []string
		err  error
	)

	limit := util.BytesPrefix(prefix)
	if reverse {
		keys, err = client.Rkeys(string(limit.Limit), string(prefix), -1)
	} else {
		keys, err = client.Rkeys(string(prefix), string(limit.Limit), -1)
	}

	it := newSSDBIt(prefix, []string{}, reverse, db)
	if err != nil || len(keys) == 0 {
		dlog.Error("get iterator error", "error", err, "keys", keys)
		return it
	}
	it.keys = keys

	benchmark.read(len(keys), time.Since(start))
	return it
}

type ssDBIt struct {
	db      *GoSSDB
	keys    []string
	index   int
	reverse bool
	prefix  []byte
}

func newSSDBIt(prefix []byte, keys []string, reverse bool, db *GoSSDB) *ssDBIt {
	return &ssDBIt{index: -1, keys: keys, reverse: reverse, prefix: prefix, db: db}
}

func (dbit *ssDBIt) Close() {
	dbit.keys = []string{}
}

func (dbit *ssDBIt) Next() bool {
	if len(dbit.keys) > dbit.index+1 {
		dbit.index++
		return true
	} else {
		return false
	}
}

func (dbit *ssDBIt) Seek(key []byte) bool {
	keyStr := string(key)
	pos := 0
	for i, v := range dbit.keys {
		if i < dbit.index {
			continue
		}
		if strings.Compare(keyStr, v) < 0 {
			continue
		} else {
			pos = i
			break
		}
	}

	tmp := dbit.index
	dbit.index = pos
	if dbit.Valid() {
		return true
	} else {
		dbit.index = tmp
		return false
	}
}

func (dbit *ssDBIt) Rewind() bool {
	dbit.index = 0
	return true
}

func (dbit *ssDBIt) Key() []byte {
	if dbit.index >= 0 && dbit.index < len(dbit.keys) {
		return []byte(dbit.keys[dbit.index])
	} else {
		return nil
	}
}
func (dbit *ssDBIt) Value() []byte {
	key := dbit.keys[dbit.index]
	value, err := dbit.db.Get([]byte(key))

	if err != nil {
		dlog.Error("get iterator value error", "key", key)
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
	if len(dbit.keys) <= dbit.index {
		return false
	}
	key := dbit.keys[dbit.index]
	benchmark.read(1, time.Since(start))
	return bytes.HasPrefix([]byte(key), dbit.prefix)
}

type ssDBBatch struct {
	db       *GoSSDB
	batchset map[string]interface{}
	batchdel []string
}

func (db *GoSSDB) NewBatch(sync bool) Batch {
	return &ssDBBatch{db: db, batchset: make(map[string]interface{})}
}

func (db *ssDBBatch) Set(key, value []byte) {
	if len(value) == 0 {
		value = types.EmptyValue
	}
	db.batchset[string(key)] = value
}

func (db *ssDBBatch) Delete(key []byte) {
	db.batchset[string(key)] = []byte{}
	db.batchdel = append(db.batchdel, string(key))
}

// 注意本方法的实现逻辑，因为ssdb没有提供删除和更新同时进行的批量操作；
// 所以这里先执行更新操作（删除的KEY在这里会将VALUE设置为空）；
// 然后再执行删除操作；
// 这样即使中间执行出错，也不会导致删除结果未写入的情况（值已经被置空）；
func (db *ssDBBatch) Write() error {
	start := time.Now()
	client := db.db.newClient()
	if client == nil {
		return SsdbPoolError
	}
	defer client.Close()

	if len(db.batchset) > 0 {
		err := client.MultiSet(db.batchset)
		if err != nil {
			dlog.Error("Write (multi_set)", "error", err)
			return err
		}
	}

	if len(db.batchdel) > 0 {
		err := client.MultiDel(db.batchdel...)
		if err != nil {
			dlog.Error("Write (multi_del)", "error", err)
			return err
		}
	}

	benchmark.read(len(db.batchset)+len(db.batchdel), time.Since(start))
	return nil
}
