package blockchain

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"

	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
	"github.com/golang/protobuf/proto"
)

var blockStoreKey = []byte("blockStoreHeight")
var storeLog = chainlog.New("submodule", "store")

type BlockStore struct {
	db     dbm.DB
	client queue.Client
	height int64
}

func NewBlockStore(db dbm.DB, q *queue.Queue) *BlockStore {
	height := LoadBlockStoreHeight(db)
	return &BlockStore{
		height: height,
		db:     db,
		client: q.NewClient(),
	}
}

// 返回BlockStore保存的当前block高度
func (bs *BlockStore) Height() int64 {
	return atomic.LoadInt64(&bs.height)
}

// 更新db中的block高度到BlockStore.Height
func (bs *BlockStore) UpdateHeight() {
	height := LoadBlockStoreHeight(bs.db)
	atomic.StoreInt64(&bs.height, height)
	storeLog.Info("UpdateHeight", "curblockheight", height)
}

func (bs *BlockStore) Get(keys *types.LocalDBGet) *types.LocalReplyValue {
	var reply types.LocalReplyValue
	for i := 0; i < len(keys.Keys); i++ {
		key := keys.Keys[i]
		reply.Values = append(reply.Values, bs.db.Get(key))
	}
	return &reply
}

//从db数据库中获取指定高度的block信息
func (bs *BlockStore) LoadBlock(height int64) *types.BlockDetail {
	blockBytes := bs.db.Get(calcBlockHeightKey(height))
	if blockBytes == nil {
		return nil
	}
	var blockDetail types.BlockDetail
	err := proto.Unmarshal(blockBytes, &blockDetail)
	if err != nil {
		storeLog.Error("LoadBlock", "Could not unmarshal bytes:", blockBytes)
		return nil
	}
	return &blockDetail
}

//  批量保存blocks信息到db数据库中
func (bs *BlockStore) SaveBlock(storeBatch dbm.Batch, blockDetail *types.BlockDetail) error {

	height := blockDetail.Block.Height
	if len(blockDetail.Receipts) == 0 && len(blockDetail.Block.Txs) != 0 {
		storeLog.Error("SaveBlock Receipts is nil ", "height", blockDetail.Block.Height)
	}
	// Save block
	blockBytes, err := proto.Marshal(blockDetail)
	if err != nil {
		storeLog.Error("SaveBlock Could not Encode block", "height", blockDetail.Block.Height, "error", err)
		return err
	}
	storeBatch.Set(calcBlockHeightKey(height), blockBytes)

	if bytes, err := json.Marshal(height); err != nil {
		storeLog.Error("SaveBlock  Could not marshal hight bytes", "error", err)
		return err
	} else {
		storeBatch.Set(blockStoreKey, bytes)
		//存储block hash和height的对应关系，便于通过hash查询block
		storeBatch.Set(calcBlockHashKey(blockDetail.Block.Hash()), bytes)
	}
	storeLog.Debug("SaveBlock success", "blockheight", height)
	return nil
}

// 批量删除block信息从db数据库中
func (bs *BlockStore) DelBlock(storeBatch dbm.Batch, blockDetail *types.BlockDetail) error {
	height := blockDetail.Block.Height
	// del block
	storeBatch.Delete(calcBlockHeightKey(height))
	//更新最新的block高度为前一个高度
	if bytes, err := json.Marshal(height - 1); err != nil {
		storeLog.Error("DelBlock  Could not marshal hight bytes", "error", err)
		return err
	} else {
		storeBatch.Set(blockStoreKey, bytes)
	}
	//删除block hash和height的对应关系
	storeBatch.Delete(calcBlockHashKey(blockDetail.Block.Hash()))
	storeLog.Debug("DelBlock success", "blockheight", height)
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
		err := errors.New("tx not exit!")
		return nil, err
	}

	var txResult types.TxResult
	err := proto.Unmarshal(rawBytes, &txResult)
	if err != nil {
		return nil, err
	}
	return &txResult, nil
}

func (bs *BlockStore) NewBatch(sync bool) dbm.Batch {
	storeBatch := bs.db.NewBatch(sync)
	return storeBatch
}

// 通过批量存储tx信息到db中
func (bs *BlockStore) AddTxs(storeBatch dbm.Batch, blockDetail *types.BlockDetail) error {
	kv, err := bs.getLocalKV(blockDetail)
	if err != nil {
		storeLog.Error("indexTxs getLocalKV err", "Height", blockDetail.Block.Height, "err", err)
		return err
	}
	for i := 0; i < len(kv.KV); i++ {
		storeBatch.Set(kv.KV[i].Key, kv.KV[i].Value)
	}
	return nil
}

//通过批量删除tx信息从db中
func (bs *BlockStore) DelTxs(storeBatch dbm.Batch, blockDetail *types.BlockDetail) error {
	//存储key:addr:flag:height ,value:txhash
	//flag :0-->from,1--> to
	//height=height*10000+index 存储账户地址相关的交易
	kv, err := bs.getDelLocalKV(blockDetail)
	if err != nil {
		storeLog.Error("indexTxs getLocalKV err", "Height", blockDetail.Block.Height, "err", err)
		return err
	}
	for i := 0; i < len(kv.KV); i++ {
		if kv.KV[i].Value == nil {
			storeBatch.Delete(kv.KV[i].Key)
		} else {
			storeBatch.Set(kv.KV[i].Key, kv.KV[i].Value)
		}
	}
	return nil
}

//存储block hash对应的block height
func calcBlockHashKey(hash []byte) []byte {
	return []byte(fmt.Sprintf("Hash:%v", hash))
}

//从db数据库中获取指定hash对应的block高度
func (bs *BlockStore) GetHeightByBlockHash(hash []byte) int64 {
	data := bs.db.Get(calcBlockHashKey(hash))
	if data == nil {
		return -1
	}
	var height int64
	err := json.Unmarshal(data, &height)
	if err != nil {
		storeLog.Error("GetHeightByBlockHash Could not unmarshal height data", "error", err)
	}
	return height
}

//存储block height对应的block信息
func calcBlockHeightKey(height int64) []byte {
	return []byte(fmt.Sprintf("H:%v", height))
}

func SaveBlockStoreHeight(db dbm.DB, height int64) {
	bytes, err := json.Marshal(height)
	if err != nil {
		storeLog.Error("SaveBlockStoreHeight  Could not marshal hight bytes", "error", err)
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
		storeLog.Error("LoadBlockStoreHeight Could not unmarshal height bytes", "error", err)
	}
	return height
}

func (bs *BlockStore) getLocalKV(detail *types.BlockDetail) (*types.LocalDBSet, error) {
	if bs.client == nil {
		panic("client not bind message queue.")
	}
	msg := bs.client.NewMessage("execs", types.EventAddBlock, detail)
	bs.client.Send(msg, true)
	resp, err := bs.client.Wait(msg)
	if err != nil {
		return nil, err
	}
	kv := resp.GetData().(*types.LocalDBSet)
	return kv, nil
}

func (bs *BlockStore) getDelLocalKV(detail *types.BlockDetail) (*types.LocalDBSet, error) {
	if bs.client == nil {
		panic("client not bind message queue.")
	}
	msg := bs.client.NewMessage("execs", types.EventDelBlock, detail)
	bs.client.Send(msg, true)
	resp, err := bs.client.Wait(msg)
	if err != nil {
		return nil, err
	}
	kv := resp.GetData().(*types.LocalDBSet)
	return kv, nil
}
