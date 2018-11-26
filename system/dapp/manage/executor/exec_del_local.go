// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"fmt"

	pty "github.com/33cn/chain33/system/dapp/manage/types"
	"github.com/33cn/chain33/types"
)

func localKey(key string) []byte {
	return []byte(fmt.Sprintf("LODB-manage-%s", key))
}

// ExecDelLocal_Modify defines  execdellocal modify func
func (c *Manage) ExecDelLocal_Modify(transfer *types.ModifyConfig, tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	set := &types.LocalDBSet{}

	for i := 0; i < len(receipt.Logs); i++ {
		item := receipt.Logs[i]
		if item.Ty == pty.ManageActionModifyConfig {
			var receipt types.ReceiptConfig
			err := types.Decode(item.Log, &receipt)
			if err != nil {
				panic(err) //数据错误了，已经被修改了
			}
			key := receipt.Current.Key
			set.KV = append(set.KV, &types.KeyValue{Key: localKey(key), Value: types.Encode(receipt.Prev)})
			clog.Debug("ExecDelLocal to savelogs", "config ", key, "value", receipt.Prev)
		}
	}
	return set, nil
}
