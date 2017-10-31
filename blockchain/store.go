package blockchain

import (
	//. "common"
	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/types"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/golang/protobuf/proto"
)

var blockStoreKey = []byte("blockStoreHeight")

var storelog = chainlog.New("submodule", "store")

type BlockStore struct {
	db     dbm.DB
	mtx    sync.RWMutex
	height int64
}

func NewBlockStore(db dbm.DB) *BlockStore {
	height := LoadBlockStoreHeight(db)
	return &BlockStore{
		height: height,
		db:     db,
	}
}

// 返回BlockStore保存的当前block高度
func (bs *BlockStore) Height() int64 {
	bs.mtx.RLock()
	defer bs.mtx.RUnlock()
	return bs.height
}

// 更新db中的block高度到BlockStore.Height
func (bs *BlockStore) UpdateHeight() {
	height := LoadBlockStoreHeight(bs.db)
	bs.mtx.Lock()
	bs.height = height
	bs.mtx.Unlock()
	storelog.Info("UpdateHeight", "curblockheight", height)
}

//从db数据库中获取指定高度的block信息
func (bs *BlockStore) LoadBlock(height int64) *types.Block {

	var block types.Block
	blockbytes := bs.db.Get(calcBlockHeightKey(height))
	if blockbytes == nil {
		return nil
	}
	err := proto.Unmarshal(blockbytes, &block)
	if err != nil {
		storelog.Error("LoadBlock", "Could not unmarshal bytes:", blockbytes)
		return nil
	}
	return &block
}

//  批量保存blocks信息到db数据库中
func (bs *BlockStore) SaveBlock(storeBatch dbm.Batch, block *types.Block) error {

	height := block.Height
	// Save block
	blockbytes, err := proto.Marshal(block)
	if err != nil {
		storelog.Error("SaveBlock Could not Encode block", "height", block.Height, "error", err)
		return err
	}
	storeBatch.Set(calcBlockHeightKey(height), blockbytes)

	bytes, err := json.Marshal(height)
	if err != nil {
		storelog.Error("SaveBlock  Could not marshal hight bytes", "error", err)
		return err
	}
	storeBatch.Set(blockStoreKey, bytes)
	storelog.Info("SaveBlock success", "blockheight", height)
	return nil
}

// 通过tx hash 从db数据库中获取tx交易信息
func (bs *BlockStore) GetTx(hash []byte) (*types.TxResult, error) {
	if len(hash) == 0 {
		err := errors.New("input hash is null")
		return nil, err
	}

	rawBytes := bs.db.Get(hash)
	if rawBytes == nil {
		return nil, nil
	}

	var txresult types.TxResult
	err := proto.Unmarshal(rawBytes, &txresult)
	if err != nil {
		return nil, err
	}
	return &txresult, nil
}
func (bs *BlockStore) NewBatch(sync bool) dbm.Batch {
	storeBatch := bs.db.NewBatch(sync)
	return storeBatch
}

// 通过批量存储tx信息到db中
func (bs *BlockStore) indexTxs(storeBatch dbm.Batch, block *types.Block) error {

	txlen := len(block.Txs)

	for index := 0; index < txlen; index++ {
		//计算tx hash
		txhash := block.Txs[index].Hash()

		//构造txresult 信息保存到db中
		var txresult types.TxResult
		txresult.Height = block.Height
		txresult.Index = int32(index)
		txresult.Tx = block.Txs[index]

		txresultbyte, err := proto.Marshal(&txresult)
		if err != nil {
			storelog.Error("indexTxs Encode txresult err", "Height", block.Height, "index", index)
			return err
		}

		storeBatch.Set(txhash, txresultbyte)

		//storelog.Debug("indexTxs Set txresult", "Height", block.Height, "index", index, "txhashbyte", txhashbyte)
	}
	return nil
}

func calcBlockHeightKey(height int64) []byte {
	return []byte(fmt.Sprintf("H:%v", height))
}

func SaveBlockStoreHeight(db dbm.DB, height int64) {
	bytes, err := json.Marshal(height)
	if err != nil {
		storelog.Error("SaveBlockStoreHeight  Could not marshal hight bytes", "error", err)
	}
	db.SetSync(blockStoreKey, bytes)
}

func LoadBlockStoreHeight(db dbm.DB) int64 {
	var height int64
	bytes := db.Get(blockStoreKey)
	if bytes == nil {
		return 0
	}

	err := json.Unmarshal(bytes, &height)
	if err != nil {
		storelog.Error("LoadBlockStoreHeight Could not unmarshal height bytes", "error", err)
	}
	return height
}
