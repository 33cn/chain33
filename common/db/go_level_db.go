// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package db

import (
	"bytes"
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/types"
	metrics "github.com/rcrowley/go-metrics"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
)

var llog = log.New("module", "db.goleveldb")

const (
	// degradationWarnInterval specifies how often warning should be printed if the
	// leveldb database cannot keep up with requested writes.
	degradationWarnInterval = time.Minute

	// metricsGatheringInterval specifies the interval to retrieve leveldb database
	// compaction, io and pause stats to report to the user.
	metricsGatheringInterval = 3 * time.Second
)

func init() {
	dbCreator := func(name string, dir string, cache int) (DB, error) {
		return NewGoLevelDB(name, dir, cache)
	}
	registerDBCreator(levelDBBackendStr, dbCreator, false)
	registerDBCreator(goLevelDBBackendStr, dbCreator, false)
}

// GoLevelDB db
type GoLevelDB struct {
	BaseDB
	db *leveldb.DB

	compTimeMeter      metrics.Meter // Meter for measuring the total time spent in database compaction
	compReadMeter      metrics.Meter // Meter for measuring the data read during compaction
	compWriteMeter     metrics.Meter // Meter for measuring the data written during compaction
	writeDelayNMeter   metrics.Meter // Meter for measuring the write delay number due to database compaction
	writeDelayMeter    metrics.Meter // Meter for measuring the write delay duration due to database compaction
	diskSizeGauge      metrics.Gauge // Gauge for tracking the size of all the levels in the database
	diskReadMeter      metrics.Meter // Meter for measuring the effective amount of data read
	diskWriteMeter     metrics.Meter // Meter for measuring the effective amount of data written
	memCompGauge       metrics.Gauge // Gauge for tracking the number of memory compaction
	level0CompGauge    metrics.Gauge // Gauge for tracking the number of table compaction in level0
	nonlevel0CompGauge metrics.Gauge // Gauge for tracking the number of table compaction in non0 level
	seekCompGauge      metrics.Gauge // Gauge for tracking the number of table compaction caused by read opt

	quitChan chan chan error // Quit channel to stop the metrics collection before closing the database
}

// NewGoLevelDB new
func NewGoLevelDB(name string, dir string, cache int) (*GoLevelDB, error) {
	dbPath := path.Join(dir, name+".db")
	if cache == 0 {
		cache = 64
	}
	handles := cache
	if handles < 16 {
		handles = 16
	}
	if cache < 4 {
		cache = 4
	}
	// Open the db and recover any potential corruptions
	db, err := leveldb.OpenFile(dbPath, &opt.Options{
		OpenFilesCacheCapacity: handles,
		BlockCacheCapacity:     cache / 2 * opt.MiB,
		WriteBuffer:            cache / 4 * opt.MiB, // Two of these are used internally
		Filter:                 filter.NewBloomFilter(10),
	})
	if _, corrupted := err.(*errors.ErrCorrupted); corrupted {
		db, err = leveldb.RecoverFile(dbPath, nil)
	}
	if err != nil {
		return nil, err
	}
	database := &GoLevelDB{
		db:       db,
		quitChan: make(chan chan error),
	}

	namespace := ""
	database.compTimeMeter = metrics.NewRegisteredMeter(namespace+"compact/time", nil)
	database.compReadMeter = metrics.NewRegisteredMeter(namespace+"compact/input", nil)
	database.compWriteMeter = metrics.NewRegisteredMeter(namespace+"compact/output", nil)
	database.diskSizeGauge = metrics.NewRegisteredGauge(namespace+"disk/size", nil)
	database.diskReadMeter = metrics.NewRegisteredMeter(namespace+"disk/read", nil)
	database.diskWriteMeter = metrics.NewRegisteredMeter(namespace+"disk/write", nil)
	database.writeDelayMeter = metrics.NewRegisteredMeter(namespace+"compact/writedelay/duration", nil)
	database.writeDelayNMeter = metrics.NewRegisteredMeter(namespace+"compact/writedelay/counter", nil)
	database.memCompGauge = metrics.NewRegisteredGauge(namespace+"compact/memory", nil)
	database.level0CompGauge = metrics.NewRegisteredGauge(namespace+"compact/level0", nil)
	database.nonlevel0CompGauge = metrics.NewRegisteredGauge(namespace+"compact/nonlevel0", nil)
	database.seekCompGauge = metrics.NewRegisteredGauge(namespace+"compact/seek", nil)

	// Start up the metrics gathering and return
	go database.meter(metricsGatheringInterval)

	return database, nil
}

