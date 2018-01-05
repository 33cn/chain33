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
var mempoolReSendInterval int64 = 60000   // mempool内交易过期时间，10分钟
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
	txExistErr      = errors.New("transaction exists")
	lowFeeErr       = errors.New("low transaction fee")
	manyTxErr       = errors.New("you have too many transactions")
	signErr         = errors.New("wrong signature")
	lowBalanceErr   = errors.New("low balance")
	bigMsgErr       = errors.New("message too big")
	expireErr       = errors.New("message expired")
	loadAccountsErr = errors.New("loadacconts error")
	emptyTxErr      = errors.New("empty transaction")
	dupTxErr        = errors.New("duplicated transaction")
	memNotReadyErr  = errors.New("mempool not ready")
	memFullErr      = errors.New("mempool is full")
)
