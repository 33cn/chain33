// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"bytes"
	"errors"
	"strconv"
	"time"

	"github.com/33cn/chain33/common"

	"github.com/33cn/chain33/types"
)

var (
	ErrNoBlockToChunk        = errors.New("ErrNoBlockToChunk")
	ErrNoChunkInfoToDownLoad = errors.New("ErrNoChunkInfoToDownLoad")
	ErrNoChunkNumSerial      = errors.New("ErrNoChunkNumSerial")
)

const (
	// 每次检测最大生成chunk数
	OnceMaxChunkNum     int32 = 10
	// 删除小于当前chunk为DelRollbackChunkNum
	DelRollbackChunkNum int32 = 2
	// 每次请求最大MaxReqChunkRecord个chunk的record
	MaxReqChunkRecord   int32 = 500
)

func (chain *BlockChain) ChunkProcessRoutine() {
	defer chain.tickerwg.Done()

	// 1.60s检测一次是否可以删除本地的body数据
	// 2.60s检测一次是否可以触发归档操作

	checkDelTicker := time.NewTicker(60 * time.Second)
	checkGenChunkTicker := time.NewTicker(60 * time.Second) //主动查询当前未归档，然后进行触发
	for {
		select {
		case <-chain.quit:
			return
		case <-checkDelTicker.C:
			// 6.5版本先不做删除
			//chain.CheckDeleteBlockBody()
		case <-checkGenChunkTicker.C:
			chain.CheckGenChunkNum()
		}
	}
}

// CheckGenChunkNum 检测是否需要生成chunkNum
func (chain *BlockChain) CheckGenChunkNum() {
	curMaxSerialChunkNum := chain.getMaxSerialChunkNum()
	height := chain.GetBlockHeight()
	curChunkNum := chain.GetCurChunkNum()
	chunkNum, _, _ := chain.CaclChunkInfo(height)
	if curMaxSerialChunkNum == curChunkNum && curChunkNum >= 0 {
		return
	}
	for i := int32(0); i < OnceMaxChunkNum; i++ {
		num := chain.getMaxSerialChunkNum() + 1
		if num < chunkNum {
			_, err := chain.blockStore.GetKey(calcChunkNumToHash(num))
			if err == nil {
				// 如果存在说明已经进行过归档，则更新连续号
				chain.updateMaxSerialChunkNum(num)
			} else {
				chunk := &types.ChunkInfo{
					ChunkNum: num,
					Start:    num * chain.cfg.ChunkblockNum,
					End:      (num+1)*chain.cfg.ChunkblockNum - 1,
				}
				chain.ChunkShardHandle(chunk, true)
			}
		}
	}
}

// CheckDeleteBlockBody 检测是否需要删除已经归档BlockBody
func (chain *BlockChain) CheckDeleteBlockBody() {
	height := chain.GetBlockHeight()
	maxHeight := chain.GetPeerMaxBlkHeight()
	if maxHeight == -1 {
		return
	}
	if height > maxHeight {
		maxHeight = height
	}
	chain.blockStore.walkOverDeleteChunk(maxHeight, chain.DeleteBlockBody, chain.CaclChunkInfo)
}

// DeleteBlockBody del chunk body
func (chain *BlockChain) DeleteBlockBody(chunkNum int64) []*types.KeyValue {
	value, err := chain.blockStore.GetKey(calcChunkNumToHash(chunkNum))
	if err != nil {
		return nil
	}
	chunk := &types.ChunkInfo{}
	err = types.Decode(value, chunk)
	if err != nil {
		return nil
	}
	var kvs []*types.KeyValue
	for i := chunk.Start; i <= chunk.End; i++ {
		kv, err := chain.deleteBlockBody(i)
		if err != nil {
			continue
		}
		kvs = append(kvs, kv...)
	}
	return kvs
}

func (chain *BlockChain) deleteBlockBody(height int64) (kvs []*types.KeyValue, err error) {
	hash, err := chain.blockStore.GetBlockHashByHeight(height)
	if err != nil {
		chainlog.Error("deleteBlockBody GetBlockHashByHeight", "height", height, "error", err)
		return nil, err
	}
	kvs, err = delBlockBodyTable(chain.blockStore.db, height, hash)
	if err != nil {
		chainlog.Error("deleteBlockBody delBlockBodyTable", "height", height, "error", err)
		return nil, err
	}
	return kvs, err
}