// Get get
func (db *GoLevelDB) Get(key []byte) ([]byte, error) {
	res, err := db.db.Get(key, nil)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, ErrNotFoundInDb
		}
		llog.Error("Get", "error", err)
		return nil, err

	}
	if res == nil {
		return []byte{}, nil
	}
	return res, nil
}

// Set set
func (db *GoLevelDB) Set(key []byte, value []byte) error {
	err := db.db.Put(key, value, nil)
	if err != nil {
		llog.Error("Set", "error", err)
		return err
	}
	return nil
}

// SetSync 同步
func (db *GoLevelDB) SetSync(key []byte, value []byte) error {
	err := db.db.Put(key, value, &opt.WriteOptions{Sync: true})
	if err != nil {
		llog.Error("SetSync", "error", err)
		return err
	}
	return nil
}

// Delete 删除
func (db *GoLevelDB) Delete(key []byte) error {
	err := db.db.Delete(key, nil)
	if err != nil {
		llog.Error("Delete", "error", err)
		return err
	}
	return nil
}

// DeleteSync 删除同步
func (db *GoLevelDB) DeleteSync(key []byte) error {
	err := db.db.Delete(key, &opt.WriteOptions{Sync: true})
	if err != nil {
		llog.Error("DeleteSync", "error", err)
		return err
	}
	return nil
}

// DB db
func (db *GoLevelDB) DB() *leveldb.DB {
	return db.db
}

// Close 关闭
func (db *GoLevelDB) Close() {
	if db.quitChan != nil {
		errc := make(chan error)
		db.quitChan <- errc
		if err := <-errc; err != nil {
			llog.Error("Metrics collection failed", "err", err)
		}
		db.quitChan = nil
	}

	err := db.db.Close()
	if err != nil {
		llog.Error("Close", "error", err)
	}
}

// Print 打印
func (db *GoLevelDB) Print() {
	str, err := db.db.GetProperty("leveldb.stats")
	if err != nil {
		return
	}
	llog.Info("Print", "stats", str)

	iter := db.db.NewIterator(nil, nil)
	for iter.Next() {
		key := iter.Key()
		value := iter.Value()
		//fmt.Printf("[%X]:\t[%X]\n", key, value)
		llog.Info("Print", "key", string(key), "value", string(value))
	}
}

// Stats ...
func (db *GoLevelDB) Stats() map[string]string {
	keys := []string{
		"leveldb.num-files-at-level{n}",
		"leveldb.stats",
		"leveldb.sstables",
		"leveldb.blockpool",
		"leveldb.cachedblock",
		"leveldb.openedtables",
		"leveldb.alivesnaps",
		"leveldb.aliveiters",
	}

	stats := make(map[string]string)
	for _, key := range keys {
		str, err := db.db.GetProperty(key)
		if err == nil {
			stats[key] = str
		}
	}
	return stats
}

// Iterator 迭代器
func (db *GoLevelDB) Iterator(start []byte, end []byte, reverse bool) Iterator {
	if end == nil {
		end = bytesPrefix(start)
	}
	if bytes.Equal(end, types.EmptyValue) {
		end = nil
	}
	r := &util.Range{Start: start, Limit: end}
	it := db.db.NewIterator(r, nil)
	return &goLevelDBIt{it, itBase{start, end, reverse}}
}

// BeginTx call panic when BeginTx not rewrite
func (db *GoLevelDB) BeginTx() (TxKV, error) {
	tx, err := db.db.OpenTransaction()
	if err != nil {
		return nil, err
	}
	return &goLevelDBTx{tx: tx}, nil
}

