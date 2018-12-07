// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found In the LICENSE file.

package timeline

import (
	clog "github.com/33cn/chain33/common/log"
	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/queue"
	drivers "github.com/33cn/chain33/system/mempool"
	"github.com/33cn/chain33/types"
)

func SetLogLevel(level string) {
	clog.SetLogLevel(level)
}

func DisableLog() {
	mlog.SetHandler(log.DiscardHandler())
}

var mlog = log.New("module", "mempool/timeline")

type Mempool struct {
	*drivers.BaseMempool
}

func NewMempool(cfg *types.MemPool) *Mempool {
	c := drivers.NewMempool(cfg)
	initConfig(cfg)
	pool := &Mempool{BaseMempool: c}
	pool.BaseCache.SetChild(newTxCache(poolCacheSize, pool.BaseCache))
	return pool
}

func init() {
	drivers.Reg("timeline", New)
}

func New(cfg *types.MemPool, sub []byte) queue.Module {
	c := drivers.NewMempool(cfg)
	pool := &Mempool{BaseMempool: c}
	pool.BaseCache.SetChild(newTxCache(poolCacheSize, pool.BaseCache))
	return pool
}

func initConfig(Cfg *types.MemPool) {
	if Cfg.PoolCacheSize > 0 {
		poolCacheSize = Cfg.PoolCacheSize
	}
	if Cfg.MaxTxNumPerAccount > 0 {
		maxTxNumPerAccount = Cfg.MaxTxNumPerAccount
	}
}

//Resize 设置mempool容量
func (mem *Mempool) Resize(size int) {
	mem.ProxyMtx.Lock()
	mem.BaseCache = drivers.NewBaseCache(int64(size))
	mem.BaseCache.SetChild(newTxCache(int64(size), mem.BaseCache))
	mem.ProxyMtx.Unlock()
}
