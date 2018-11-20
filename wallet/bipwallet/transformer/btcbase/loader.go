// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package btcbase

import (
	"github.com/33cn/chain33/wallet/bipwallet/transformer"
)

//不同币种的前缀版本号
var coinPrefix = map[string][]byte{
	"BTC":  {0x00},
	"BCH":  {0x00},
	"BTY":  {0x00},
	"LTC":  {0x30},
	"ZEC":  {0x1c, 0xb8},
	"USDT": {0x00},
}

func init() {
	//注册
	for name, prefix := range coinPrefix {
		transformer.Register(name, &btcBaseTransformer{prefix})
	}
}
