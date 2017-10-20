package blockchain

import (
	. "common"
	dbm "common/db"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"types"
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

// Height() returns the last known contiguous block height.
func (bs *BlockStore) Height() int64 {
	bs.mtx.RLock()
	defer bs.mtx.RUnlock()
	return bs.height
}

func (bs *BlockStore) LoadBlock(height int64) *types.Block {
	var block types.Block
	blockbytes := bs.db.Get(calcBlockHeightKey(height))
	if blockbytes == nil {
		return nil
	}
	err := Decode(blockbytes, &block)
	if err != nil {
		storelog.Error("LoadBlock", "Could not unmarshal bytes:", blockbytes)
		return nil
	}
	return &block
}

func (bs *BlockStore) SaveBlock(block *types.Block) error {
	height := block.Height
	if height != bs.Height()+1 {
		outstr := fmt.Sprintf("BlockStore can only save contiguous blocks. Wanted %v, got %v", bs.Height()+1, height)
		err := errors.New(outstr)
		storelog.Info("SaveBlock can only save contiguous blocks ", "Wanted", bs.Height()+1, "got", height)
		return err
	}

	// Save block
	blockbytes, err := Encode(block)
	if err != nil {
		storelog.Error("SaveBlock Could not Encode block", "height", block.Height, "error", err)
		return err
	}
	bs.db.Set(calcBlockHeightKey(height), blockbytes)

	// Save new BlockStoreHeight descriptor
	SaveBlockStoreHeight(bs.db, height)

	// Done!
	bs.mtx.Lock()
	bs.height = height
	bs.mtx.Unlock()

	// Flush
	bs.db.SetSync(nil, nil)

	//storelog.Info("SaveBlock success", "blockheight", height)
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
