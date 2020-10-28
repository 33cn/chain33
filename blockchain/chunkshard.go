// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
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

func (chain *BlockChain) chunkProcessRoutine() {
	defer chain.tickerwg.Done()

	// 1.60s检测一次是否可以删除本地的body数据
	// 2.60s检测一次是否可以触发归档操作

	checkDelTicker := time.NewTicker(60 * time.Second)
	checkGenChunkTicker := time.NewTicker(30 * time.Second) //主动查询当前未归档，然后进行触发
	for {
		select {
		case <-chain.quit:
			return
		case <-checkDelTicker.C:
			// 6.5版本先不做删除
			chain.CheckDeleteBlockBody()
		case <-checkGenChunkTicker.C:
			chain.CheckGenChunkNum()
		}
	}
}

// CheckGenChunkNum 检测是否需要生成chunkNum
func (chain *BlockChain) CheckGenChunkNum() {
	safetyChunkNum, _, _ := chain.CalcSafetyChunkInfo(chain.GetBlockHeight())
	for i := int32(0); i < OnceMaxChunkNum; i++ {
		chunkNum := chain.getMaxSerialChunkNum() + 1
		if chunkNum > safetyChunkNum {
			break
		}
		if err := chain.chunkShardHandle(chunkNum); err != nil {
			break
		}
		if err := chain.updateMaxSerialChunkNum(); err != nil {
			break
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
	chain.walkOverDeleteChunk(maxHeight)
}

// 通过遍历ToDeleteChunkSign 去删除归档body
func (chain *BlockChain) walkOverDeleteChunk(maxHeight int64) {
	db := chain.GetDB()
	it := db.Iterator(ToDeleteChunkSign, nil, false)
	defer it.Close()
	// 每次walkOverDeleteChunk的最大删除chunk个数
	const onceDelChunkNum = 100
	batch := db.NewBatch(true)
	count := 0
	for it.Rewind(); it.Valid(); it.Next() {
		value := it.Value()
		data := &types.Int64{}
		err := types.Decode(value, data)
		if err != nil {
			continue
		}
		chunkNum, err := strconv.ParseInt(string(bytes.TrimPrefix(it.Key(), ToDeleteChunkSign)), 10, 64)
		if err != nil {
			continue
		}
		if chunkNum > atomic.LoadInt64(&chain.maxSerialChunkNum) {
			break
		}
		key := make([]byte, len(it.Key()))
		copy(key, it.Key())
		var kvs []*types.KeyValue
		if data.Data < 0 {
			data.Data = maxHeight
			kvs = append(kvs, &types.KeyValue{Key: key, Value: types.Encode(data)})
		} else {
			delChunkNum, _, _ := chain.CalcChunkInfo(data.Data)
			maxChunkNum, _, _ := chain.CalcSafetyChunkInfo(maxHeight)
			if maxChunkNum > delChunkNum+int64(DelRollbackChunkNum) {
				kvs = append(kvs, &types.KeyValue{Key: key, Value: nil}) // 将相应的ToDeleteChunkSign进行删除
				kv := chain.DeleteBlockBody(chunkNum)
				if len(kv) > 0 {
					kvs = append(kvs, kv...)
				}
			}
		}
		// 批量写入数据库
		if len(kvs) > 0 {
			batch.Reset()
			for _, kv := range kvs {
				if kv.GetValue() == nil {
					batch.Delete(kv.GetKey())
				} else {
					batch.Set(kv.GetKey(), kv.GetValue())
				}
			}
			dbm.MustWrite(batch)
		}
		count++
		if count > onceDelChunkNum {
			break
		}
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
	kvs = append(kvs, chain.genDeleteChunkSign(chunk.ChunkNum))
	chain.saveChunkRecord(kvs)
	if err := chain.notifyStoreChunkToP2P(chunk); err != nil {
		return err
	}
	chainlog.Info("chunkShardHandle", "chunkNum", chunk.ChunkNum, "start", start, "end", end, "chunkHash", common.ToHex(chunkHash))
	return nil
}

func (chain *BlockChain) genDeleteChunkSign(chunkNum int64) *types.KeyValue {
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
	return atomic.LoadInt64(&chain.maxSerialChunkNum)
}

func (chain *BlockChain) updateMaxSerialChunkNum() error {
	err := chain.blockStore.SetMaxSerialChunkNum(atomic.LoadInt64(&chain.maxSerialChunkNum) + 1)
	if err != nil {
		return err
	}
	atomic.AddInt64(&chain.maxSerialChunkNum, 1)
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
	for _, body := range bodys.Items {
		kvs = append(kvs, &types.KeyValue{Key: calcBlockHashToChunkHash(body.Hash), Value: chunk.ChunkHash})
	}
	kvs = append(kvs, &types.KeyValue{Key: calcChunkNumToHash(chunk.ChunkNum), Value: types.Encode(chunk)})
	kvs = append(kvs, &types.KeyValue{Key: calcChunkHashToNum(chunk.ChunkHash), Value: types.Encode(chunk)})
	return kvs
}
