// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"bytes"
	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/queue"
	"strconv"
	"time"

	"github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
)


func (chain *BlockChain) DeleteHaveShardData() {
	// 30s检测一次 需要删除本地的body数据 这里body数据的索引还有联系需要在处理下
	checkTicker := time.NewTicker(30 * time.Second)
	for {
		select {
		case <-chain.quit:
			return
		case <-checkTicker.C:
			height := chain.GetBlockHeight()
			chunkHeight := chain.blockStore.getCurChunkHeight()
			chunkNum := (height - MaxRollBlockNum)/chain.cfg.ChunkblockNum
			if chunkNum > chunkHeight + 1 {

			}
		}
	}
}

func (chain *BlockChain) deleteShardBody() {
	chain.blockStore.GetKey(calcChunkNumToHash(chunkHeight))
}


func (chain *BlockChain) ShardDataHandle(height int64) {
	// 1 从block中查询出当前索引档
	// 2 从block中读取当前配置的需要归档大小以及将这些block做hash处理，计算出归档hash，然后本地保存，并且将归档hash广播出去
	// 3
	// getChunkBlocks在blockchain需要在广播的时候主动生成，或者收到通知的时候主动生成
	// 类似与provide时候需要提供两种命令一种是挖矿产生的给对端节点通知哪些节点需要生成，另外一种命令就是平衡的时候给予对端节点实际数据
	chunkNum := (height - MaxRollBlockNum)/chain.cfg.ChunkblockNum
	_, err := chain.blockStore.GetKey(calcHeightToChunkHash(chunkNum))
	if err == nil {
		//说明已经做过归档
		storeLog.Error("ShardDataHandle", "had chunkNum", chunkNum)
		return
	}
	start := chunkNum * chain.cfg.ChunkblockNum
	end   := start + chain.cfg.ChunkblockNum - 1
	chunkHash, bodys, err := chain.getChunkBlocks(start, end)
	if err != nil {
		storeLog.Error("ShardDataHandle", "chunkNum", chunkNum, "start", start, "end", end, "err", err)
		return
	}
	// TODO 归档数据失败的话可以等到下次在发送，每次挖矿节点需要检查一下上次需要归档的数据是否已经发出去
	chain.storeChunkToP2Pstore(chunkHash, bodys)
	// 生成归档记录
	reqBlock := &types.ReqChunkBlock{
		ChunkHash: chunkHash,
		Start: start,
		End: end,
	}
	chunkRds := genChunkRecord(chunkNum, reqBlock, bodys)
	// 将归档记录先保存在本地，归档记录在这里可以不保存或者通过接受广播数据进行归档
	chain.saveChunkRecord(chunkRds)
}

// ShardDataHandle 分片数据处理
// 由外部网络触发进行归档
func (chain *BlockChain) triggerShardDataHandle(chunkNum int64, chunkHash []byte) {
	// 1 从block中查询出当前索引档
	// 2 从block中读取当前配置的需要归档大小以及将这些block做hash处理，计算出归档hash，然后本地保存，并且将归档hash广播出去
	// 3
	// getChunkBlocks在blockchain需要在广播的时候主动生成，或者收到通知的时候主动生成
	// 类似与provide时候需要提供两种命令一种是挖矿产生的给对端节点通知哪些节点需要生成，另外一种命令就是平衡的时候给予对端节点实际数据
	value, err := chain.blockStore.GetKey(calcHeightToChunkHash(chunkNum))
	if err == nil && bytes.Equal(chunkHash, value) {
		//说明已经做过归档
		storeLog.Error("ShardDataHandle", "had chunkNum", chunkNum)
		return
	}
	start := chunkNum * chain.cfg.ChunkblockNum
	end   := start + chain.cfg.ChunkblockNum - 1
	chunkHash, bodys, err := chain.getChunkBlocks(start, end)
	if err != nil {
		storeLog.Error("ShardDataHandle", "chunkNum", chunkNum, "start", start, "end", end, "err", err)
		return
	}
	// TODO 触发发送可以归档数据失败的话可以等到下次在发送，每次挖矿节点需要检查一下上次需要归档的数据是否已经发出去
	chain.storeChunkToP2Pstore(chunkHash, bodys)
	// 生成归档记录
	reqBlock := &types.ReqChunkBlock{
		ChunkHash: chunkHash,
		Start: start,
		End: end,
	}
	chunkRds := genChunkRecord(chunkNum, reqBlock, bodys)
	// 将归档记录先保存在本地
	chain.saveChunkRecord(chunkRds)
}

