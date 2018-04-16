package blockchain

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"

	"github.com/golang/protobuf/proto"
	"gitlab.33.cn/chain33/chain33/common"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/common/difficulty"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	blockLastHeight = []byte("blockLastHeight")
	storeLog        = chainlog.New("submodule", "store")
	lastheaderlock  sync.Mutex
)

//存储block hash对应的blockbody信息
func calcHashToBlockBodyKey(hash []byte) []byte {
	return []byte(fmt.Sprintf("Body:%v", hash))
}

//存储block hash对应的header信息
func calcHashToBlockHeaderKey(hash []byte) []byte {
	return []byte(fmt.Sprintf("Header:%v", hash))
}

//存储block hash对应的block height
func calcHashToHeightKey(hash []byte) []byte {
	return []byte(fmt.Sprintf("Hash:%v", hash))
}

//存储block hash对应的block总难度TD
func calcHashToTdKey(hash []byte) []byte {
	return []byte(fmt.Sprintf("TD:%v", hash))
}

//存储block height 对应的block  hash
func calcHeightToHashKey(height int64) []byte {
	return []byte(fmt.Sprintf("Height:%v", height))
}

type BlockStore struct {
	db        dbm.DB
	client    queue.Client
	height    int64
	lastBlock *types.Block
}

func NewBlockStore(db dbm.DB, client queue.Client) *BlockStore {
	height, err := LoadBlockStoreHeight(db)
	if err != nil {
		chainlog.Info("init::LoadBlockStoreHeight::database may be crash", "err", err.Error())
		if err != types.ErrHeightNotExist {
			panic(err)
		}

	}
	blockStore := &BlockStore{
		height: height,
		db:     db,
		client: client,
	}
	if height == -1 {
		chainlog.Info("load block height error, may be init database", "height", height)
	} else {
		blockdetail, err := blockStore.LoadBlockByHeight(height)
		if err != nil {
			chainlog.Error("init::LoadBlockByHeight::database may be crash")
			panic(err)
		}
		blockStore.lastBlock = blockdetail.GetBlock()
	}
	return blockStore
}

// 返回BlockStore保存的当前block高度
func (bs *BlockStore) Height() int64 {
	return atomic.LoadInt64(&bs.height)
}

// 更新db中的block高度到BlockStore.Height
func (bs *BlockStore) UpdateHeight() {
	height, _ := LoadBlockStoreHeight(bs.db)
	atomic.StoreInt64(&bs.height, height)
	storeLog.Info("UpdateHeight", "curblockheight", height)
}

// 返回BlockStore保存的当前blockheader
func (bs *BlockStore) LastHeader() *types.Header {
	lastheaderlock.Lock()
	defer lastheaderlock.Unlock()

	// 通过lastBlock获取lastheader
	var blockheader = types.Header{}
	if bs.lastBlock != nil {
		blockheader.Version = bs.lastBlock.Version
		blockheader.ParentHash = bs.lastBlock.ParentHash
		blockheader.TxHash = bs.lastBlock.TxHash
		blockheader.StateHash = bs.lastBlock.StateHash
		blockheader.Height = bs.lastBlock.Height
		blockheader.BlockTime = bs.lastBlock.BlockTime
		blockheader.Signature = bs.lastBlock.Signature
		blockheader.Difficulty = bs.lastBlock.Difficulty

		blockheader.Hash = bs.lastBlock.Hash()
		blockheader.TxCount = int64(len(bs.lastBlock.Txs))
	}
	return &blockheader
}

// 更新LastBlock到缓存中
func (bs *BlockStore) UpdateLastBlock(hash []byte) {
	blockdetail, err := bs.LoadBlockByHash(hash)
	if err != nil {
		storeLog.Error("UpdateLastBlock", "hash", common.ToHex(hash), "error", err)
		return
	}
	lastheaderlock.Lock()
	defer lastheaderlock.Unlock()
	if blockdetail != nil {
		bs.lastBlock = blockdetail.Block
	}
	storeLog.Debug("UpdateLastBlock", "UpdateLastBlock", blockdetail.Block.Height, "LastHederhash", common.ToHex(blockdetail.Block.Hash()))
}