// IsNeedChunk is need chunk
func (chain *BlockChain) IsNeedChunk(height int64) (isNeed bool, chunk *types.ChunkInfo) {
	chunkNum, start, end := chain.CaclChunkInfo(height)
	chunk = &types.ChunkInfo{
		ChunkNum: chunkNum,
		Start:    start,
		End:      end,
	}
	curMaxChunkNum := chain.GetCurChunkNum()
	return curMaxChunkNum < chunkNum, chunk
}

// ShardChunkHandle
func (chain *BlockChain) ChunkShardHandle(chunk *types.ChunkInfo, isNotifyChunk bool) {
	// 1、计算当前chunk信息；
	// 2、通知p2p,如果是挖矿节点则通知，否则则不通知；
	// 3、生成归档记录；
	// 4、生成辅助删除信息；
	// 5、保存归档记录信息；
	// 6、更新chunk最大连续序列号
	start := chunk.Start
	end := chunk.End
	chunkHash, bodys, err := chain.genChunkBlocks(start, end)
	if err != nil {
		storeLog.Error("ShardDataHandle", "chunkNum", chunk.ChunkNum, "start", start, "end", end, "err", err)
		return
	}
	chunk.ChunkHash = chunkHash
	if isNotifyChunk {
		chain.notifyStoreChunkToP2P(chunk)
	}
	chunkRds := genChunkRecord(chunk, bodys)
	kv := chain.genToDeleteChunkSign(chunk.ChunkNum)
	chunkRds.Kvs = append(chunkRds.Kvs, kv)
	chain.saveChunkRecord(chunkRds)
	chain.updateMaxSerialChunkNum(chunk.ChunkNum)
}

func (chain *BlockChain) genToDeleteChunkSign(chunkNum int64) *types.KeyValue {
	maxHeight := chain.GetPeerMaxBlkHeight()
	if maxHeight != -1 && chain.GetBlockHeight() > maxHeight {
		maxHeight = chain.GetBlockHeight()
	}
	kv := &types.KeyValue{
		Key:   calcToDeleteChunkSign(chunkNum),
		Value: types.Encode(&types.Int64{Data: maxHeight}),
	}
	return kv
}

func (chain *BlockChain) getMaxSerialChunkNum() int64 {
	chain.maxSeriallock.Lock()
	defer chain.maxSeriallock.Unlock()
	serial := chain.maxSerialChunkNum
	return serial
}

func (chain *BlockChain) updateMaxSerialChunkNum(chunkNum int64) error {
	chain.maxSeriallock.Lock()
	defer chain.maxSeriallock.Unlock()
	if chain.maxSerialChunkNum+1 != chunkNum {
		return ErrNoChunkNumSerial
	}
	chain.maxSerialChunkNum = chunkNum
	return chain.blockStore.SetMaxSerialChunkNum(chunkNum)
}

func (chain *BlockChain) notifyStoreChunkToP2P(data *types.ChunkInfo) {
	if chain.client == nil {
		chainlog.Error("storeChunkToP2Pstore: chain client not bind message queue.")
		return
	}

	chainlog.Debug("storeChunkToP2Pstore", "chunknum", data.ChunkNum, "block start height",
		data.Start, "block end height", data.End, "chunk hash", common.ToHex(data.ChunkHash))

	msg := chain.client.NewMessage("p2p", types.EventNotifyStoreChunk, data)
	err := chain.client.Send(msg, true)
	if err != nil {
		chainlog.Error("storeChunkToP2Pstore", "chunknum", data.ChunkNum, "block start height",
			data.Start, "block end height", data.End, "chunk hash", common.ToHex(data.ChunkHash), "err", err)
	}
	_, err = chain.client.Wait(msg)
	if err != nil {
		synlog.Error("storeChunkToP2Pstore", "client.Wait err:", err)
		return
	}
}

func (chain *BlockChain) genChunkBlocks(start, end int64) ([]byte, *types.BlockBodys, error) {
	var hashs types.ReplyHashes
	var bodys types.BlockBodys
	for i := start; i <= end; i++ {
		detail, err := chain.blockStore.LoadBlockByHeight(i)
		if err != nil {
			return nil, nil, err
		}
		body := chain.blockStore.BlockdetailToBlockBody(detail)
		bodys.Items = append(bodys.Items, body)
		hashs.Hashes = append(hashs.Hashes, body.Hash)
	}
	return hashs.Hash(), &bodys, nil
}

