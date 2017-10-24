package blockchain

import (
	. "common"
	dbm "common/db"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"types"

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

// ����BlockStore����ĵ�ǰblock�߶�
func (bs *BlockStore) Height() int64 {
	bs.mtx.RLock()
	defer bs.mtx.RUnlock()
	return bs.height
}

// ����db�е�block�߶ȵ�BlockStore.Height
func (bs *BlockStore) UpdateHeight() {
	height := LoadBlockStoreHeight(bs.db)
	bs.mtx.Lock()
	bs.height = height
	bs.mtx.Unlock()
	storelog.Info("UpdateHeight", "curblockheight", height)
}

//��db���ݿ��л�ȡָ���߶ȵ�block��Ϣ
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

//  ��������blocks��Ϣ��db���ݿ���
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

// ͨ��tx hash ��db���ݿ��л�ȡtx������Ϣ
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

// ͨ�������洢tx��Ϣ��db��
func (bs *BlockStore) indexTxs(storeBatch dbm.Batch, block *types.Block) error {

	txlen := len(block.Txs)

	for index := 0; index < txlen; index++ {
		// ����txhashֵ
		txbyte, err := proto.Marshal(block.Txs[index])
		if err != nil {
			storelog.Error("indexTxs Encode tx err", "Height", block.Height, "index", index)
			return err
		}
		//����tx hashֵ
		txhash := BytesToHash(txbyte)
		txhashbyte := txhash.Bytes()

		//����txresult ��Ϣ���浽db��
		var txresult types.TxResult
		txresult.Height = block.Height
		txresult.Index = int32(index + 1)
		txresult.Tx = block.Txs[index]

		txresultbyte, err := proto.Marshal(&txresult)
		if err != nil {
			storelog.Error("indexTxs Encode txresult err", "Height", block.Height, "index", index)
			return err
		}
		testtxhash = txhashbyte //test

		storeBatch.Set(txhashbyte, txresultbyte)

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
