// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package p2p



//更新内容：
// 1.p2p 修改为在nat结束后，在启动peer的stream，ping,version 等功能

//2018-3-26 更新内容
// 1. p2p 过滤重复数据，改用blockhash 提换block height
// 2. 增加p2p私钥自动导入到钱包功能

//p2p版本区间
const (
	minP2PVersion = 10020
	maxP2PVersion = 11000
)


//历史版本
const (
	//p2p广播交易哈希而非完整区块数据
	lightBroadCastVersion = 10030
)


// VERSION number
const VERSION = minP2PVersion