//SendChunkRecordBroadcast blockchain模块广播ChunkRecords到网络中
//func (chain *BlockChain) SendChunkRecordBroadcast(chunkRds *types.ChunkRecords, chunkNum int64) {
//	if chain.client == nil {
//		chainlog.Error("SendChunkRecordBroadcast: chain client not bind message queue.")
//		return
//	}
//	if chunkRds == nil {
//		chainlog.Error("SendChunkRecordBroadcast chunkRds is null")
//		return
//	}
//	chainlog.Debug("SendChunkRecordBroadcast", "chunkNum", chunkNum)
//
//	msg := chain.client.NewMessage("p2p", types.EventChunkRecordBroadcast, chunkRds)
//	err := chain.client.Send(msg, true)
//	if err != nil {
//		chainlog.Error("SendChunkRecordBroadcast", "chunkNum", chunkNum, "err", err)
//	}
//	_, err = chain.client.Wait(msg)
//	if err != nil {
//		synlog.Error("SendChunkRecordBroadcast", "client.Wait err:", err)
//		return
//	}
//	return
//}

func (chain *BlockChain) storeChunkToP2Pstore(chunkHash []byte, data *types.BlockBodys) {
	if chain.client == nil {
		chainlog.Error("storeChunkToP2Pstore: chain client not bind message queue.")
		return
	}

	chainlog.Debug("storeChunkToP2Pstore", "chunk block num", len(data.Items), "chunk hash", common.ToHex(chunkHash))

	kv := &types.KeyValue{
		Key: chunkHash,
		Value: types.Encode(data),
	}

	msg := chain.client.NewMessage("p2p", types.EventNotifyStoreChunk, kv)
	err := chain.client.Send(msg, true)
	if err != nil {
		chainlog.Error("storeChunkToP2Pstore", "chunk block num", len(data.Items), "chunk hash", common.ToHex(chunkHash), "err", err)
	}
	_, err = chain.client.Wait(msg)
	if err != nil {
		synlog.Error("storeChunkToP2Pstore", "client.Wait err:", err)
		return
	}
	return
}

// calcChunkHash
func (chain *BlockChain) getChunkBlocks(start, end int64) ([]byte, *types.BlockBodys, error){
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

// 从P2P网络中获取Chunk blcoks数据，主要用于区块链同步
func (chain *BlockChain) GetChunkBlocks(reqBlock *types.ReqChunkBlock) ([]*types.Block, error) {
	if reqBlock == nil {
		return nil, types.ErrInvalidParam
	}
	// 先在localdb中获取区块头


	return nil
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
			continue
		}
		rep.Kvs = append(rep.Kvs, &types.KeyValue{Key: key, Value: value})
	}
	if len(rep.Kvs) == 0 {
		return rep, types.ErrNotFound
	}
	return rep, nil
}

func (chain *BlockChain) GetCurRecvChunkNum() int64 {
	return chain.blockStore.getCurChunkNum(RecvChunkNumToHash)
}

func (chain *BlockChain) GetCurChunkNum() int64 {
	return chain.blockStore.getCurChunkNum(ChunkNumToHash)
}

func (chain *BlockChain) CaclChunkNum(height int64) int64 {
	return (height - MaxRollBlockNum)/chain.cfg.ChunkblockNum
}

func genChunkRecord(chunkNum int64, chunk *types.ReqChunkBlock, bodys *types.BlockBodys) *types.ChunkRecords {
	chunkRds := &types.ChunkRecords{}
	for _, body := range bodys.Items {
		chunkRds.Kvs = append(chunkRds.Kvs, &types.KeyValue{Key: calcBlockHashToChunkHash(body.Hash), Value: chunk.ChunkHash})
		chunkRds.Kvs = append(chunkRds.Kvs, &types.KeyValue{Key: calcHeightToChunkHash(body.Height), Value: chunk.ChunkHash})
	}
	chunkRds.Kvs = append(chunkRds.Kvs, &types.KeyValue{Key: calcChunkNumToHash(chunkNum), Value: types.Encode(chunk)})
	return chunkRds
}



