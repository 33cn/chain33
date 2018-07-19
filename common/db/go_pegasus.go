package db

import (
	"bytes"
	"context"
	"github.com/XiaoMi/pegasus-go-client/pegasus"
	log "github.com/inconshreveable/log15"
	"github.com/syndtr/goleveldb/leveldb/util"
	"gitlab.33.cn/chain33/chain33/types"
	"strings"
	"time"
)

var slog = log.New("module", "db.pegasus")
var bm = &SsdbBench{}
var hashKey = []byte("hash")
var mgetOpts = pegasus.MultiGetOptions{StartInclusive: true, StopInclusive: true}

func init() {
	dbCreator := func(name string, dir string, cache int) (DB, error) {
		return NewPegasusDB(name, dir, cache)
	}
	registerDBCreator(goPegasusDbBackendStr, dbCreator, false)
}

type PegasusDB struct {
	TransactionDB

	cfg    *pegasus.Config
	name   string
	client pegasus.Client
	table  pegasus.TableConnector
}

func NewPegasusDB(name string, dir string, cache int) (*PegasusDB, error) {
	database := &PegasusDB{name: name}
	database.cfg = parsePegasusNodes(dir)

	if database.cfg == nil {
		slog.Error("no valid instance exists, exit!")
		return nil, types.ErrDataBaseDamage
	}
	var err error
	database.client = pegasus.NewClient(*database.cfg)
	tb, err := database.client.OpenTable(context.Background(), database.name)
	if err != nil {
		slog.Error("connect to pegasus error!", "pegasus", database.cfg, "error", err)
		database.client.Close()
		return nil, types.ErrDataBaseDamage
	}
	database.table = tb
	return database, nil
}

// url pattern: ip:port,ip:port
func parsePegasusNodes(url string) *pegasus.Config {
	hosts := strings.Split(url, ",")
	if hosts == nil {
		slog.Error("invalid url")
		return nil
	}

	cfg := &pegasus.Config{hosts}
	return cfg
}

func (db *PegasusDB) Get(key []byte) ([]byte, error) {
	start := time.Now()

	value, err := db.table.Get(context.Background(), hashKey, key)
	if err != nil {
		//slog.Error("Get value error", "error", err, "key", key, "keyhex", hex.EncodeToString(key), "keystr", string(key))
		return nil, err
	}
	if value == nil {
		return nil, ErrNotFoundInDb
	}

	bm.read(1, time.Since(start))
	return value, nil
}

func (db *PegasusDB) BatchGet(keys [][]byte) (values [][]byte, err error) {
	start := time.Now()
	var keylist = []string{}
	for _, v := range keys {
		keylist = append(keylist, string(v))
	}

	vals, err := db.table.MultiGet(context.Background(), hashKey, keys, mgetOpts)
	if err != nil {
		//slog.Error("Get value error", "error", err, "key", key, "keyhex", hex.EncodeToString(key), "keystr", string(key))
		return nil, err
	}
	if vals == nil {
		return nil, ErrNotFoundInDb
	}

	for _, v := range vals {
		values = append(values, v.Value)
	}
	bm.read(1, time.Since(start))
	return values, nil
}

func (db *PegasusDB) Set(key []byte, value []byte) error {
	start := time.Now()
	err := db.table.Set(context.Background(), hashKey, key, value)
	if err != nil {
		slog.Error("Set", "error", err)
		return err
	}
	bm.write(1, time.Since(start))
	return nil
}

func (db *PegasusDB) SetSync(key []byte, value []byte) error {
	return db.Set(key, value)
}

func (db *PegasusDB) Delete(key []byte) error {
	start := time.Now()
	err := db.table.Del(context.Background(), hashKey, key)
	if err != nil {
		slog.Error("Delete", "error", err)
		return err
	}
	bm.write(1, time.Since(start))
	return nil
}

func (db *PegasusDB) DeleteSync(key []byte) error {
	return db.Delete(key)
}

