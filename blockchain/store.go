package blockchain

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
	"github.com/golang/protobuf/proto"
)

var blockStoreKey = []byte("blockStoreHeight")
var storelog = chainlog.New("submodule", "store")
var MaxTxsPerBlock int64 = 100000

type BlockStore struct {
	db      dbm.DB
	mtx     sync.RWMutex
	qclient queue.Client
	height  int64
}

func NewBlockStore(db dbm.DB, q *queue.Queue) *BlockStore {
	height := LoadBlockStoreHeight(db)
	return &BlockStore{
		height:  height,
		db:      db,
		qclient: q.NewClient(),
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
	if len(blockdetail.Receipts) == 0 && len(blockdetail.Block.Txs) != 0 {
		storelog.Error("SaveBlock Receipts is nil ", "height", blockdetail.Block.Height)
	}
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

	storelog.Debug("SaveBlock success", "blockheight", height)
	return nil
}

// 批量删除block信息从db数据库中
func (bs *BlockStore) DelBlock(storeBatch dbm.Batch, blockdetail *types.BlockDetail) error {
	height := blockdetail.Block.Height
	// del block
	storeBatch.Delete(calcBlockHeightKey(height))
	//更新最新的block高度为前一个高度
	bytes, err := json.Marshal(height - 1)
	if err != nil {
		storelog.Error("DelBlock  Could not marshal hight bytes", "error", err)
		return err
	}
	storeBatch.Set(blockStoreKey, bytes)
	//删除block hash和height的对应关系
	storeBatch.Delete(calcBlockHashKey(blockdetail.Block.Hash()))

	storelog.Debug("DelBlock success", "blockheight", height)
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
func (bs *BlockStore) AddTxs(storeBatch dbm.Batch, blockdetail *types.BlockDetail) error {
	kv, err := bs.getLocakKV(blockdetail)
	if err != nil {
		storelog.Error("indexTxs getLocalKV err", "Height", blockdetail.Block.Height, "err", err)
		return err
	}
	for i := 0; i < len(kv.KV); i++ {
		storeBatch.Set(kv.KV[i].Key, kv.KV[i].Value)
	}
	return nil
}

//通过批量删除tx信息从db中
func (bs *BlockStore) DelTxs(storeBatch dbm.Batch, blockdetail *types.BlockDetail) error {
	//存储key:addr:flag:height ,value:txhash
	//flag :0-->from,1--> to
	//height=height*10000+index 存储账户地址相关的交易
	kv, err := bs.getDelLocakKV(blockdetail)
	if err != nil {
		storelog.Error("indexTxs getLocalKV err", "Height", blockdetail.Block.Height, "err", err)
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

//存储block height对应的block信息
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

func (bs *BlockStore) getLocakKV(detail *types.BlockDetail) (*types.LocalDBSet, error) {
	if bs.qclient == nil {
		panic("client not bind message queue.")
	}
	msg := bs.qclient.NewMessage("execs", types.EventAddBlock, detail)
	bs.qclient.Send(msg, true)
	resp, err := bs.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}
	kv := resp.GetData().(*types.LocalDBSet)
	return kv, nil
}

func (bs *BlockStore) getDelLocakKV(detail *types.BlockDetail) (*types.LocalDBSet, error) {
	if bs.qclient == nil {
		panic("client not bind message queue.")
	}
	msg := bs.qclient.NewMessage("execs", types.EventDelBlock, detail)
	bs.qclient.Send(msg, true)
	resp, err := bs.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}
	kv := resp.GetData().(*types.LocalDBSet)
	return kv, nil
}
