// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/33cn/chain33/common"
	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
)

var (
	//ErrNoBlockToChunk ...
	ErrNoBlockToChunk = errors.New("ErrNoBlockToChunk")
	//ErrNoChunkInfoToDownLoad ...
	ErrNoChunkInfoToDownLoad = errors.New("ErrNoChunkInfoToDownLoad")
)

const (
	// OnceMaxChunkNum 每次检测最大生成chunk数
	OnceMaxChunkNum int32 = 30
	// DelRollbackChunkNum 删除小于当前chunk为DelRollbackChunkNum
	DelRollbackChunkNum int32 = 10
	// MaxReqChunkRecord 每次请求最大MaxReqChunkRecord个chunk的record
	MaxReqChunkRecord int32 = 100
)

func (chain *BlockChain) chunkDeleteRoutine() {
	defer chain.tickerwg.Done()
	// 60s检测一次是否可以删除本地的body数据
	checkDelTicker := time.NewTicker(time.Minute)
	for {
		select {
		case <-chain.quit:
			return
		case <-checkDelTicker.C:
			chain.CheckDeleteBlockBody()
		}
	}
}

func (chain *BlockChain) chunkGenerateRoutine() {
	defer chain.tickerwg.Done()
	// 10s检测一次是否可以触发归档操作
	checkGenChunkTicker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-chain.quit:
			return
		case <-checkGenChunkTicker.C:
			//主动查询当前未归档，然后进行触发
			chain.CheckGenChunkNum()
		}
	}
}

// CheckGenChunkNum 检测是否需要生成chunkNum
func (chain *BlockChain) CheckGenChunkNum() {
	if !atomic.CompareAndSwapInt32(&chain.processingGenChunk, 0, 1) {
		// 保证同一时刻只存在一个该协程
		return
	}
	defer atomic.StoreInt32(&chain.processingGenChunk, 0)
	safetyChunkNum, _, _ := chain.CalcSafetyChunkInfo(chain.GetBlockHeight())
	for i := int32(0); i < OnceMaxChunkNum; i++ {
		chunkNum := chain.getMaxSerialChunkNum() + 1
		if chunkNum > safetyChunkNum {
			break
		}
		if err := chain.chunkShardHandle(chunkNum); err != nil {
			break
		}
		if err := chain.updateMaxSerialChunkNum(chunkNum); err != nil {
			break
		}
	}
}