func (db *PegasusDB) Close() {
	db.table.Close()
	db.client.Close()
}

func (db *PegasusDB) Print() {
	//slog.Info(db.pool.Info())
}

func (db *PegasusDB) Stats() map[string]string {
	return make(map[string]string)
}

func (db *PegasusDB) Iterator(prefix []byte, reverse bool) Iterator {
	var (
		err  error
		vals []pegasus.KeyValue
	)

	limit := util.BytesPrefix(prefix)

	it := &PegasusIt{reverse: reverse, index: -1, table: db.table, begin: prefix, end: limit.Limit}

	opts := pegasus.MultiGetOptions{StartInclusive: true, StopInclusive: false, MaxFetchCount: IteratorPageSize}
	if reverse {
		//TODO reverse
		vals, err = db.table.MultiGetRange(context.Background(), hashKey, prefix, limit.Limit, opts)
	} else {
		vals, err = db.table.MultiGetRange(context.Background(), hashKey, prefix, limit.Limit, opts)
	}

	if err != nil {
		slog.Error("create iterator error!")
		return nil
	}

	if len(vals) > 0 {
		it.vals = vals

		// 如果返回的数据大小刚好满足分页，则假设下一页还有数据
		if len(it.vals) == IteratorPageSize {
			it.nextPage = true
			it.tmpEnd = it.vals[IteratorPageSize-1].SortKey
		}
	}

	if len(vals) == 10240 {
		it.nextPage = true
	}
	return it
}

type PegasusIt struct {
	table    pegasus.TableConnector
	vals     []pegasus.KeyValue
	index    int
	reverse  bool
	nextPage bool
	tmpEnd   []byte

	// 迭代开始位置
	begin []byte
	// 迭代结束位置
	end []byte
	// 当前所属的页数（从0开始）
	pageNo int
}

func (dbit *PegasusIt) Close() {
	dbit.index = -1
}

func (dbit *PegasusIt) Next() bool {
	if len(dbit.vals) > dbit.index+1 {
		dbit.index++
		return true
	} else {
		// 如果有下一页数据，则自动抓取
		if dbit.nextPage {
			return dbit.cacheNextPage(dbit.tmpEnd)
		}
		return false
	}
}

func (dbit *PegasusIt) initPage(begin, end []byte) bool {
	var (
		vals []pegasus.KeyValue
		err  error
	)

	opts := pegasus.MultiGetOptions{StartInclusive: false, StopInclusive: false, MaxFetchCount: IteratorPageSize}

	if dbit.reverse {
		//TODO reverse
		vals, err = dbit.table.MultiGetRange(context.Background(), hashKey, begin, end, opts)
	} else {
		vals, err = dbit.table.MultiGetRange(context.Background(), hashKey, begin, end, opts)
	}
	if err != nil {
		slog.Error("get iterator next page error", "error", err, "begin", begin, "end", dbit.end, "reverse", dbit.reverse)
		return false
	}

	if len(vals) > 0 {
		// 这里只改变keys，不改变index
		dbit.vals = vals

		// 如果返回的数据大小刚好满足分页，则假设下一页还有数据
		if len(vals) == IteratorPageSize {
			dbit.nextPage = true
			dbit.tmpEnd = dbit.vals[IteratorPageSize-1].SortKey
		} else {
			dbit.nextPage = false
		}
		return true
	} else {
		return false
	}
}

// 获取下一页的数据
func (dbit *PegasusIt) cacheNextPage(begin []byte) bool {
	if dbit.initPage(begin, dbit.end) {
		dbit.index = 0
		dbit.pageNo++
		return true
	} else {
		return false
	}
}

func (dbit *PegasusIt) findInPage(key []byte) int {
	pos := -1
	for i, v := range dbit.vals {
		if i < dbit.index {
			continue
		}
		if bytes.Compare(key, v.SortKey) < 0 {
			continue
		} else {
			pos = i
			break
		}
	}
	return pos
}

