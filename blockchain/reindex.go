package blockchain

import (
	"strings"

	"gitlab.33.cn/chain33/chain33/common/version"
	"gitlab.33.cn/chain33/chain33/types"
)

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
	}
	if chain.needReIndex(meta) {
		//如果没有开始重建index，那么先del all keys
		if !meta.Indexing {
			chainlog.Info("begin del all keys")
			chain.blockStore.delAllKeys()
			chainlog.Info("end del all keys")
		}
		start := meta.Height
		//reindex 的过程中，会每个高度都去更新meta
		chain.reIndex(start, curheight)
		meta := &types.UpgradeMeta{
			Indexing: false,
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
	if meta.Indexing { //正在index
		return true
	}
	v1 := meta.Version
	v2 := version.GetLocalDBVersion()
	v1arr := strings.Split(v1, ".")
	v2arr := strings.Split(v2, ".")
	if len(v1arr) != 3 || len(v2arr) != 3 {
		panic("upgrade meta version error")
	}
	return v1arr[0] != v2arr[0]
}

func (chain *BlockChain) reIndexOne(height int64) error {
	newbatch := chain.blockStore.NewBatch(true)
	blockdetail, err := chain.GetBlock(height)
	if err != nil {
		chainlog.Error("reindexone.GetBlock", "err", err)
		panic(err)
	}
	if height%1000 == 0 {
		chainlog.Info("reindex -> ", "height", height)
	}
	//保存tx信息到db中 (newbatch, blockdetail)
	err = chain.blockStore.AddTxs(newbatch, blockdetail)
	if err != nil {
		chainlog.Error("reIndexOne indexTxs:", "height", blockdetail.Block.Height, "err", err)
		panic(err)
	}
	meta := &types.UpgradeMeta{
		Indexing: true,
		Version:  version.GetLocalDBVersion(),
		Height:   height + 1,
	}
	newbatch.Set(version.LocalDBMeta, types.Encode(meta))
	return newbatch.Write()
}
