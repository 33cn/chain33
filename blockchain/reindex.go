// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"strconv"
	"strings"

	"github.com/33cn/chain33/common/version"
	"github.com/33cn/chain33/types"
)

// UpgradeChain 升级localdb
func (chain *BlockChain) UpgradeChain() {
	meta, err := chain.blockStore.GetUpgradeMeta()
	if err != nil {
		panic(err)
	}
	curheight := chain.GetBlockHeight()
	if curheight == -1 {
		meta = &types.UpgradeMeta{
			Version: version.GetLocalDBVersion(),
		}
		err = chain.blockStore.SetUpgradeMeta(meta)
		if err != nil {
			panic(err)
		}
		return
	}
	start := meta.Height
	v1, _, _ := getLocalDBVersion()
	if chain.needReIndex(meta) {
		//如果没有开始重建index，那么先del all keys
		//reindex 的过程中，会每个高度都去更新meta
		if v1 == 1 {
			if !meta.Starting {
				chainlog.Info("begin del all keys")
				chain.blockStore.delAllKeys()
				chainlog.Info("end del all keys")
			}
			chain.reIndex(start, curheight)
		} else if v1 == 2 {
			//如果节点开启isRecordBlockSequence功能，需要按照seq来升级区块信息
			var isSeq bool
			lastSequence, err := chain.blockStore.LoadBlockLastSequence()
			if err != nil || lastSequence < 0 {
				isSeq = false
			} else {
				curheight = lastSequence
				isSeq = true
			}
			chain.reIndexForTable(start, curheight, isSeq)
		}
		meta := &types.UpgradeMeta{
			Starting: false,
			Version:  version.GetLocalDBVersion(),
			Height:   0,
		}
		err = chain.blockStore.SetUpgradeMeta(meta)
		if err != nil {
			panic(err)
		}
	}
}

func (chain *BlockChain) reIndex(start, end int64) {
	for i := start; i <= end; i++ {
		err := chain.reIndexOne(i)
		if err != nil {
			panic(err)
		}
	}
}

func (chain *BlockChain) needReIndex(meta *types.UpgradeMeta) bool {
	if meta.Starting { //正在index
		return true
	}
	v1 := meta.Version
	v2 := version.GetLocalDBVersion()
	v1arr := strings.Split(v1, ".")
	v2arr := strings.Split(v2, ".")
	if len(v1arr) != 3 || len(v2arr) != 3 {
		panic("upgrade meta version error")
	}
	//只支持从低版本升级到高版本
	v1Value, err := strconv.Atoi(v1arr[0])
	if err != nil {
		panic("upgrade meta strconv.Atoi error")
	}
	v2Value, err := strconv.Atoi(v2arr[0])
	if err != nil {
		panic("upgrade meta strconv.Atoi error")
	}
	if v1arr[0] != v2arr[0] && v2Value > v1Value {
		return true
	}
	return false
}

func (chain *BlockChain) reIndexOne(height int64) error {
	newbatch := chain.blockStore.NewBatch(false)
	blockdetail, err := chain.GetBlock(height)
	if err != nil {
		chainlog.Error("reindexone.GetBlock", "err", err)
		panic(err)
	}
	if height%1000 == 0 {
		chainlog.Info("reindex -> ", "height", height, "lastheight", chain.GetBlockHeight())
	}
	//保存tx信息到db中(newbatch, blockdetail)
	err = chain.blockStore.AddTxs(newbatch, blockdetail)
	if err != nil {
		chainlog.Error("reIndexOne indexTxs:", "height", blockdetail.Block.Height, "err", err)
		panic(err)
	}
	meta := &types.UpgradeMeta{
		Starting: true,
		Version:  version.GetLocalDBVersion(),
		Height:   height + 1,
	}
	newbatch.Set(version.LocalDBMeta, types.Encode(meta))
	return newbatch.Write()
}

