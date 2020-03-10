// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package system 系统基础插件包
package system

import (
	_ "github.com/33cn/chain33/p2p" // init p2p
	_ "github.com/33cn/chain33/p2pnext"
	_ "github.com/33cn/chain33/system/consensus/init" //register consensus init package
	_ "github.com/33cn/chain33/system/crypto/init"
	_ "github.com/33cn/chain33/system/dapp/init"
	_ "github.com/33cn/chain33/system/mempool/init"
	_ "github.com/33cn/chain33/system/store/init"
)
