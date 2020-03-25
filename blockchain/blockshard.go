// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"bytes"
	"errors"
	"github.com/33cn/chain33/common"
	"strconv"
	"time"

	"github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
)

var (
	ErrNoBlockToChunk        = errors.New("ErrNoBlockToChunk")
	ErrNoChunkInfoToDownLoad = errors.New("ErrNoChunkInfoToDownLoad")
	triggerChunkFlag         bool
)

const (
	MaxCheckForwardChunkNum  int32 = 10
	MaxReqChunkRecord        int32 = 500
)

func (chain *BlockChain) DeleteHaveChunkData() {
	defer chain.tickerwg.Done()
	// 1.60s检测一次是否可以进行归档，删除本地的body数据
	// 2.60s检测一次是否可以触发归档操作，这个触发需要上个挖矿节点内部进行触发(主要针对比如网络分布式节点刚刚启动时
	// 候需要分布式处理，或者上次挖矿节点没有将归档请求发出)
	checkTicker := time.NewTicker(60 * time.Second)
	trigChunkTicker := time.NewTicker(60 * time.Second) //主动查询当前未归档，然后进行触发
	// 3.最大查询距离当前chunkNum范围MaxCheckForwardChunkNum之内是否已经归档
	// 4. TODO 如果DHT中链接的单独一个节点则挖矿节点需要存储否则，不存储（这个可以通过检测节点）
	// 5. TODO 触发升级如果么有升级之前，则进行之前的触发升级
	for {
		select {
		case <-chain.quit:
			return
		case <-checkTicker.C:
			height := chain.GetBlockHeight()
			curChunkNum := chain.GetCurChunkNum()
			chunkNum, _, _ := chain.CaclChunkInfo(height)
			if chunkNum > curChunkNum+1 {
				chain.DeleteLocalBlockBody(chunkNum) // TODO 同步进行删除，可以在notify之后返回参数是否可以删除，
				                                     // TODO 否则TriggerCheckChunkNum下无法准确删除(这种情况下需要记录发出时候的高度， 然后可删除高度 > curChunkNum+1)
			}
		case <-trigChunkTicker.C:
			if getTriggerChunkFlag() {
				chain.TriggerCheckChunkNum() //TODO 该种情况下的body数据也需要删除
				//置位
				setTriggerChunkFlag(false)
			}
		}
	}
}

// TriggerCheckChunkNum 触发检测chunkNum的连续性，如果不连续则处理
func (chain *BlockChain) TriggerCheckChunkNum() []int64 {
	var Nums []int64
	curChunkNum := chain.GetCurChunkNum()
	for i := 1; i <= int(MaxCheckForwardChunkNum); i++ {
		chunkNum := curChunkNum - int64(i)
		if chunkNum >= 0 {
			_, err := chain.blockStore.GetKey(calcChunkNumToHash(chunkNum))
			if err != nil {
				Nums = append(Nums, chunkNum)
			}
		}
	}
	if len(Nums) > 0 {
		for _, num := range Nums  {
			chunk := &types.ChunkInfo{
				ChunkNum: num,
				Start: num * chain.cfg.ChunkblockNum,
				End: (num + 1)* chain.cfg.ChunkblockNum - 1,
			}
			chain.ShardChunkHandle(chunk, true)
		}
	}
	return Nums
}