// CompactRange ...
func (db *GoLevelDB) CompactRange(start, limit []byte) error {
	r := util.Range{Start: start, Limit: limit}
	return db.db.CompactRange(r)
}

// meter periodically retrieves internal leveldb counters and reports them to
// the metrics subsystem.
//
// This is how a LevelDB stats table looks like (currently):
//
//	Compactions
//	 Level |   Tables   |    Size(MB)   |    Time(sec)  |    Read(MB)   |   Write(MB)
//	-------+------------+---------------+---------------+---------------+---------------
//	   0   |          0 |       0.00000 |       1.27969 |       0.00000 |      12.31098
//	   1   |         85 |     109.27913 |      28.09293 |     213.92493 |     214.26294
//	   2   |        523 |    1000.37159 |       7.26059 |      66.86342 |      66.77884
//	   3   |        570 |    1113.18458 |       0.00000 |       0.00000 |       0.00000
//
// This is how the write delay look like (currently):
// DelayN:5 Delay:406.604657ms Paused: false
//
// This is how the iostats look like (currently):
// Read(MB):3895.04860 Write(MB):3654.64712
func (db *GoLevelDB) meter(refresh time.Duration) {
	// Create the counters to store current and previous compaction values
	compactions := make([][]float64, 2)
	for i := 0; i < 2; i++ {
		compactions[i] = make([]float64, 4)
	}
	// Create storage for iostats.
	var iostats [2]float64

	// Create storage and warning log tracer for write delay.
	var (
		delaystats      [2]int64
		lastWritePaused time.Time
	)

	var (
		errc chan error
		merr error
	)

	// Iterate ad infinitum and collect the stats
	for i := 1; errc == nil && merr == nil; i++ {
		// Retrieve the database stats
		stats, err := db.db.GetProperty("leveldb.stats")
		if err != nil {
			llog.Error("Failed to read database stats", "err", err)
			merr = err
			continue
		}
		// Find the compaction table, skip the header
		lines := strings.Split(stats, "\n")
		for len(lines) > 0 && strings.TrimSpace(lines[0]) != "Compactions" {
			lines = lines[1:]
		}
		if len(lines) <= 3 {
			llog.Error("Compaction leveldbTable not found")
			merr = errors.New("compaction leveldbTable not found")
			continue
		}
		lines = lines[3:]

		// Iterate over all the leveldbTable rows, and accumulate the entries
		for j := 0; j < len(compactions[i%2]); j++ {
			compactions[i%2][j] = 0
		}
		for _, line := range lines {
			parts := strings.Split(line, "|")
			if len(parts) != 6 {
				break
			}
			for idx, counter := range parts[2:] {
				value, err := strconv.ParseFloat(strings.TrimSpace(counter), 64)
				if err != nil {
					llog.Error("Compaction entry parsing failed", "err", err)
					merr = err
					continue
				}
				compactions[i%2][idx] += value
			}
		}
		// Update all the requested meters
		if db.diskSizeGauge != nil {
			db.diskSizeGauge.Update(int64(compactions[i%2][0] * 1024 * 1024))
		}
		if db.compTimeMeter != nil {
			db.compTimeMeter.Mark(int64((compactions[i%2][1] - compactions[(i-1)%2][1]) * 1000 * 1000 * 1000))
		}
		if db.compReadMeter != nil {
			db.compReadMeter.Mark(int64((compactions[i%2][2] - compactions[(i-1)%2][2]) * 1024 * 1024))
		}
		if db.compWriteMeter != nil {
			db.compWriteMeter.Mark(int64((compactions[i%2][3] - compactions[(i-1)%2][3]) * 1024 * 1024))
		}
		// Retrieve the write delay statistic
		writedelay, err := db.db.GetProperty("leveldb.writedelay")
		if err != nil {
			llog.Error("Failed to read database write delay statistic", "err", err)
			merr = err
			continue
		}
		var (
			delayN        int64
			delayDuration string
			duration      time.Duration
			paused        bool
		)
		if n, err := fmt.Sscanf(writedelay, "DelayN:%d Delay:%s Paused:%t", &delayN, &delayDuration, &paused); n != 3 || err != nil {
			llog.Error("Write delay statistic not found")
			merr = err
			continue
		}
		duration, err = time.ParseDuration(delayDuration)
		if err != nil {
			llog.Error("Failed to parse delay duration", "err", err)
			merr = err
			continue
		}
		if db.writeDelayNMeter != nil {
			db.writeDelayNMeter.Mark(delayN - delaystats[0])
		}
		if db.writeDelayMeter != nil {
			db.writeDelayMeter.Mark(duration.Nanoseconds() - delaystats[1])
		}
		// If a warning that db is performing compaction has been displayed, any subsequent
		// warnings will be withheld for one minute not to overwhelm the user.
		if paused && delayN-delaystats[0] == 0 && duration.Nanoseconds()-delaystats[1] == 0 &&
			time.Now().After(lastWritePaused.Add(degradationWarnInterval)) {
			llog.Warn("Database compacting, degraded performance")
			lastWritePaused = time.Now()
		}
		delaystats[0], delaystats[1] = delayN, duration.Nanoseconds()

		// Retrieve the database iostats.
		ioStats, err := db.db.GetProperty("leveldb.iostats")
		if err != nil {
			llog.Error("Failed to read database iostats", "err", err)
			merr = err
			continue
		}
		var nRead, nWrite float64
		parts := strings.Split(ioStats, " ")
		if len(parts) < 2 {
			llog.Error("Bad syntax of ioStats", "ioStats", ioStats)
			merr = fmt.Errorf("bad syntax of ioStats %s", ioStats)
			continue
		}
		if n, err := fmt.Sscanf(parts[0], "Read(MB):%f", &nRead); n != 1 || err != nil {
			llog.Error("Bad syntax of read entry", "entry", parts[0])
			merr = err
			continue
		}
		if n, err := fmt.Sscanf(parts[1], "Write(MB):%f", &nWrite); n != 1 || err != nil {
			llog.Error("Bad syntax of write entry", "entry", parts[1])
			merr = err
			continue
		}
		if db.diskReadMeter != nil {
			db.diskReadMeter.Mark(int64((nRead - iostats[0]) * 1024 * 1024))
		}
		if db.diskWriteMeter != nil {
			db.diskWriteMeter.Mark(int64((nWrite - iostats[1]) * 1024 * 1024))
		}
		iostats[0], iostats[1] = nRead, nWrite

		//compCount, err := db.db.GetProperty("leveldb.compcount")
		//if err != nil {
		//	llog.Error("Failed to read database iostats", "err", err)
		//	merr = err
		//	continue
		//}
		//
		//var (
		//	memComp       uint32
		//	level0Comp    uint32
		//	nonLevel0Comp uint32
		//	seekComp      uint32
		//)
		//if n, err := fmt.Sscanf(compCount, "MemComp:%d Level0Comp:%d NonLevel0Comp:%d SeekComp:%d", &memComp, &level0Comp, &nonLevel0Comp, &seekComp); n != 4 || err != nil {
		//	llog.Error("Compaction count statistic not found")
		//	merr = err
		//	continue
		//}
		//db.memCompGauge.Update(int64(memComp))
		//db.level0CompGauge.Update(int64(level0Comp))
		//db.nonlevel0CompGauge.Update(int64(nonLevel0Comp))
		//db.seekCompGauge.Update(int64(seekComp))

		// Sleep a bit, then repeat the stats collection
		select {
		case errc = <-db.quitChan:
			// Quit requesting, stop hammering the database
		case <-time.After(refresh):
			// Timeout, gather a new set of stats
		}
	}
	if nil != merr {
		llog.Error("level-db meter error", "err", merr.Error())
	}

	if errc == nil {
		errc = <-db.quitChan
	}
	errc <- merr
}