func (dbit *PegasusIt) Seek(key []byte) bool {
	pos := -1
	pos = dbit.findInPage(key)

	// 如果第一页已经找到，不会走入此逻辑
	for pos == -1 && dbit.nextPage {
		if dbit.cacheNextPage(dbit.tmpEnd) {
			pos = dbit.findInPage(key)
		} else {
			break
		}
	}

	dbit.index = pos
	return dbit.Valid()
}

func (dbit *PegasusIt) Rewind() bool {
	// 目前代码的Rewind调用都是在第一页，正常情况下走不到else分支；
	// 但为了代码健壮性考虑，这里增加对else分支的处理
	if dbit.pageNo == 0 {
		dbit.index = 0
		return true
	}

	// 当数据取到第N页的情况时，Rewind需要返回到第一页第一条
	if dbit.initPage(dbit.begin, dbit.end) {
		dbit.index = 0
		dbit.pageNo = 0
		return true
	} else {
		return false
	}
}

func (dbit *PegasusIt) Key() []byte {
	if dbit.index >= 0 && dbit.index < len(dbit.vals) {
		return dbit.vals[dbit.index].SortKey
	} else {
		return nil
	}
}
func (dbit *PegasusIt) Value() []byte {
	if dbit.index >= len(dbit.vals) {
		slog.Error("get iterator value error: index out of bounds")
		return nil
	}

	return dbit.vals[dbit.index].Value
}

func (dbit *PegasusIt) Error() error {
	return nil
}

func (dbit *PegasusIt) ValueCopy() []byte {
	v := dbit.Value()
	value := make([]byte, len(v))
	copy(value, v)
	return value
}

func (dbit *PegasusIt) Valid() bool {
	start := time.Now()
	if dbit.index < 0 {
		return false
	}
	if len(dbit.vals) <= dbit.index {
		return false
	}
	key := dbit.vals[dbit.index].SortKey
	benchmark.read(1, time.Since(start))
	return bytes.HasPrefix(key, dbit.begin)
}

type PegasusBatch struct {
	table    pegasus.TableConnector
	batchset map[string][]byte
	batchdel map[string][]byte
}

func (db *PegasusDB) NewBatch(sync bool) Batch {
	return &PegasusBatch{table: db.table, batchset: make(map[string][]byte), batchdel: make(map[string][]byte)}
}

func (db *PegasusBatch) Set(key, value []byte) {
	db.batchset[string(key)] = value
	delete(db.batchdel, string(key))
}

func (db *PegasusBatch) Delete(key []byte) {
	db.batchset[string(key)] = []byte{}
	db.batchdel[string(key)] = key
}

// 注意本方法的实现逻辑，因为ssdb没有提供删除和更新同时进行的批量操作；
// 所以这里先执行更新操作（删除的KEY在这里会将VALUE设置为空）；
// 然后再执行删除操作；
// 这样即使中间执行出错，也不会导致删除结果未写入的情况（值已经被置空）；
func (db *PegasusBatch) Write() error {
	start := time.Now()

	if len(db.batchset) > 0 {
		var keys [][]byte
		var values [][]byte
		for k, v := range db.batchset {
			keys = append(keys, []byte(k))
			values = append(values, v)
		}
		err := db.table.MultiSet(context.Background(), hashKey, keys, values, 0)
		if err != nil {
			slog.Error("Write (multi_set)", "error", err)
			return err
		}
	}

	if len(db.batchdel) > 0 {
		var dkeys [][]byte
		for _, v := range db.batchdel {
			dkeys = append(dkeys, v)
		}
		err := db.table.MultiDel(context.Background(), hashKey, dkeys)
		if err != nil {
			slog.Error("Write (multi_del)", "error", err)
			return err
		}
	}

	benchmark.write(len(db.batchset)+len(db.batchdel), time.Since(start))
	return nil
}
