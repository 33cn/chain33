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
	qclient *queue.IClient
	height  int64
}

func NewBlockStore(db dbm.DB, q *queue.Queue) *BlockStore {
	height := LoadBlockStoreHeight(db)
	return &BlockStore{
		height:  height,
		db:      db,
		qclient: q.GetClient(),
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

	storelog.Info("SaveBlock success", "blockheight", height)
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
	storelog.Error("DelBlock success", "blockheight", height)
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

//用于存储地址相关的hash列表，key=TxAddrHash:addr:height*100000 + index
func calcTxAddrHashKey(addr string, heightindex string) []byte {
	return []byte(fmt.Sprintf("TxAddrHash:%s:%s", addr, heightindex))
}

//用于存储地址相关的hash列表，key=TxAddrHash:addr:flag:height*100000 + index
func calcTxAddrDirHashKey(addr string, flag int32, heightindex string) []byte {
	return []byte(fmt.Sprintf("TxAddrDirHash:%s:%d:%s", addr, flag, heightindex))
}

// 通过批量存储tx信息到db中
func (bs *BlockStore) AddTxs(storeBatch dbm.Batch, cacheDB *CacheDB, blockdetail *types.BlockDetail) error {

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
		txresult.Blocktime = blockdetail.Block.BlockTime
		txresultbyte, err := proto.Marshal(&txresult)
		if err != nil {
			storelog.Error("indexTxs Encode txresult err", "Height", blockdetail.Block.Height, "index", index)
			return err
		}

		storeBatch.Set(txhash, txresultbyte)

		//存储key:addr:flag:height ,value:txhash
		//flag :0-->from,1--> to
		//height=height*10000+index 存储账户地址相关的交易
	}
	kv, err := bs.getLocakKV(blockdetail)
	if err != nil {
		storelog.Error("indexTxs getLocalKV err", "Height", blockdetail.Block.Height, "index", index, "err", err)
		return err
	}
	for i := 0; i < len(kv); i++ {
		storeBatch.Set(kv[i].Key, kv[i].Value)
	}
	return nil
}

//通过批量删除tx信息从db中
func (bs *BlockStore) DelTxs(storeBatch dbm.Batch, cacheDB *CacheDB, blockdetail *types.BlockDetail) error {
	for index := 0; index < len(blockdetail.Block.Txs); index++ {
		//计算tx hash
		txhash := blockdetail.Block.Txs[index].Hash()
		storeBatch.Delete(txhash)
	}
	//存储key:addr:flag:height ,value:txhash
	//flag :0-->from,1--> to
	//height=height*10000+index 存储账户地址相关的交易
	kv, err := bs.getDelLocakKV(blockdetail)
	if err != nil {
		storelog.Error("indexTxs getLocalKV err", "Height", blockdetail.Block.Height, "index", index, "err", err)
		return err
	}
	for i := 0; i < len(kv); i++ {
		if kv[i].Value == nil {
			storeBatch.Delete(kv[i].Key)
		} else {
			storeBatch.Set(kv[i].Key, kv[i].Value)
		}
	}
	return nil
}

// 通过addr前缀查找本地址参与的所有交易
func (bs *BlockStore) GetTxsByAddr(addr *types.ReqAddr) (*types.ReplyTxInfos, error) {

	var Prefix []byte
	var key []byte
	var Txinfos [][]byte
	//取最新的交易hash列表
	if addr.GetHeight() == -1 {

		if addr.Flag == 0 { //所有的交易hash列表
			Prefix = calcTxAddrHashKey(addr.GetAddr(), "")
		} else if addr.Flag == 1 { //from的交易hash列表
			Prefix = calcTxAddrDirHashKey(addr.GetAddr(), 1, "")
		} else if addr.Flag == 2 { //to的交易hash列表
			Prefix = calcTxAddrDirHashKey(addr.GetAddr(), 2, "")
		} else {
			err := errors.New("Flag unknow!")
			return nil, err
		}

		Txinfos = bs.db.IteratorScanFromLast(Prefix, addr.Count, addr.Direction)
		if len(Txinfos) == 0 {
			err := errors.New("does not exist tx!")
			return nil, err
		}
	} else { //翻页查找指定的txhash列表
		blockheight := addr.GetHeight()*MaxTxsPerBlock + int64(addr.GetIndex())
		heightstr := fmt.Sprintf("%018d", blockheight)

		if addr.Flag == 0 {
			Prefix = calcTxAddrHashKey(addr.GetAddr(), "")
			key = calcTxAddrHashKey(addr.GetAddr(), heightstr)
		} else if addr.Flag == 1 { //from的交易hash列表
			Prefix = calcTxAddrDirHashKey(addr.GetAddr(), 1, "")
			key = calcTxAddrDirHashKey(addr.GetAddr(), 1, heightstr)
		} else if addr.Flag == 2 { //to的交易hash列表
			Prefix = calcTxAddrDirHashKey(addr.GetAddr(), 2, "")
			key = calcTxAddrDirHashKey(addr.GetAddr(), 2, heightstr)
		} else {
			err := errors.New("Flag unknow!")
			return nil, err
		}

		Txinfos = bs.db.IteratorScan(Prefix, key, addr.Count, addr.Direction)
		if len(Txinfos) == 0 {
			err := errors.New("does not exist tx!")
			return nil, err
		}
	}
	var replyTxInfos types.ReplyTxInfos
	replyTxInfos.TxInfos = make([]*types.ReplyTxInfo, len(Txinfos))

	for index, txinfobyte := range Txinfos {
		var replyTxInfo types.ReplyTxInfo
		err := proto.Unmarshal(txinfobyte, &replyTxInfo)
		if err != nil {
			storelog.Error("GetTxsByAddr proto.Unmarshal!", "err:", err)
			return nil, err
		}
		replyTxInfos.TxInfos[index] = &replyTxInfo
	}
	return &replyTxInfos, nil
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
	if client.qclient == nil {
		panic("client not bind message queue.")
	}
	msg := client.qclient.NewMessage("execs", types.EventAddBlock, detail)
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}
	kv := resp.GetData().(*types.LocalDBSet)
	return kv, nil
}

func (bs *BlockStore) getDelLocakKV(detail *types.BlockDetail) (*types.LocalDBSet, error) {
	if client.qclient == nil {
		panic("client not bind message queue.")
	}
	msg := client.qclient.NewMessage("execs", types.EventDelBlock, detail)
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}
	kv := resp.GetData().(*types.LocalDBSet)
	return kv, nil
}
