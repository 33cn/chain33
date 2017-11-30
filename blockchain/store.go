package blockchain

import (
	//. "common"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/types"

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
func (bs *BlockStore) LoadBlock(height int64) *types.BlockDetail {

	var blockdetail types.BlockDetail
	blockbytes := bs.db.Get(calcBlockHeightKey(height))
	if blockbytes == nil {
		return nil
	}
	err := proto.Unmarshal(blockbytes, &blockdetail)
	if err != nil {
		storelog.Error("LoadBlock", "Could not unmarshal bytes:", blockbytes)
		return nil
	}
	return &blockdetail
}

//  批量保存blocks信息到db数据库中
func (bs *BlockStore) SaveBlock(storeBatch dbm.Batch, blockdetail *types.BlockDetail) error {

	height := blockdetail.Block.Height
	// Save block
	blockbytes, err := proto.Marshal(blockdetail)
	if err != nil {
		storelog.Error("SaveBlock Could not Encode block", "height", blockdetail.Block.Height, "error", err)
		return err
	}
	storeBatch.Set(calcBlockHeightKey(height), blockbytes)

	bytes, err := json.Marshal(height)
	if err != nil {
		storelog.Error("SaveBlock  Could not marshal hight bytes", "error", err)
		return err
	}
	storeBatch.Set(blockStoreKey, bytes)

	//存储block hash和height的对应关系，便于通过hash查询block
	storeBatch.Set(calcBlockHashKey(blockdetail.Block.Hash()), bytes)

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
func (bs *BlockStore) indexTxs(storeBatch dbm.Batch, blockdetail *types.BlockDetail) error {

	txlen := len(blockdetail.Block.Txs)

	for index := 0; index < txlen; index++ {
		//计算tx hash
		txhash := blockdetail.Block.Txs[index].Hash()

		//构造txresult 信息保存到db中
		var txresult types.TxResult
		txresult.Height = blockdetail.Block.Height
		txresult.Index = int32(index)
		txresult.Tx = blockdetail.Block.Txs[index]
		txresult.Receiptdate = blockdetail.Receipts[index]

		txresultbyte, err := proto.Marshal(&txresult)
		if err != nil {
			storelog.Error("indexTxs Encode txresult err", "Height", blockdetail.Block.Height, "index", index)
			return err
		}

		storeBatch.Set(txhash, txresultbyte)
		//storelog.Debug("indexTxs Set txresult", "Height", blockdetail.Block.Height, "index", index, "txhashbyte", txhash)
	}
	return nil
}

//从db数据库中获取指定hash对应的block高度
func (bs *BlockStore) GetHeightByBlockHash(hash []byte) int64 {

	heightbytes := bs.db.Get(calcBlockHashKey(hash))
	if heightbytes == nil {
		return -1
	}
	var height int64
	err := json.Unmarshal(heightbytes, &height)
	if err != nil {
		storelog.Error("GetHeightByBlockHash Could not unmarshal height bytes", "error", err)
	}
	return height
}
func calcBlockHashKey(hash []byte) []byte {
	return []byte(fmt.Sprintf("Hash:%v", hash))
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
		return -1
	}

	err := json.Unmarshal(bytes, &height)
	if err != nil {
		storelog.Error("LoadBlockStoreHeight Could not unmarshal height bytes", "error", err)
	}
	return height
}