func (chain *BlockChain) reIndexForTable(start, end int64, isSeq bool) {
	for i := start; i <= end; i++ {
		err := chain.reIndexForTableOne(i, end, isSeq)
		if err != nil {
			panic(err)
		}
	}
	chainlog.Info("reindex:reIndexForTable:complete")
}

// 使用table的方式保存block的header和body以及paratx信息，
// 然后删除对应的bodyPrefix, headerPrefix, heightToHeaderPrefix,key值
// 需要区分是按照seq还是height来升级block
func (chain *BlockChain) reIndexForTableOne(index int64, lastindex int64, isSeq bool) error {
	newbatch := chain.blockStore.NewBatch(false)
	var blockdetail *types.BlockDetail
	var err error
	blockOptType := types.AddBlock

	if isSeq {
		blockdetail, blockOptType, err = chain.blockStore.loadBlockBySequenceOld(index)
	} else {
		blockdetail, err = chain.blockStore.loadBlockByHeightOld(index)
	}
	if err != nil {
		chainlog.Error("reindex:reIndexForTable:load Block Err", "index", index, "isSeq", isSeq, "err", err)
		panic(err)
	}
	height := blockdetail.Block.GetHeight()
	hash := blockdetail.Block.Hash(chain.client.GetConfig())
	curHeight := chain.GetBlockHeight()

	//只重构add时区块的存储方式，del时只需要删除对应区块上的平行链标记即可
	if blockOptType == types.AddBlock {
		if index%1000 == 0 {
			chainlog.Info("reindex -> ", "index", index, "lastindex", lastindex, "isSeq", isSeq)
		}

		saveReceipt := true
		// 精简localdb,情况下直接不对 blockReceipt做保存
		if chain.client.GetConfig().IsEnable("reduceLocaldb") && curHeight-SafetyReduceHeight > height {
			saveReceipt = false
		}
		//使用table格式保存header和body以及paratx标识
		err = chain.blockStore.saveBlockForTable(newbatch, blockdetail, true, saveReceipt)
		if err != nil {
			chainlog.Error("reindex:reIndexForTable", "height", height, "isSeq", isSeq, "err", err)
			panic(err)
		}

		//删除旧的用于存储header和body的key值
		newbatch.Delete(calcHashToBlockHeaderKey(hash))
		newbatch.Delete(calcHashToBlockBodyKey(hash))
		newbatch.Delete(calcHeightToBlockHeaderKey(height))

		// 精简localdb
		if chain.client.GetConfig().IsEnable("reduceLocaldb") && curHeight-SafetyReduceHeight > height {
			chain.reduceIndexTx(newbatch, blockdetail.GetBlock())
			newbatch.Set(types.ReduceLocaldbHeight, types.Encode(&types.Int64{Data: height}))
		}
	} else {
		parakvs, _ := delParaTxTable(chain.blockStore.db, height)
		for _, kv := range parakvs {
			if len(kv.GetKey()) != 0 && kv.GetValue() == nil {
				newbatch.Delete(kv.GetKey())
			}
		}
		// 精简localdb，为了提升效率，所有索引tx均生成，而不从数据库中读取，因此需要删除侧链生成的tx
		if chain.client.GetConfig().IsEnable("reduceLocaldb") && curHeight-SafetyReduceHeight > height {
			chain.deleteTx(newbatch, blockdetail.GetBlock())
		}
	}

	meta := &types.UpgradeMeta{
		Starting: true,
		Version:  version.GetLocalDBVersion(),
		Height:   index + 1,
	}
	newbatch.Set(version.LocalDBMeta, types.Encode(meta))

	return newbatch.Write()
}

// 返回当前版本的V1,V2,V3的值
func getLocalDBVersion() (int, int, int) {

	version := version.GetLocalDBVersion()
	vArr := strings.Split(version, ".")
	if len(vArr) != 3 {
		panic("getLocalDBVersion version error")
	}
	v1, _ := strconv.Atoi(vArr[0])
	v2, _ := strconv.Atoi(vArr[1])
	v3, _ := strconv.Atoi(vArr[2])

	return v1, v2, v3
}
