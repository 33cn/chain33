// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
chain33 是由复杂美科技有限公司开发的区块链链框架

1. 高度可定制

2. 丰富的插件库

3. 创新的 合约 调用和组合方式
*/

package chain33

//有些包国内需要翻墙才能下载，我们把部分参见的包含在这里
import (
	_ "golang.org/x/crypto/nacl/box" //register box package
	_ "golang.org/x/crypto/nacl/secretbox"
	_ "golang.org/x/crypto/ssh"
)
