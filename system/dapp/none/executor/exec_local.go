// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	nty "github.com/33cn/chain33/system/dapp/none/types"
	"github.com/33cn/chain33/types"
)

// ExecLocal_CommitDelayTx exec local commit delay tx
func (n *None) ExecLocal_CommitDelayTx(commit *nty.CommitDelayTx, tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	return nil, nil
}
