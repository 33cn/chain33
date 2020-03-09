// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package txindex

import (
	"fmt"

	plugins "github.com/33cn/chain33/system/index"
	"github.com/33cn/chain33/types"
)

// 这个是value cfg.CalcTxKeyValue(&txresult)

// CalcTxPrefixOld Tx prefix
func CalcTxPrefixOld() []byte {
	return types.TxHashPerfix
}

// CalcTxShortHashPerfixOld short tx
func CalcTxShortHashPerfixOld() []byte {
	return types.TxShortHashPerfix
}

// CalcTxPrefix Calc Tx Prefix
func CalcTxPrefix(name string) []byte {
	return []byte(fmt.Sprintf("%s-%s-%s:", types.LocalPluginPrefix, name, "TX"))
}

// CalcTxShortPerfix Calc Tx Short Prefix
func CalcTxShortPerfix(name string) []byte {
	return []byte(fmt.Sprintf("%s-%s-%s:", types.LocalPluginPrefix, name, "STX"))
}

// Upgrade 支持配置quickIndex, 如果没有不能通过前缀列出来(确认过, 现在已经没有quickIndex=false的))
func (p *txindexPlugin) Upgrade(count int32) (bool, error) {
	toVersion := 2
	prefixes := []plugins.Prefixes{
		{CalcTxPrefixOld(), CalcTxPrefix(name)},
		{CalcTxShortHashPerfixOld(), CalcTxShortPerfix(name)},
	}

	return plugins.Upgrade(p.GetLocalDB(), name, toVersion, prefixes, count)
}
