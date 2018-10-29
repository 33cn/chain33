package store

import (
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

type StoreCreate func(cfg *types.Store, sub []byte) queue.Module

var regStore = make(map[string]StoreCreate)

func Reg(name string, create StoreCreate) {
	if create == nil {
		panic("Store: Register driver is nil")
	}
	if _, dup := regStore[name]; dup {
		panic("Store: Register called twice for driver " + name)
	}
	regStore[name] = create
}

func Load(name string) (create StoreCreate, err error) {
	if driver, ok := regStore[name]; ok {
		return driver, nil
	}
	return nil, types.ErrNotFound
}
