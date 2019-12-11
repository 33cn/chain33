// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
	"time"
)

// ReduceLocalDB 实时精简localdb
func (chain *BlockChain) ReduceLocalDB() {
	chain.reducewg.Add(1)
	defer chain.reducewg.Done()

	flagHeight, err := chain.blockStore.loadFlag(types.ReduceLocaldbHeight)
	if err != nil {
		panic(err)
	}
	if flagHeight < 0 {
		flagHeight = 0
	}
	// 1分钟检测一次是否可以进行reduce localdb
	checkTicker := time.NewTicker(60 * time.Second)
	for {
		select {
		case <-chain.quit:
			return
		case <-checkTicker.C:
			height := chain.GetBlockHeight()
			safetyHeight := height - MaxRollBlockNum  // 当前高度减去最大回滚高度即为安全高度
			if (safetyHeight)/100 > flagHeight/100 {  // 每隔100区块进行一次精简
				chain.blockStore.reduceLocaldb(flagHeight, safetyHeight, true, chain.blockStore.reduceBody,
					func(batch dbm.Batch, height int64) {
						// 记录的时候记录下一个，中断开始执行的就是下一个
						height++
						batch.Set(types.ReduceLocaldbHeight, types.Encode(&types.Int64{Data:height}))
					})
				flagHeight = safetyHeight + 1
				chainlog.Debug("reduceLocaldb ticker", "current height", flagHeight)
			}
		}
	}
}