type goLevelDBIt struct {
	iterator.Iterator
	itBase
}

// Close 关闭
func (dbit *goLevelDBIt) Close() {
	dbit.Iterator.Release()
}

// Next next
func (dbit *goLevelDBIt) Next() bool {
	if dbit.reverse {
		return dbit.Iterator.Prev() && dbit.Valid()
	}
	return dbit.Iterator.Next() && dbit.Valid()
}

// Seek seek key
func (dbit *goLevelDBIt) Seek(key []byte) bool {

	exist := dbit.Iterator.Seek(key)
	dbKey := dbit.Key()
	if dbit.reverse && !bytes.Equal(dbKey, key) {
		return dbit.Iterator.Prev() && dbit.Valid()
	}
	return exist
}

// Rewind ...
func (dbit *goLevelDBIt) Rewind() bool {
	if dbit.reverse {
		return dbit.Iterator.Last() && dbit.Valid()
	}
	return dbit.Iterator.First() && dbit.Valid()
}

func (dbit *goLevelDBIt) Value() []byte {
	return dbit.Iterator.Value()
}

func cloneByte(v []byte) []byte {
	value := make([]byte, len(v))
	copy(value, v)
	return value
}

func (dbit *goLevelDBIt) ValueCopy() []byte {
	v := dbit.Iterator.Value()
	return cloneByte(v)
}

