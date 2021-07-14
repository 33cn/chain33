// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import "github.com/33cn/chain33/types"

/*
 * 实现区块回退时本地执行的数据清除
 */

// ExecDelLocal localdb kv数据自动回滚接口
func (n *None) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return nil, nil
}
