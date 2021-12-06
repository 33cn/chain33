// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package broadcast

import (
	"time"

	"github.com/33cn/chain33/types"
)

const (
	defaultMaxBatchTxNum      = 100
	defaultMaxBatchTxInterval = 100 // ms
)

func (p *broadcastProtocol) handleSendBatchTx(batchChan chan interface{}) {

	interval := time.Millisecond * time.Duration(p.cfg.MaxBatchTxInterval)
	batchNum := p.cfg.MaxBatchTxNum
	ticker := time.NewTicker(interval)

	defer p.ps.Unsub(batchChan)
	txs := &types.Transactions{Txs: make([]*types.Transaction, 0, batchNum)}
	readySend := false
	for {

		select {
		case <-p.Ctx.Done():
			ticker.Stop()
			return
		case msg := <-batchChan:
			txs.Txs = append(txs.Txs, msg.(*types.Transaction))
			if len(txs.Txs) >= batchNum {
				readySend = true
			}
		case <-ticker.C:
			readySend = true
		}

		if !readySend || len(txs.Txs) == 0 {
			continue
		}
		p.ps.Pub(publishMsg{msg: txs, topic: psBatchTxTopic}, psBroadcast)
		ticker.Reset(interval)
		// 异步处理, 需要重新申请对象
		txs = &types.Transactions{Txs: make([]*types.Transaction, 0, batchNum)}
		readySend = false
	}

}
