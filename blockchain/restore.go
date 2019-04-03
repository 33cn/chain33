// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"strings"

	"fmt"

	"github.com/33cn/chain33/common/version"
	"github.com/33cn/chain33/types"
)

// Upgrade 升级localDB和storeDB
func (chain *BlockChain) Upgrade() {
	chain.UpgradeChain()
	chain.UpgradeStore()
}

//UpgradeStore 升级storedb
func (chain *BlockChain) UpgradeStore() {
	meta, err := chain.blockStore.GetStoreUpgradeMeta()
	if err != nil {
		panic(err)
	}
	curheight := chain.GetBlockHeight()
	if curheight == -1 {
		meta = &types.UpgradeMeta{
			Version: version.GetStoreDBVersion(),
		}
		err = chain.blockStore.SetStoreUpgradeMeta(meta)
		if err != nil {
			panic(err)
		}
	}
	if chain.needReExec(meta) {
		start := meta.Height
		//reExecBlock 的过程中，会每个高度都去更新meta
		chain.ReExecBlock(start, curheight)
		meta := &types.UpgradeMeta{
			Starting: false,
			Version:  version.GetStoreDBVersion(),
			Height:   0,
		}
		err = chain.blockStore.SetStoreUpgradeMeta(meta)
		if err != nil {
			panic(err)
		}
	}
}

// ReExecBlock 从对应高度本地重新执行区块
func (chain *BlockChain) ReExecBlock(startHeight, curHeight int64) {
	var prevStateHash []byte
	if startHeight > 0 {
		blockdetail, err := chain.GetBlock(startHeight - 1)
		if err != nil {
			panic(fmt.Sprintf("get height=%d err, this not allow fail", startHeight-1))
		}
		prevStateHash = blockdetail.Block.StateHash
	}
	for i := startHeight; i <= curHeight; i++ {
		blockdetail, err := chain.GetBlock(i)
		if err != nil {
			panic(fmt.Sprintf("get height=%d err, this not allow fail", i))
		}
		block := blockdetail.Block
		err = execBlockUpgrade(chain.client, prevStateHash, block, false)
		if err != nil {
			panic(fmt.Sprintf("execBlockEx height=%d err=%s, this not allow fail", i, err.Error()))
		}
		prevStateHash = block.StateHash
		//更新高度
		err = chain.upgradeMeta(i)
		if err != nil {
			panic(err)
		}
	}
}

func (chain *BlockChain) needReExec(meta *types.UpgradeMeta) bool {
	if meta.Starting { //正在
		return true
	}
	v1 := meta.Version
	v2 := version.GetStoreDBVersion()
	v1arr := strings.Split(v1, ".")
	v2arr := strings.Split(v2, ".")
	if len(v1arr) != 3 || len(v2arr) != 3 {
		panic("upgrade store meta version error")
	}
	return v1arr[0] != v2arr[0]
}

func (chain *BlockChain) upgradeMeta(height int64) error {
	meta := &types.UpgradeMeta{
		Starting: true,
		Version:  version.GetStoreDBVersion(),
		Height:   height + 1,
	}
	return chain.blockStore.SetStoreUpgradeMeta(meta)
}
