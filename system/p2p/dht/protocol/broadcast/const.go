// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package broadcast

import "errors"

//TTL
const (
	// 默认交易开始哈希广播发送的ttl， 首发为1
	defaultLtTxBroadCastTTL = 3
	// 默认交易最大的广播发送ttl， 大于该值不再向外发送
	defaultMaxTxBroadCastTTL = 25
	// 默认区块轻广播的最小交易数
	defaultMinLtBlockTxNum = 5
)

// P2pCacheTxSize p2pcache size of transaction
const (
	//接收的交易哈希过滤缓存设为mempool最大接收交易量
	txRecvFilterCacheNum    = 10240
	blockRecvFilterCacheNum = 1024
	//发送过滤主要用于发送时冗余检测, 发送完即可以被删除, 维护较小缓存数
	txSendFilterCacheNum    = 500
	blockSendFilterCacheNum = 50
	blockCacheNum           = 10
	maxBlockCacheByteSize   = 100 * 1024 * 1024
)

// 内部自定义错误
var (
	errSendMempool      = errors.New("errSendMempool")
	errRecvMempool      = errors.New("errRecvMempool")
	errSendStream       = errors.New("errSendStream")
	errSendBlockChain   = errors.New("errSendBlockChain")
	errBuildBlockFailed = errors.New("errBuildBlockFailed")
)