// CheckDeleteBlockBody 检测是否需要删除已经归档BlockBody
func (chain *BlockChain) CheckDeleteBlockBody() {
	if !atomic.CompareAndSwapInt32(&chain.processingDeleteChunk, 0, 1) {
		// 保证同一时刻只存在一个该协程
		return
	}
	defer atomic.StoreInt32(&chain.processingDeleteChunk, 0)
	const onceDelChunkNum = 100 // 每次walkOverDeleteChunk的最大删除chunk个数
	var count int64
	var kvs []*types.KeyValue
	toDelete := chain.blockStore.GetMaxDeletedChunkNum() + 1
	chainlog.Info("CheckDeleteBlockBody start", "start", toDelete)

	for toDelete+int64(DelRollbackChunkNum) < atomic.LoadInt64(&chain.maxSerialChunkNum) && count < onceDelChunkNum {
		chainlog.Info("CheckDeleteBlockBody toDelete", "toDelete", toDelete)
		kv := chain.DeleteBlockBody(toDelete)
		kvs = append(kvs, kv...)
		toDelete++
		count++
	}
	atomic.AddInt64(&chain.deleteChunkCount, count)
	batch := chain.GetDB().NewBatch(true)
	batch.Reset()
	for _, kv := range kvs {
		batch.Delete(kv.GetKey())
	}
	if count != 0 {
		dbm.MustWrite(batch)
		if err := chain.blockStore.SetMaxDeletedChunkNum(toDelete - 1); err != nil {
			chainlog.Error("CheckDeleteBlockBody", "SetMaxDeletedChunkNum error", err)
		}
	}

	//删除超过100个chunk则进行数据库压缩
	if atomic.LoadInt64(&chain.deleteChunkCount) >= 100 {
		now := time.Now()
		start := []byte("CHAIN-body-body-")
		limit := make([]byte, len(start))
		copy(limit, start)
		limit[len(limit)-1]++
		if err := chain.blockStore.db.CompactRange(start, limit); err != nil {
			chainlog.Error("walkOverDeleteChunk", "CompactRange error", err)
			return
		}
		chainlog.Info("walkOverDeleteChunk", "CompactRange time cost", time.Since(now))
		atomic.StoreInt64(&chain.deleteChunkCount, 0)
	}

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

func (chain *BlockChain) chunkShardHandle(chunkNum int64) error {
	// 1、计算当前chunk信息；
	// 2、生成归档记录；
	// 3、生成辅助删除信息；
	// 4、保存归档记录信息；
	// 5、更新chunk最大连续序列号
	start := chunkNum * chain.cfg.ChunkblockNum
	end := start + chain.cfg.ChunkblockNum - 1
	chunkHash, bodys, err := chain.genChunkBlocks(start, end)
	if err != nil {
		chainlog.Error("chunkShardHandle", "chunkNum", chunkNum, "start", start, "end", end, "err", err)
		return err
	}
	chunk := &types.ChunkInfo{
		ChunkNum:  chunkNum,
		ChunkHash: chunkHash,
		Start:     start,
		End:       end,
	}
	kvs := genChunkRecord(chunk, bodys)
	chain.saveChunkRecord(kvs)
	if err := chain.notifyStoreChunkToP2P(chunk); err != nil {
		return err
	}
	chainlog.Info("chunkShardHandle", "chunkNum", chunk.ChunkNum, "start", start, "end", end, "chunkHash", common.ToHex(chunkHash))
	return nil
}
func (chain *BlockChain) getMaxSerialChunkNum() int64 {
	return atomic.LoadInt64(&chain.maxSerialChunkNum)
}

func (chain *BlockChain) updateMaxSerialChunkNum(chunkNum int64) error {
	err := chain.blockStore.SetMaxSerialChunkNum(chunkNum)
	if err != nil {
		return err
	}
	atomic.StoreInt64(&chain.maxSerialChunkNum, chunkNum)
	return nil
}

func (chain *BlockChain) notifyStoreChunkToP2P(data *types.ChunkInfo) error {
	if chain.client == nil {
		chainlog.Error("notifyStoreChunkToP2P: chain client not bind message queue.")
		return fmt.Errorf("no message queue")
	}

	req := &types.ChunkInfoMsg{
		ChunkHash: data.ChunkHash,
		Start:     data.Start,
		End:       data.End,
	}

	chainlog.Debug("notifyStoreChunkToP2P", "chunknum", data.ChunkNum, "block start height",
		data.Start, "block end height", data.End, "chunk hash", common.ToHex(data.ChunkHash))

	msg := chain.client.NewMessage("p2p", types.EventNotifyStoreChunk, req)
	err := chain.client.Send(msg, true)
	if err != nil {
		return err
	}
	resp, err := chain.client.Wait(msg)
	if err != nil {
		return err
	}

	if data, ok := resp.Data.(*types.Reply); ok && data.IsOk {
		return nil
	}
	return fmt.Errorf("p2p process error")
}

func (chain *BlockChain) genChunkBlocks(start, end int64) ([]byte, *types.BlockBodys, error) {
	var hashs types.ReplyHashes
	var bodys types.BlockBodys
	for i := start; i <= end; i++ {
		detail, err := chain.blockStore.LoadBlock(i, nil)
		if err != nil {
			return nil, nil, err
		}
		body := chain.blockStore.BlockdetailToBlockBody(detail)
		bodys.Items = append(bodys.Items, body)
		hashs.Hashes = append(hashs.Hashes, body.Hash)
	}
	return hashs.Hash(), &bodys, nil
}

func (chain *BlockChain) saveChunkRecord(kvs []*types.KeyValue) {
	chain.blockStore.mustSaveKvset(&types.LocalDBSet{KV: kvs})
}

// GetChunkBlockBody 从localdb本地获取chunkbody
func (chain *BlockChain) GetChunkBlockBody(req *types.ChunkInfoMsg) (*types.BlockBodys, error) {
	if req == nil || req.Start > req.End {
		return nil, types.ErrInvalidParam
	}
	_, bodys, err := chain.genChunkBlocks(req.Start, req.End)
	return bodys, err
}

// AddChunkRecord ...
func (chain *BlockChain) AddChunkRecord(req *types.ChunkRecords) {
	dbset := &types.LocalDBSet{}
	for _, info := range req.Infos {
		dbset.KV = append(dbset.KV, &types.KeyValue{Key: calcRecvChunkNumToHash(info.ChunkNum), Value: types.Encode(info)})
	}
	if len(dbset.KV) > 0 {
		chain.blockStore.mustSaveKvset(dbset)
	}
}

// GetChunkRecord ...
func (chain *BlockChain) GetChunkRecord(req *types.ReqChunkRecords) (*types.ChunkRecords, error) {
	if req.Start > req.End {
		return nil, types.ErrInvalidParam
	}
	rep := &types.ChunkRecords{}
	for i := req.Start; i <= req.End; i++ {
		key := append([]byte{}, calcChunkNumToHash(i)...)
		value, err := chain.blockStore.GetKey(key)
		if err != nil {
			return nil, types.ErrNotFound
		}
		chunk := &types.ChunkInfo{}
		err = types.Decode(value, chunk)
		if err != nil {
			return nil, err
		}
		rep.Infos = append(rep.Infos, chunk)
	}
	if len(rep.Infos) == 0 {
		return nil, types.ErrNotFound
	}
	return rep, nil
}

// GetCurRecvChunkNum ...
func (chain *BlockChain) GetCurRecvChunkNum() int64 {
	return chain.blockStore.getCurChunkNum(RecvChunkNumToHash)
}

// GetCurChunkNum ...
func (chain *BlockChain) GetCurChunkNum() int64 {
	return chain.blockStore.getCurChunkNum(ChunkNumToHash)
}

// CalcSafetyChunkInfo 计算安全的chunkNum用于生成chunk时候或者删除时候
func (chain *BlockChain) CalcSafetyChunkInfo(height int64) (chunkNum, start, end int64) {
	height = chain.calcSafetyChunkHeight(height)
	if height < 0 {
		return -1, 0, 0
	}
	return calcChunkInfo(chain.cfg, height)
}

func (chain *BlockChain) calcSafetyChunkHeight(height int64) int64 {
	return height - MaxRollBlockNum - chain.cfg.ChunkblockNum
}

// CalcChunkInfo 主要用于计算验证
func (chain *BlockChain) CalcChunkInfo(height int64) (chunkNum, start, end int64) {
	return calcChunkInfo(chain.cfg, height)
}

func calcChunkInfo(cfg *types.BlockChain, height int64) (chunkNum, start, end int64) {
	chunkNum = height / cfg.ChunkblockNum
	start = chunkNum * cfg.ChunkblockNum
	end = start + cfg.ChunkblockNum - 1
	return chunkNum, start, end
}

// genChunkRecord 生成归档索引 1:blockhash--->chunkhash 2:blockHeight--->chunkhash
func genChunkRecord(chunk *types.ChunkInfo, bodys *types.BlockBodys) []*types.KeyValue {
	var kvs []*types.KeyValue
	kvs = append(kvs, &types.KeyValue{Key: calcChunkNumToHash(chunk.ChunkNum), Value: types.Encode(chunk)})
	kvs = append(kvs, &types.KeyValue{Key: calcChunkHashToNum(chunk.ChunkHash), Value: types.Encode(chunk)})
	return kvs
}
