package store

//store package store the world - state data
import (
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/store/drivers/kvdb"
	"gitlab.33.cn/chain33/chain33/store/drivers/mavl"
	"gitlab.33.cn/chain33/chain33/types"
)

func New(cfg *types.Store) queue.Module {
	storeType := cfg.Name
	if storeType == "mavl" {
		m := mavl.New(cfg)
		return m
	} else if storeType == "kvdb" {
		k := kvdb.New(cfg)
		return k
	} else if storeType == "mtp" {
		// TODO:
		panic("empty")
	}
	panic("Unsupported store type")
}
