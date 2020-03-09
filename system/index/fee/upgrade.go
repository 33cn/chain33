// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fee

import (
	"fmt"

	plugins "github.com/33cn/chain33/system/plugin"
	"github.com/33cn/chain33/types"
)

// CalcTotalFeePrefixOld 获得老的前缀
func CalcTotalFeePrefixOld() []byte {
	return []byte("TotalFeeKey:")
}

// CalcTotalFeePrefix 用于存储地址相关的hash列表，key=TxAddrHash:addr:height*100000 + index
func CalcTotalFeePrefix(name string) []byte {
	return []byte(fmt.Sprintf("%s-%s-%s:", types.LocalPluginPrefix, name, "Fee"))
}

func (p *feePlugin) Upgrade(count int32) (bool, error) {
	toVersion := 2
	prefixes := []plugins.Prefixes{
		{CalcTotalFeePrefixOld(), CalcTotalFeePrefix(name)},
	}
	return plugins.Upgrade(p.GetLocalDB(), name, toVersion, prefixes, count)
}