func (dbit *goLevelDBIt) Valid() bool {
	return dbit.Iterator.Valid() && dbit.checkKey(dbit.Key())
}

type goLevelDBBatch struct {
	db    *GoLevelDB
	batch *leveldb.Batch
	wop   *opt.WriteOptions
	size  int
	len   int
}

// NewBatch new
func (db *GoLevelDB) NewBatch(sync bool) Batch {
	batch := new(leveldb.Batch)
	wop := &opt.WriteOptions{Sync: sync}
	return &goLevelDBBatch{db, batch, wop, 0, 0}
}

func (mBatch *goLevelDBBatch) Set(key, value []byte) {
	mBatch.batch.Put(key, value)
	mBatch.size += len(key)
	mBatch.size += len(value)
	mBatch.len += len(value)
}

func (mBatch *goLevelDBBatch) Delete(key []byte) {
	mBatch.batch.Delete(key)
	mBatch.size += len(key)
	mBatch.len++
}

func (mBatch *goLevelDBBatch) Write() error {
	err := mBatch.db.db.Write(mBatch.batch, mBatch.wop)
	if err != nil {
		llog.Error("Write", "error", err)
		return err
	}
	return nil
}

func (mBatch *goLevelDBBatch) ValueSize() int {
	return mBatch.size
}

// ValueLen  batch数量
func (mBatch *goLevelDBBatch) ValueLen() int {
	return mBatch.len
}

func (mBatch *goLevelDBBatch) Reset() {
	mBatch.batch.Reset()
	mBatch.len = 0
	mBatch.size = 0
}

func (mBatch *goLevelDBBatch) UpdateWriteSync(sync bool) {
	mBatch.wop.Sync = sync
}

type goLevelDBTx struct {
	tx *leveldb.Transaction
}

func (db *goLevelDBTx) Commit() error {
	return db.tx.Commit()
}

func (db *goLevelDBTx) Rollback() {
	db.tx.Discard()
}

// Get get in transaction
func (db *goLevelDBTx) Get(key []byte) ([]byte, error) {
	res, err := db.tx.Get(key, nil)
	if err != nil {
		if err == errors.ErrNotFound {
			return nil, ErrNotFoundInDb
		}
		llog.Error("tx Get", "error", err)
		return nil, err
	}
	return res, nil
}

// Set set in transaction
func (db *goLevelDBTx) Set(key []byte, value []byte) error {
	err := db.tx.Put(key, value, nil)
	if err != nil {
		llog.Error("tx Set", "error", err)
		return err
	}
	return nil
}

// Iterator 迭代器 in transaction
func (db *goLevelDBTx) Iterator(start []byte, end []byte, reverse bool) Iterator {
	if end == nil {
		end = bytesPrefix(start)
	}
	if bytes.Equal(end, types.EmptyValue) {
		end = nil
	}
	r := &util.Range{Start: start, Limit: end}
	it := db.tx.NewIterator(r, nil)
	return &goLevelDBIt{it, itBase{start, end, reverse}}
}

// Begin call panic when Begin not rewrite
func (db *goLevelDBTx) Begin() {
	panic("Begin not impl")
}
