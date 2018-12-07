// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mempool

import (
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

type MempoolCreate func(cfg *types.MemPool, sub []byte) queue.Module

var regMempool = make(map[string]MempoolCreate)

func Reg(name string, create MempoolCreate) {
	if create == nil {
		panic("Mempool: Register driver is nil")
	}
	if _, dup := regMempool[name]; dup {
		panic("Mempool: Register called twice for driver " + name)
	}
	regMempool[name] = create
}

func Load(name string) (create MempoolCreate, err error) {
	if driver, ok := regMempool[name]; ok {
		return driver, nil
	}
	return nil, types.ErrNotFound
}
