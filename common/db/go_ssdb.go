package db

import (
	"bytes"
	log "github.com/inconshreveable/log15"
	"github.com/syndtr/goleveldb/leveldb/util"

	"encoding/hex"
	"fmt"
	"github.com/pkg/errors"
	"github.com/ssdb/gossdb/ssdb"
	"gitlab.33.cn/chain33/chain33/types"
	"strconv"
	"strings"
)

var dlog = log.New("module", "db.ssdb")
var SsdbClientError = errors.New("client_error")

func init() {
	dbCreator := func(name string, dir string, cache int) (DB, error) {
		return NewGoSSDB(name, dir, cache)
	}
	registerDBCreator(ssDBBackendStr, dbCreator, false)
}

type SsdbNode struct {
	ip   string
	port int
}
type GoSSDB struct {
	TransactionDB
	db    *ssdb.Client
	nodes []*SsdbNode
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
	database.db, err = ssdb.Connect(database.nodes[0].ip, database.nodes[0].port)
	if err != nil {
		dlog.Error("connect to ssdb error!", "ssdb", database.nodes[0])
		return nil, types.ErrDataBaseDamage
	}
	return database, nil
}

func (db *GoSSDB) Get(key []byte) ([]byte, error) {
	dlog.Info("Get", "key", key, "keyhex", hex.EncodeToString(key), "keystr", string(key))
	res, err := ssdbGet(string(key), db.db)

	// SSDB在压力大的情况下会返回错误的client_error，这种情况下可以尝试3次
	for i := 0; i < 3; i++ {
		if err == SsdbClientError {
			res, err = ssdbGet(string(key), db.db)
		} else {
			break
		}
	}

	if err == SsdbClientError {
		dlog.Error("Get", "error", err, "key", key)
	}
	if err != nil {
		dlog.Error("Get", "error", err, "key", key)
		return nil, err
	}
	if res == nil {
		return nil, ErrNotFoundInDb
	}

	if v, ok := res.(string); ok {
		return []byte(v), nil
	} else {
		return nil, ErrNotFoundInDb
	}
}

func ssdbGet(key string, db *ssdb.Client) (interface{}, error) {
	resp, err := db.Do("get", key)
	if err != nil {
		return nil, err
	}
	if len(resp) == 2 && resp[0] == "ok" {
		return resp[1], nil
	}
	if resp[0] == "not_found" {
		return nil, nil
	}
	if resp[0] == "client_error" {
		return nil, SsdbClientError
	}

	return nil, fmt.Errorf("bad response: %v", resp[0])
}

func (db *GoSSDB) Set(key []byte, value []byte) error {
	_, err := db.db.Set(string(key), string(value))
	if err != nil {
		llog.Error("Set", "error", err)
		return err
	}
	return nil
}

func (db *GoSSDB) SetSync(key []byte, value []byte) error {
	return db.Set(key, value)
}

func (db *GoSSDB) Delete(key []byte) error {
	_, err := db.db.Del(string(key))
	if err != nil {
		llog.Error("Delete", "error", err)
		return err
	}
	return nil
}

func (db *GoSSDB) DeleteSync(key []byte) error {
	return db.Delete(key)
}

func (db *GoSSDB) Close() {
	db.db.Close()
}

func (db *GoSSDB) Print() {
	dlog.Info("db info")
}

func (db *GoSSDB) Stats() map[string]string {
	stats := make(map[string]string)
	return stats
}

func (db *GoSSDB) Iterator(prefix []byte, reverse bool) Iterator {
	limit := util.BytesPrefix(prefix)
	var (
		cmd   string
		start []byte
		end   []byte
	)

	if reverse {
		cmd = "rkeys"
		start = prefix
		end = limit.Limit
	} else {
		cmd = "keys"
		start = limit.Limit
		end = prefix
	}

	it := newSSDBIt(prefix, []string{}, reverse, db)
	resp, err := db.db.Do(cmd, string(start), string(end), -1)
	if err != nil || len(resp) == 0 {
		dlog.Error("get iterator error", "error", err)
		return it
	}

	if len(resp) > 0 && resp[0] == "ok" {
		return newSSDBIt(prefix, resp[1:], reverse, db)
	} else {
		dlog.Warn("get iterator empty!", "resp", resp)
		return it
	}
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
	if len(dbit.keys) <= dbit.index {
		return false
	}
	key := dbit.keys[dbit.index]
	return bytes.HasPrefix([]byte(key), dbit.prefix)
}

type ssDBBatch struct {
	db       *ssdb.Client
	batchset map[string]string
	batchdel []string
}

func (db *GoSSDB) NewBatch(sync bool) Batch {
	return &ssDBBatch{db: db.db, batchset: make(map[string]string)}
}

func (db *ssDBBatch) Set(key, value []byte) {
	db.batchset[string(key)] = string(value)
}

func (db *ssDBBatch) Delete(key []byte) {
	db.batchset[string(key)] = ""
	db.batchdel = append(db.batchdel, string(key))
}

// 注意本方法的实现逻辑，因为ssdb没有提供删除和更新同时进行的批量操作；
// 所以这里先执行更新操作（删除的KEY在这里会将VALUE设置为空）；
// 然后再执行删除操作；
// 这样即使中间执行出错，也不会导致删除结果未写入的情况（值已经被置空）；
func (db *ssDBBatch) Write() error {
	args := []interface{}{"multi_set"}

	for k, v := range db.batchset {
		args = append(args, k)
		args = append(args, v)
	}
	_, err := db.db.Do(args...)
	if err != nil {
		dlog.Error("Write (multi_set)", "error", err)
		return err
	}

	args = []interface{}{"multi_del"}
	for _, v := range db.batchdel {
		args = append(args, v)
	}
	_, err = db.db.Do(args...)
	if err != nil {
		dlog.Error("Write (multi_del)", "error", err)
		return err
	}
	return nil
}
