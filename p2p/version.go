// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package p2p

//更新内容：
// 1.p2p 修改为在nat结束后，在启动peer的stream，ping,version 等功能

//2018-3-26 更新内容
// 1. p2p 过滤重复数据，改用blockhash 提换block height
// 2. 增加p2p私钥自动导入到钱包功能

//p2p版本区间 10020, 11000

//历史版本
const (
	//p2p广播交易哈希而非完整区块数据
	lightBroadCastVersion = 10030
)

// VERSION number
const VERSION = lightBroadCastVersion

// MainNet Channel = 0x0000

const (
	defaultTestNetChannel = 256
	versionMask           = 0xFFFF
)

// channelVersion = channel << 16 + version
func calcChannelVersion(channel int32) int32 {
	return channel<<16 + VERSION
}

func decodeChannelVersion(channelVersion int32) (channel int32, version int32) {
	channel = channelVersion >> 16
	version = channelVersion & versionMask
	return
}
