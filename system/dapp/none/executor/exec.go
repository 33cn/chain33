// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"github.com/33cn/chain33/common"
	nty "github.com/33cn/chain33/system/dapp/none/types"
	"github.com/33cn/chain33/types"
)

// Exec_CommitDelayTx exec commit dealy transaction
func (n *None) Exec_CommitDelayTx(commit *nty.CommitDelayTx, tx *types.Transaction, index int) (*types.Receipt, error) {

	receipt := &types.Receipt{Ty: types.ExecOk}
	delayInfo := &nty.CommitDelayLog{}

	delayTxHash := commit.GetDelayTx().Hash()
	if commit.IsBlockHeightDelayTime {
		delayInfo.EndDelayTime = n.GetHeight() + commit.RelativeDelayTime
	} else {
		delayInfo.EndDelayTime = n.GetBlockTime() + commit.RelativeDelayTime
	}
	delayInfo.Submitter = tx.From()
	receipt.KV = append(receipt.KV,
		&types.KeyValue{Key: formatDelayTxKey(delayTxHash), Value: types.Encode(delayInfo)})

	// 交易哈希只做回执展示信息，不需要保存到链上
	delayInfo.DelayTxHash = common.ToHex(delayTxHash)
	receipt.Logs = append(receipt.Logs,
		&types.ReceiptLog{Ty: nty.TyCommitDelayTxLog, Log: types.Encode(delayInfo)})

	// send delay tx to mempool
	_, err := n.GetAPI().SendDelayTx(&types.DelayTx{Tx: commit.GetDelayTx(), EndDelayTime: delayInfo.EndDelayTime})
	if err != nil {
		eLog.Error("Exec_CommitDelayTx", "txHash", common.ToHex(tx.Hash()), "send delay tx err", err)
		return nil, err
	}
	return receipt, nil
}
