package mempool

import (
	"runtime"

	log "github.com/inconshreveable/log15"
)

var (
	mlog                         = log.New("module", "mempool")
	poolCacheSize          int64 = 10240  // mempool容量
	channelSize            int64 = 1024   // channel缓存大小
	mempoolExpiredInterval int64 = 600000 // mempool内交易过期时间，10分钟
	mempoolReSendInterval  int64 = 60000  // mempool内交易重发时间，1分钟
	mempoolAddedTxSize           = 102400 // 已添加过的交易缓存大小
	maxTxNumPerAccount     int64 = 100    // TODO 每个账户在mempool中最大交易数量，10
	processNum             int
)

// TODO
func init() {
	processNum = runtime.NumCPU()
	if processNum >= 2 {
		//processNum -= 1
	}
}