func (chain *BlockChain) saveChunkRecord(chunkRds *types.ChunkRecords) {
	chain.blockStore.mustSaveKvset(&types.LocalDBSet{KV: chunkRds.GetKvs()})
}

// GetChunkBlockBody 从localdb本地获取chunkbody
func (chain *BlockChain) GetChunkBlockBody(req *types.ReqChunkBlockBody) (*types.BlockBodys, error) {
	if req == nil {
		return nil, types.ErrInvalidParam
	}
	start := req.Start
	end := req.End
	value, err := chain.blockStore.GetKey(calcChunkHashToNum(req.ChunkHash))
	if err == nil {
		chunk := &types.ChunkInfo{}
		err = types.Decode(value, chunk)
		if err != nil {
			return nil, err
		}
		start = chunk.Start
		end = chunk.End
	}
	_, bodys, err := chain.genChunkBlocks(start, end)
	return bodys, err
}

func (chain *BlockChain) AddChunkRecord(req *types.ChunkRecords) {
	dbset := &types.LocalDBSet{}
	for _, kv := range req.Kvs {
		if bytes.Contains(kv.Key, ChunkNumToHash) {
			height, err := strconv.Atoi(string(bytes.TrimPrefix(kv.Key, ChunkNumToHash)))
			if err == nil {
				continue
			}
			dbset.KV = append(dbset.KV, &types.KeyValue{Key: calcRecvChunkNumToHash(int64(height)), Value: kv.Value})
		} else {
			// TODO 其它的前缀暂时不去处理
			dbset.KV = append(dbset.KV, kv)
		}
	}
	chain.blockStore.mustSaveKvset(dbset)
}

func (chain *BlockChain) GetChunkRecord(req *types.ReqChunkRecords) (*types.ChunkRecords, error) {
	rep := &types.ChunkRecords{}
	// TODO req.IsDetail 后面需要再加
	for i := req.Start; i <= req.End; i++ {
		key := append([]byte{}, calcChunkNumToHash(i)...)
		value, err := chain.blockStore.GetKey(key)
		if err != nil {
			return rep, types.ErrNotFound
		}
		rep.Kvs = append(rep.Kvs, &types.KeyValue{Key: key, Value: value})
	}
	if len(rep.Kvs) == 0 {
		return rep, types.ErrNotFound
	}
	return rep, nil
}

// GetCurRecvChunkNum TODO 后续需要放入结构体中，且在内存中保存一份
func (chain *BlockChain) GetCurRecvChunkNum() int64 {
	return chain.blockStore.getCurChunkNum(RecvChunkNumToHash)
}

// GetCurChunkNum TODO 后续需要放入结构体中，且在内存中保存一份
func (chain *BlockChain) GetCurChunkNum() int64 {
	return chain.blockStore.getCurChunkNum(ChunkNumToHash)
}

func (chain *BlockChain) CaclChunkInfo(height int64) (chunkNum, start, end int64) {
	return caclChunkInfo(chain.cfg, height)
}

func caclChunkInfo(cfg *types.BlockChain, height int64) (chunkNum, start, end int64) {
	if cfg.ChunkblockNum == 0 {
		panic("toml chunkblockNum can be zero or have't cfg")
	}
	chunkNum = height / cfg.ChunkblockNum
	start = chunkNum * cfg.ChunkblockNum
	end = start + cfg.ChunkblockNum - 1
	return chunkNum, start, end
}

// genChunkRecord 生成归档索引 1:blockhash--->chunkhash 2:blockHeight--->chunkhash
func genChunkRecord(chunk *types.ChunkInfo, bodys *types.BlockBodys) *types.ChunkRecords {
	chunkRds := &types.ChunkRecords{}
	for _, body := range bodys.Items {
		chunkRds.Kvs = append(chunkRds.Kvs, &types.KeyValue{Key: calcBlockHashToChunkHash(body.Hash), Value: chunk.ChunkHash})
	}
	chunkRds.Kvs = append(chunkRds.Kvs, &types.KeyValue{Key: calcChunkNumToHash(chunk.ChunkNum), Value: types.Encode(chunk)})
	chunkRds.Kvs = append(chunkRds.Kvs, &types.KeyValue{Key: calcChunkHashToNum(chunk.ChunkHash), Value: types.Encode(chunk)})
	return chunkRds
}