//获取最新的block信息
func (bs *BlockStore) LastBlock() *types.Block {
	lastheaderlock.Lock()
	defer lastheaderlock.Unlock()
	if bs.lastBlock != nil {
		return bs.lastBlock
	}
	return nil
}

func (bs *BlockStore) Get(keys *types.LocalDBGet) *types.LocalReplyValue {
	var reply types.LocalReplyValue
	for i := 0; i < len(keys.Keys); i++ {
		key := keys.Keys[i]
		value, _ := bs.db.Get(key)
		reply.Values = append(reply.Values, value)
	}
	return &reply
}

//通过height高度获取BlockDetail信息
func (bs *BlockStore) LoadBlockByHeight(height int64) (*types.BlockDetail, error) {
	//首先通过height获取block hash从db中
	hash, err := bs.GetBlockHashByHeight(height)
	if err != nil {
		return nil, err
	}
	return bs.LoadBlockByHash(hash)
}

//通过hash获取BlockDetail信息
func (bs *BlockStore) LoadBlockByHash(hash []byte) (*types.BlockDetail, error) {
	var blockdetail types.BlockDetail
	var blockheader types.Header
	var blockbody types.BlockBody
	var block types.Block

	//通过hash获取blockheader
	header, err := bs.db.Get(calcHashToBlockHeaderKey(hash))
	if header == nil || err != nil {
		if err != dbm.ErrNotFoundInDb {
			storeLog.Error("LoadBlockByHash calcHashToBlockHeaderKey", "hash", common.ToHex(hash), "err", err)
		}
		return nil, types.ErrHashNotExist
	}
	err = proto.Unmarshal(header, &blockheader)
	if err != nil {
		storeLog.Error("LoadBlockByHash", "err", err)
		return nil, err
	}
	//通过hash获取blockbody
	body, err := bs.db.Get(calcHashToBlockBodyKey(hash))
	if body == nil || err != nil {
		if err != dbm.ErrNotFoundInDb {
			storeLog.Error("LoadBlockByHash calcHashToBlockBodyKey ", "err", err)
		}
		return nil, types.ErrHashNotExist
	}
	err = proto.Unmarshal(body, &blockbody)
	if err != nil {
		storeLog.Error("LoadBlockByHash", "err", err)
		return nil, err
	}
	block.Version = blockheader.Version
	block.ParentHash = blockheader.ParentHash
	block.TxHash = blockheader.TxHash
	block.StateHash = blockheader.StateHash
	block.Height = blockheader.Height
	block.BlockTime = blockheader.BlockTime
	block.Signature = blockheader.Signature
	block.Difficulty = blockheader.Difficulty
	block.Txs = blockbody.Txs

	blockdetail.Receipts = blockbody.Receipts
	blockdetail.Block = &block

	//storeLog.Info("LoadBlockByHash", "Height", block.Height, "Difficulty", blockdetail.Block.Difficulty)

	return &blockdetail, nil
}

