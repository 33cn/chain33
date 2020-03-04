// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package broadcast

//TTL
const (
	DefaultLtTxBroadCastTTL  = 3
	DefaultMaxTxBroadCastTTL = 25
	DefaultMinLtBlockTxNum   = 5
)

// P2pCacheTxSize p2pcache size of transaction
const (
	//接收的交易哈希过滤缓存设为mempool最大接收交易量
	TxRecvFilterCacheNum    = 10240
	BlockRecvFilterCacheNum = 1024
	//发送过滤主要用于发送时冗余检测, 发送完即可以被删除, 维护较小缓存数
	TxSendFilterCacheNum    = 500
	BlockSendFilterCacheNum = 50
	BlockCacheNum           = 10
	MaxBlockCacheByteSize   = 100 * 1024 * 1024
)
