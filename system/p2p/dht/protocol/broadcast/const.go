// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package broadcast

import "errors"

// TTL
const (

	// 默认最小区块轻广播的大小， 100KB
	defaultMinLtBlockSize = 100
)

// P2pCacheTxSize p2pcache size of transaction
const (
	//接收的交易哈希过滤缓存设为mempool最大接收交易量
	txRecvFilterCacheNum    = 10240
	blockRecvFilterCacheNum = 1024
)

// pubsub error
var (
	errSnappyDecode = errors.New("errSnappyDecode")
)

// peer msg id
const (
	blockReqMsgID = iota + 1
	blockRespMsgID
)
