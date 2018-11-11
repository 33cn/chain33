// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mempool

import (
	"runtime"

	log "github.com/33cn/chain33/common/log/log15"
)

var (
	mlog                           = log.New("module", "mempool")
	poolCacheSize            int64 = 10240  // mempool容量
	mempoolExpiredInterval   int64 = 600    // mempool内交易过期时间，10分钟
	mempoolReSendInterval    int64 = 60     // mempool内交易重发时间，1分钟
	mempoolDupResendInterval int64 = 120    // mempool重复交易可再次发送间隔，120秒
	mempoolAddedTxSize             = 102400 // 已添加过的交易缓存大小
	maxTxNumPerAccount       int64 = 100    // TODO 每个账户在mempool中最大交易数量，10
	processNum               int
)

// TODO
func init() {
	processNum = runtime.NumCPU()
	//if processNum >= 2 {
	//processNum -= 1
	//}
}