func (chain *BlockChain) DeleteLocalBlockBody(chunkNum int64) {
	value, err := chain.blockStore.GetKey(calcChunkNumToHash(chunkNum))
	if err != nil {
		return
	}
	chunk := &types.ChunkInfo{}
	err = types.Decode(value, chunk)
	if err != nil {
		return
	}
	var kvs []*types.KeyValue
	for i := chunk.Start; i <= chunk.End; i++ {
		kv, err := chain.deleteBlockBody(i)
		if err != nil {
			continue
		}
		kvs = append(kvs, kv...)
	}
	batch := chain.blockStore.NewBatch(true)
	for _, kv := range kvs {
		if kv.GetValue() == nil {
			batch.Delete(kv.GetKey())
		}
	}
	db.MustWrite(batch)
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

func (chain *BlockChain) IsNeedChunk(height int64) (isNeed bool, chunk *types.ChunkInfo) {
	chunkNum, start, end := chain.CaclChunkInfo(height)
	chunk = &types.ChunkInfo{
		ChunkNum: chunkNum,
		Start:    start,
		End:      end,
	}
	return chain.curChunkNum < chunkNum, chunk
}

func (chain *BlockChain) ShardChunkHandle(chunk *types.ChunkInfo, isNotifyChunk bool) {
	// 1 从block中查询出当前索引档
	// 2 从block中读取当前配置的需要归档大小以及将这些block做hash处理，计算出归档hash，然后本地保存，并且将归档hash广播出去
	// 3
	// getChunkBlocks在blockchain需要在广播的时候主动生成，或者收到通知的时候主动生成
	// 类似与provide时候需要提供两种命令一种是挖矿产生的给对端节点通知哪些节点需要生成，另外一种命令就是平衡的时候给予对端节点实际数据
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
		// 设置该节点有权利触发接下来onceTriggerChunkCount的归档数据
		setTriggerChunkFlag(true)
	}
	// 生成归档记录
	chunkRds := genChunkRecord(chunk, bodys)
	// 将归档记录先保存在本地，归档记录在这里可以不保存或者通过接受广播数据进行归档
	chain.saveChunkRecord(chunkRds)
	// updata chain.curChunkNum
	chain.curChunkNum = chunk.ChunkNum
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
	return
}

// calcChunkHash
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

// 保存归档索引 1:blockhash--->chunkhash 2:blockHeight--->chunkhash
func (chain *BlockChain) saveChunkRecord(chunkRds *types.ChunkRecords) {
	newbatch := chain.blockStore.NewBatch(true)
	for _, kv := range chunkRds.Kvs {
		newbatch.Set(kv.Key, kv.Value)
	}
	db.MustWrite(newbatch)
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
	return
}

func (chain *BlockChain) GetChunkRecord(req *types.ReqChunkRecords) (*types.ChunkRecords, error) {
	rep := &types.ChunkRecords{}
	// TODO req.IsDetail 后面需要再加
	for i := req.Start; i <= req.End; i++ {
		key := calcChunkNumToHash(i)
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

// TODO GetCurRecvChunkNum 后续需要放入结构体中，且在内存中保存一份
func (chain *BlockChain) GetCurRecvChunkNum() int64 {
	return chain.blockStore.getCurChunkNum(RecvChunkNumToHash)
}

// TODO GetCurChunkNum 后续需要放入结构体中，且在内存中保存一份
func (chain *BlockChain) GetCurChunkNum() int64 {
	return chain.blockStore.getCurChunkNum(ChunkNumToHash)
}

func (chain *BlockChain) CaclChunkInfo(height int64) (chunkNum, start, end int64) {
	chunkNum = height / chain.cfg.ChunkblockNum
	start = chunkNum * chain.cfg.ChunkblockNum
	end = start + chain.cfg.ChunkblockNum - 1
	return chunkNum, start, end
}

func setTriggerChunkFlag(flag bool) {
	triggerChunkFlag = flag
}

func getTriggerChunkFlag() bool {
	return triggerChunkFlag
}

func genChunkRecord(chunk *types.ChunkInfo, bodys *types.BlockBodys) *types.ChunkRecords {
	chunkRds := &types.ChunkRecords{}
	for _, body := range bodys.Items {
		chunkRds.Kvs = append(chunkRds.Kvs, &types.KeyValue{Key: calcBlockHashToChunkHash(body.Hash), Value: chunk.ChunkHash})
		chunkRds.Kvs = append(chunkRds.Kvs, &types.KeyValue{Key: calcHeightToChunkHash(body.Height), Value: chunk.ChunkHash})
	}
	chunkRds.Kvs = append(chunkRds.Kvs, &types.KeyValue{Key: calcChunkNumToHash(chunk.ChunkNum), Value: types.Encode(chunk)})
	chunkRds.Kvs = append(chunkRds.Kvs, &types.KeyValue{Key: calcChunkHashToNum(chunk.ChunkHash), Value: types.Encode(chunk)})
	return chunkRds
}