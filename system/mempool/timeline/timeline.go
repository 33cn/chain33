// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found In the LICENSE file.

package timeline

import (
	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/queue"
	drivers "github.com/33cn/chain33/system/mempool"
	"github.com/33cn/chain33/types"
)

var mlog = log.New("module", "mempool/timeline")

func init() {
	drivers.Reg("timeline", New)
}

type subConfig struct {
	PoolCacheSize int64 `json:"poolCacheSize"`
}

func New(cfg *types.MemPool, sub []byte) queue.Module {
	c := drivers.NewMempool(cfg)
	var subcfg subConfig
	types.MustDecode(sub, &subcfg)
	if subcfg.PoolCacheSize == 0 {
		subcfg.PoolCacheSize = cfg.PoolCacheSize
	}
	c.SetQueueCache(newTxCache(int(subcfg.PoolCacheSize)))
	return c
}