//  批量保存blocks信息到db数据库中
func (bs *BlockStore) SaveBlock(storeBatch dbm.Batch, blockdetail *types.BlockDetail) error {

	height := blockdetail.Block.Height
	if len(blockdetail.Receipts) == 0 && len(blockdetail.Block.Txs) != 0 {
		storeLog.Error("SaveBlock Receipts is nil ", "height", height)
	}
	hash := blockdetail.Block.Hash()

	// Save blockbody通过block hash
	var blockbody types.BlockBody
	blockbody.Txs = blockdetail.Block.Txs
	blockbody.Receipts = blockdetail.Receipts

	body, err := proto.Marshal(&blockbody)
	if err != nil {
		storeLog.Error("SaveBlock Marshal blockbody", "height", height, "hash", common.ToHex(hash), "error", err)
		return err
	}
	storeBatch.Set(calcHashToBlockBodyKey(hash), body)

	// Save blockheader通过block hash
	var blockheader types.Header
	blockheader.Version = blockdetail.Block.Version
	blockheader.ParentHash = blockdetail.Block.ParentHash
	blockheader.TxHash = blockdetail.Block.TxHash
	blockheader.StateHash = blockdetail.Block.StateHash
	blockheader.Height = blockdetail.Block.Height
	blockheader.BlockTime = blockdetail.Block.BlockTime
	blockheader.Signature = blockdetail.Block.Signature
	blockheader.Difficulty = blockdetail.Block.Difficulty

	blockheader.Hash = hash
	blockheader.TxCount = int64(len(blockdetail.Block.Txs))

	header, err := proto.Marshal(&blockheader)
	if err != nil {
		storeLog.Error("SaveBlock Marshal blockheader", "height", height, "hash", common.ToHex(hash), "error", err)
		return err
	}

	storeBatch.Set(calcHashToBlockHeaderKey(hash), header)

	//更新最新的block 高度
	heightbytes := types.Encode(&types.Int64{height})
	storeBatch.Set(blockLastHeight, heightbytes)

	//存储block hash和height的对应关系，便于通过hash查询block
	storeBatch.Set(calcHashToHeightKey(hash), heightbytes)

	//存储block height和block hash的对应关系，便于通过height查询block
	storeBatch.Set(calcHeightToHashKey(height), hash)

	storeLog.Debug("SaveBlock success", "blockheight", height, "hash", common.ToHex(hash))
	return nil
}

// 删除block信息从db数据库中
func (bs *BlockStore) DelBlock(storeBatch dbm.Batch, blockdetail *types.BlockDetail) error {

	height := blockdetail.Block.Height
	hash := blockdetail.Block.Hash()

	// del blockbody
	//storeBatch.Delete(calcHashToBlockBodyKey(hash))

	// del blockheader
	//storeBatch.Delete(calcHashToBlockHeaderKey(hash))

	//更新最新的block高度为前一个高度
	bytes := types.Encode(&types.Int64{height - 1})
	storeBatch.Set(blockLastHeight, bytes)

	//删除block hash和height的对应关系
	storeBatch.Delete(calcHashToHeightKey(hash))

	//删除block height和block hash的对应关系，便于通过height查询block
	storeBatch.Delete(calcHeightToHashKey(height))

	//删除block hash和block td的对应关系
	//storeBatch.Delete(calcHashToTdKey(hash))

	storeLog.Debug("DelBlock success", "blockheight", height, "hash", common.ToHex(hash))
	return nil
}

// 通过tx hash 从db数据库中获取tx交易信息
func (bs *BlockStore) GetTx(hash []byte) (*types.TxResult, error) {
	if len(hash) == 0 {
		err := errors.New("input hash is null")
		return nil, err
	}

	rawBytes, err := bs.db.Get(hash)
	if rawBytes == nil || err != nil {
		if err != dbm.ErrNotFoundInDb {
			storeLog.Error("GetTx", "hash", common.ToHex(hash), "err", err)
		}
		err = errors.New("tx not exist")
		return nil, err
	}

	var txResult types.TxResult
	err = proto.Unmarshal(rawBytes, &txResult)
	if err != nil {
		return nil, err
	}
	return &txResult, nil
}

