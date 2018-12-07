// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package timeline

import (
	"runtime"
	//log "github.com/33cn/chain33/common/log/log15"
)

var (
	poolCacheSize            int64 = 10240 // mempool容量
	mempoolDupResendInterval int64 = 120   // mempool重复交易可再次发送间隔，120秒
	maxTxNumPerAccount       int64 = 100   // TODO 每个账户在mempool中最大交易数量，10
	processNum               int
)

// TODO
func init() {
	processNum = runtime.NumCPU()
}
