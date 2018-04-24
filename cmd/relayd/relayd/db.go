package relayd

import (
	"gitlab.33.cn/chain33/chain33/common/db"
)

// Store hash and blockHeader
// SPV information

// Namespace keys
var (
	hashKey     = []byte("h")
	blockHeader = []byte("b")
)

type relaydDB struct {
	*db.GoLevelDB
}

func NewRelayDB(name string, dir string, cache int) *relaydDB {
	d, err := db.NewGoLevelDB(name, dir, cache)
	if err != nil {
		panic(err)
	}
	return &relaydDB{d}
}
