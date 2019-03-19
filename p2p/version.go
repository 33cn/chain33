// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package p2p

// VERSION number
var VERSION int32

//更新内容：
// 1.p2p 修改为在nat结束后，在启动peer的stream，ping,version 等功能

//2018-3-26 更新内容
// 1. p2p 过滤重复数据，改用blockhash 提换block height
// 2. 增加p2p私钥自动导入到钱包功能
