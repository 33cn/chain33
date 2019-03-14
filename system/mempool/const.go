// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mempool

import (
	"runtime"
	//log "github.com/33cn/chain33/common/log/log15"
)

var (
	poolCacheSize          int64 = 10240 // mempool容量
	mempoolExpiredInterval int64 = 600   // mempool内交易过期时间，10分钟
	maxTxNumPerAccount     int64 = 100   // TODO 每个账户在mempool中最大交易数量，10
	maxTxLast              int64 = 10
	processNum             int
)

// TODO
func init() {
	processNum = runtime.NumCPU()
}
