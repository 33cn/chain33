package relayd

import (
	"fmt"

	"gitlab.33.cn/chain33/chain33/common/db"
)

// Store hash and blockHeader
// SPV information

// Namespace keys
var (
	blockHashPrefix = []byte("blockHash")
	heightPrefix    = []byte("height")
	orderPrefix     = []byte("order")
)

type relaydDB struct {
	db db.DB
}

func NewRelayDB(name string, dir string, cache int32) *relaydDB {
	d := db.NewDB(name, "goleveldb", dir, cache)
	return &relaydDB{d}
}

func (r *relaydDB) Get(key []byte) ([]byte, error) {
	return r.db.Get(key)
}

func (r *relaydDB) Set(key, value []byte) error {
	return r.db.Set(key, value)
}

func (r *relaydDB) storeOrder() {
	panic("unimplemented")
	// TODO 分两两种存储，一中是prefix + hash，另一种是prefix + status
}

func (r *relaydDB) storeHeader() {
	panic("unimplemented")
	// TODO 分两两种存储，一中是prefix + hash，另一种是prefix + height
}

func (r *relaydDB) queryOrderByHash(hash []byte) ([]byte, error) {
	return r.db.Get(append(orderPrefix, hash...))
}

func (r *relaydDB) queryOrderByStatus(status uint64) [][]byte {
	iter := r.db.Iterator(append(orderPrefix, fmt.Sprintf("%d", status)...), false)
	var orders [][]byte
	for {
		if iter.Next() {
			orders = append(orders, iter.Value())
		} else {
			break
		}
	}
	return orders
}

func (r *relaydDB) BlockHeader(value interface{}) ([]byte, error) {
	switch val := value.(type) {
	case uint64:
		return r.db.Get(makeHeightKey(val))

	case []byte:
		return r.db.Get(makeBlockHashKey(val))

	default:
		panic(val)
	}
}

func makeHeightKey(height uint64) []byte {
	return append(heightPrefix, []byte(fmt.Sprintf("%d", height))...)
}

func makeBlockHashKey(hash []byte) []byte {
	return append(blockHashPrefix, hash...)
}