// 通过批量存储tx信息到db中
func (bs *BlockStore) AddTxs(storeBatch dbm.Batch, blockDetail *types.BlockDetail) error {
	kv, err := bs.getLocalKV(blockDetail)
	if err != nil {
		storeLog.Error("indexTxs getLocalKV err", "Height", blockDetail.Block.Height, "err", err)
		return err
	}
	//storelog.Info("add txs kv num", "n", len(kv.KV))
	for i := 0; i < len(kv.KV); i++ {
		if kv.KV[i].Value == nil {
			storeBatch.Delete(kv.KV[i].Key)
		} else {
			storeBatch.Set(kv.KV[i].Key, kv.KV[i].Value)
		}
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

//从db数据库中获取指定hash对应的block高度
func (bs *BlockStore) GetHeightByBlockHash(hash []byte) (int64, error) {

	heightbytes, err := bs.db.Get(calcHashToHeightKey(hash))
	if heightbytes == nil || err != nil {
		if err != dbm.ErrNotFoundInDb {
			storeLog.Error("GetHeightByBlockHash", "error", err)
		}
		return -1, types.ErrHashNotExist
	}
	return decodeHeight(heightbytes)
}

func decodeHeight(heightbytes []byte) (int64, error) {
	var height types.Int64
	err := types.Decode(heightbytes, &height)
	if err != nil {
		//may be old database format json...
		err = json.Unmarshal(heightbytes, &height.Data)
		if err != nil {
			storeLog.Error("GetHeightByBlockHash Could not unmarshal height bytes", "error", err)
			return -1, types.ErrUnmarshal
		}
	}
	return height.Data, nil
}

//从db数据库中获取指定height对应的blockhash
func (bs *BlockStore) GetBlockHashByHeight(height int64) ([]byte, error) {

	hash, err := bs.db.Get(calcHeightToHashKey(height))
	if hash == nil || err != nil {
		if err != dbm.ErrNotFoundInDb {
			storeLog.Error("GetBlockHashByHeight", "error", err)
		}
		return nil, types.ErrHeightNotExist
	}
	return hash, nil
}

//通过blockheight获取blockheader
func (bs *BlockStore) GetBlockHeaderByHeight(height int64) (*types.Header, error) {

	//首先通过height获取block hash从db中
	hash, err := bs.GetBlockHashByHeight(height)
	if err != nil {
		return nil, err
	}
	var header types.Header
	blockheader, err := bs.db.Get(calcHashToBlockHeaderKey(hash))
	if blockheader == nil || err != nil {
		if err != dbm.ErrNotFoundInDb {
			storeLog.Error("GetBlockHeaderByHeight calcHashToBlockHeaderKey", "error", err)
		}
		return nil, types.ErrHashNotExist
	}
	err = proto.Unmarshal(blockheader, &header)
	if err != nil {
		storeLog.Error("GetBlockHerderByHeight", "Could not unmarshal blockheader:", blockheader)
		return nil, err
	}
	return &header, nil
}

//通过blockhash获取blockheader
func (bs *BlockStore) GetBlockHerderByHash(hash []byte) (*types.Header, error) {

	var header types.Header
	blockheader, err := bs.db.Get(calcHashToBlockHeaderKey(hash))
	if blockheader == nil || err != nil {
		if err != dbm.ErrNotFoundInDb {
			storeLog.Error("GetBlockHerderByHash calcHashToBlockHeaderKey ", "err", err)
		}
		return nil, types.ErrHashNotExist
	}
	err = proto.Unmarshal(blockheader, &header)
	if err != nil {
		storeLog.Error("GetBlockHerderByHash", "err", err)
		return nil, err
	}
	return &header, nil
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

//从db数据库中获取指定blockhash对应的block总难度td
func (bs *BlockStore) GetTdByBlockHash(hash []byte) (*big.Int, error) {

	blocktd, err := bs.db.Get(calcHashToTdKey(hash))
	if blocktd == nil || err != nil {
		if err != dbm.ErrNotFoundInDb {
			storeLog.Error("GetTdByBlockHash ", "error", err)
		}
		return nil, types.ErrHashNotExist
	}
	td := new(big.Int)
	return td.SetBytes(blocktd), nil
}

//保存block hash对应的总难度到db中
func (bs *BlockStore) SaveTdByBlockHash(storeBatch dbm.Batch, hash []byte, td *big.Int) error {
	if td == nil {
		return types.ErrInputPara
	}

	storeBatch.Set(calcHashToTdKey(hash), td.Bytes())
	return nil
}

func (bs *BlockStore) NewBatch(sync bool) dbm.Batch {
	storeBatch := bs.db.NewBatch(sync)
	return storeBatch
}

func LoadBlockStoreHeight(db dbm.DB) (int64, error) {
	bytes, err := db.Get(blockLastHeight)
	if bytes == nil || err != nil {
		if err != dbm.ErrNotFoundInDb {
			storeLog.Error("LoadBlockStoreHeight", "error", err)
		}
		return -1, types.ErrHeightNotExist
	}
	return decodeHeight(bytes)
}

// 将收到的block都暂时存储到db中，加入主链之后会重新覆盖。主要是用于chain重组时获取侧链的block使用
func (bs *BlockStore) dbMaybeStoreBlock(blockdetail *types.BlockDetail, sync bool) error {
	if blockdetail == nil {
		return types.ErrInputPara
	}
	height := blockdetail.Block.GetHeight()
	hash := blockdetail.Block.Hash()
	storeBatch := bs.NewBatch(sync)

	// Save blockbody通过block hash
	var blockbody types.BlockBody
	blockbody.Txs = blockdetail.Block.Txs
	blockbody.Receipts = blockdetail.Receipts

	body, err := proto.Marshal(&blockbody)
	if err != nil {
		storeLog.Error("dbMaybeStoreBlock Marshal blockbody", "height", height, "hash", common.ToHex(hash), "error", err)
		return types.ErrMarshal
	}
	storeBatch.Set(calcHashToBlockBodyKey(hash), body)

	// Save blockheader通过block hash
	var blockheader types.Header
	blockheader.Version = blockdetail.Block.Version
	blockheader.ParentHash = blockdetail.Block.ParentHash
	blockheader.TxHash = blockdetail.Block.TxHash
	blockheader.StateHash = blockdetail.Block.StateHash
	blockheader.Height = blockdetail.Block.Height
	blockheader.BlockTime = blockdetail.Block.BlockTime
	blockheader.Signature = blockdetail.Block.Signature
	blockheader.Difficulty = blockdetail.Block.Difficulty
	blockheader.Hash = hash
	blockheader.TxCount = int64(len(blockdetail.Block.Txs))

	header, err := proto.Marshal(&blockheader)
	if err != nil {
		storeLog.Error("dbMaybeStoreBlock Marshal blockheader", "height", height, "hash", common.ToHex(hash), "error", err)
		return types.ErrMarshal
	}
	storeBatch.Set(calcHashToBlockHeaderKey(hash), header)

	//保存block的总难度到db中
	parentHash := blockdetail.Block.ParentHash

	//转换自己的难度成big.int
	difficulty := difficulty.CalcWork(blockdetail.Block.Difficulty)
	//chainlog.Error("dbMaybeStoreBlock Difficulty", "height", height, "Block.Difficulty", blockdetail.Block.Difficulty)
	//chainlog.Error("dbMaybeStoreBlock Difficulty bigint", "height", height, "self.Difficulty", difficulty.BigToCompact(difficulty))

	var blocktd *big.Int
	if height == 0 {
		blocktd = difficulty
	} else {
		parenttd, _ := bs.GetTdByBlockHash(parentHash)
		blocktd = new(big.Int).Add(difficulty, parenttd)
		//chainlog.Error("dbMaybeStoreBlock Difficulty", "height", height, "parenttd.td", difficulty.BigToCompact(parenttd))
		//chainlog.Error("dbMaybeStoreBlock Difficulty", "height", height, "self.td", difficulty.BigToCompact(blocktd))
	}

	err = bs.SaveTdByBlockHash(storeBatch, blockdetail.Block.Hash(), blocktd)
	if err != nil {
		chainlog.Error("dbMaybeStoreBlock SaveTdByBlockHash:", "height", height, "hash", common.ToHex(hash), "err", err)
		return err
	}

	storeBatch.Write()
	return nil
}
