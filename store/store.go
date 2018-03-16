package store

//store package store the world - state data
import (
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/store/drivers/kvdb"
	"code.aliyun.com/chain33/chain33/store/drivers/mavl"
	"code.aliyun.com/chain33/chain33/types"
)

func New(cfg *types.Store) Store {
	storeType := cfg.Name
	if storeType == "mavl" {
		m := mavl.New(cfg)
		return m
	} else if storeType == "kvdb" {
		k := kvdb.New(cfg)
		return k
	} else if storeType == "mtp" {
		// TODO:
	}
	panic("Unsupported store type")
}

type Store interface {
	SetQueue(q *queue.Queue)
	Close()
}
