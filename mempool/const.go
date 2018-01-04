package mempool

import (
	"errors"
	"runtime"

	log "github.com/inconshreveable/log15"
)

var poolCacheSize int64 = 10240           // mempool容量
var channelSize int64 = 1024              // channel缓存大小
var minTxFee int64 = 10000000             // 最低交易费
var maxMsgByte int64 = 100000             // 交易消息最大字节数
var mempoolExpiredInterval int64 = 600000 // mempool内交易过期时间，10分钟
var mempoolAddedTxSize int = 102400       // 已添加过的交易缓存大小
var maxTxNumPerAccount int64 = 10000
var mlog = log.New("module", "mempool")
var processNum int

func init() {
	processNum = runtime.NumCPU()
	if processNum >= 2 {
		//processNum -= 1
	}
}

// error codes
var (
	//	e00 = errors.New("success")
	e01 = errors.New("transaction exists")
	e02 = errors.New("low transaction fee")
	e03 = errors.New("you have too many transactions")
	e04 = errors.New("wrong signature")
	e05 = errors.New("low balance")
	e06 = errors.New("message too big")
	e07 = errors.New("message expired")
	e08 = errors.New("loadacconts error")
	e09 = errors.New("empty transaction")
	e10 = errors.New("duplicated transaction")
	e11 = errors.New("mempool not ready")
	e12 = errors.New("mempool is full")
)
