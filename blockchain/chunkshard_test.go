// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/mock"

	dbm "github.com/33cn/chain33/common/db"
	qmocks "github.com/33cn/chain33/queue/mocks"
	"github.com/stretchr/testify/assert"
)

func TestCheckGenChunkNum(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录

	chain := InitEnv()
	blockStoreDB := dbm.NewDB("blockchain", "leveldb", dir, 100)
	blockStore := NewBlockStore(chain, blockStoreDB, chain.client)
	assert.NotNil(t, blockStore)
	chain.blockStore = blockStore
	// mock client
	client := &qmocks.Client{}
	chain.client = client
	data := &types.ChunkInfo{}
	client.On("NewMessage", mock.Anything, mock.Anything, mock.Anything).Return(&queue.Message{Data: data})
	client.On("Send", mock.Anything, mock.Anything).Return(nil)
	rspMsg := &queue.Message{Data: &types.BlockBodys{Items: []*types.BlockBody{{}, {}}}}
	client.On("Wait", mock.Anything).Return(rspMsg, nil)
	// set config
	chain.cfg.ChunkblockNum = 5

	start := int64(0)
	end := int64(150)
	saveBlockToDB(chain, start, end)
	//just for test
	chain.blockStore.UpdateHeight2(MaxRollBlockNum + 150)
	// check
	lastChunkNum := int64(0)
	for i := 0; i < 5; i++ {
		chain.CheckGenChunkNum()
		// check
		serChunkNum := chain.getMaxSerialChunkNum()
		curChunkNum := chain.GetCurChunkNum()
		assert.Equal(t, serChunkNum, curChunkNum)
		if i >= 3 {
			assert.Equal(t, lastChunkNum, curChunkNum)
		} else {
			assert.NotEqualf(t, lastChunkNum, curChunkNum, "not equal")
		}
		lastChunkNum = curChunkNum
	}
}

func TestDeleteBlockBody(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录

	chain := InitEnv()
	blockStoreDB := dbm.NewDB("blockchain", "leveldb", dir, 100)
	blockStore := NewBlockStore(chain, blockStoreDB, chain.client)
	assert.NotNil(t, blockStore)
	chain.blockStore = blockStore
	start := int64(0)
	end := int64(15)
	saveBlockToDB(chain, start, end)
	var hashs [][]byte
	for i := start; i <= end; i++ {
		head, err := chain.blockStore.loadHeaderByIndex(i)
		assert.NoError(t, err)
		hashs = append(hashs, head.Hash)
	}
	blockStore.Set(calcChunkNumToHash(1), types.Encode(&types.ChunkInfo{Start: 0, End: 10}))
	kvs := chain.DeleteBlockBody(1)
	chain.blockStore.mustSaveKvset(&types.LocalDBSet{KV: kvs})
	for i := start; i <= 10; i++ {
		_, err = getBodyByIndex(blockStore.db, "", calcHeightHashKey(i, hashs[int(i)]), nil)
		assert.Error(t, err, types.ErrNotFound)
	}
	for i := 11; i <= 15; i++ {
		_, err = getBodyByIndex(blockStore.db, "", calcHeightHashKey(int64(i), hashs[i]), nil)
		assert.NoError(t, err)
	}
}

func TestIsNeedChunk(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录

	chain := InitEnv()
	blockStoreDB := dbm.NewDB("blockchain", "leveldb", dir, 100)
	blockStore := NewBlockStore(chain, blockStoreDB, chain.client)
	assert.NotNil(t, blockStore)
	chain.blockStore = blockStore
	chain.cfg.ChunkblockNum = 2
	setChunkInfo := &types.ChunkInfo{
		ChunkNum: 6,
	}
	blockStore.Set(calcChunkNumToHash(6), types.Encode(setChunkInfo))
	// check
	// 当前数据库中最大chunNum=6 高度为3的区块计算的chunkNum为1
	baseNum := MaxRollBlockNum + chain.cfg.ChunkblockNum
	isNeed, chunk := chain.IsNeedChunk(baseNum + 3)
	assert.Equal(t, isNeed, false)
	assert.Equal(t, chunk.Start, int64(2))
	assert.Equal(t, chunk.End, int64(3))
	// 当前数据库中最大chunNum=6 高度为12的区块计算的chunkNum为6
	isNeed, chunk = chain.IsNeedChunk(baseNum + 12)
	assert.Equal(t, isNeed, false)
	assert.Equal(t, chunk.Start, int64(12))
	assert.Equal(t, chunk.End, int64(13))
	// 当前数据库中最大chunNum=6 高度为13的区块计算的chunkNum为6
	isNeed, chunk = chain.IsNeedChunk(baseNum + 13)
	assert.Equal(t, isNeed, false)
	assert.Equal(t, chunk.Start, int64(12))
	assert.Equal(t, chunk.End, int64(13))
	// 当前数据库中最大chunNum=6 高度为14的区块计算的chunkNum为7
	isNeed, chunk = chain.IsNeedChunk(baseNum + 14)
	assert.Equal(t, isNeed, true)
	assert.Equal(t, chunk.Start, int64(14))
	assert.Equal(t, chunk.End, int64(15))
}

func TestGenDeleteChunkSign(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录

	chain := InitEnv()
	blockStoreDB := dbm.NewDB("blockchain", "leveldb", dir, 100)
	blockStore := NewBlockStore(chain, blockStoreDB, chain.client)
	assert.NotNil(t, blockStore)
	chain.blockStore = blockStore

	blockStore.UpdateHeight2(10)
	kv := chain.genDeleteChunkSign(1)
	fn := func(value []byte) int64 {
		data := &types.Int64{}
		err = types.Decode(value, data)
		assert.NoError(t, err)
		return data.Data
	}
	// test for no GetPeerMaxBlkHeight
	assert.Equal(t, int64(-1), fn(kv.Value))
	// test 可以获取到最大peer节点，但是比本地高度低的情况
	chain.peerList = PeerInfoList{
		&PeerInfo{
			Height: 9,
		},
	}
	kv = chain.genDeleteChunkSign(1)
	assert.Equal(t, int64(10), fn(kv.Value))
	// test 可以获取到最大peer节点，但是比本地高度高的情况
	chain.peerList[0].Height = 15
	kv = chain.genDeleteChunkSign(1)
	assert.Equal(t, int64(15), fn(kv.Value))
}

func TestMaxSerialChunkNum(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录

	chain := InitEnv()
	blockStoreDB := dbm.NewDB("blockchain", "leveldb", dir, 100)
	blockStore := NewBlockStore(chain, blockStoreDB, chain.client)
	assert.NotNil(t, blockStore)
	chain.blockStore = blockStore
	// test noerror
	end := 100
	for i := 0; i < 100; i++ {
		err = chain.updateMaxSerialChunkNum(int64(i))
		assert.NoError(t, err)
		chunkNum := chain.getMaxSerialChunkNum()
		assert.Equal(t, int64(i), chunkNum)
	}
	// test error
	err = chain.updateMaxSerialChunkNum(int64(end + 5))
	assert.Error(t, err, ErrNoChunkNumSerial)
}

func TestNotifyStoreChunkToP2P(t *testing.T) {
	client := &qmocks.Client{}
	chain := BlockChain{client: client}
	data := &types.ChunkInfo{
		ChunkNum:  1,
		ChunkHash: []byte("1111111111111"),
		Start:     1,
		End:       2,
	}
	client.On("NewMessage", mock.Anything, mock.Anything, mock.Anything).Return(&queue.Message{Data: data})
	client.On("Send", mock.Anything, mock.Anything).Return(nil)
	//rspMsg := &queue.Message{Data: &types.BlockBodys{Items: []*types.BlockBody{{}, {}}}}
	//client.On("Wait", mock.Anything).Return(rspMsg, nil)
	chain.notifyStoreChunkToP2P(data)
}

func TestGenChunkBlocks(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录

	chain := InitEnv()
	blockStoreDB := dbm.NewDB("blockchain", "leveldb", dir, 100)
	blockStore := NewBlockStore(chain, blockStoreDB, chain.client)
	assert.NotNil(t, blockStore)
	chain.blockStore = blockStore
	start := int64(0)
	end := int64(10)
	saveBlockToDB(chain, start, end)
	chunkHash, bodys, err := chain.genChunkBlocks(start, end)
	assert.NoError(t, err)
	assert.NotNil(t, chunkHash)
	assert.Equal(t, int(end-start+1), len(bodys.Items))
	// for error
	end = int64(11)
	chunkHash, bodys, err = chain.genChunkBlocks(start, end)
	assert.Nil(t, chunkHash)
	assert.Nil(t, bodys)
	assert.Error(t, err, types.ErrHashNotExist)
}

func TestGetChunkBlockBody(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录

	chain := InitEnv()
	blockStoreDB := dbm.NewDB("blockchain", "leveldb", dir, 100)
	blockStore := NewBlockStore(chain, blockStoreDB, chain.client)
	assert.NotNil(t, blockStore)
	chain.blockStore = blockStore
	req := &types.ChunkInfoMsg{
		ChunkHash: nil,
		Start:     2,
		End:       0,
	}
	body, err := chain.GetChunkBlockBody(req)
	assert.Error(t, err, types.ErrInvalidParam)
	assert.Nil(t, body)
}

func TestGetChunkRecord(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录

	chain := InitEnv()
	blockStoreDB := dbm.NewDB("blockchain", "leveldb", dir, 100)
	blockStore := NewBlockStore(chain, blockStoreDB, chain.client)
	assert.NotNil(t, blockStore)
	chain.blockStore = blockStore
	chunk := &types.ChunkInfo{
		ChunkHash: []byte("11111111111"),
	}
	for i := 0; i < 5; i++ {
		chunk.ChunkNum = int64(i)
		blockStore.Set(calcChunkNumToHash(int64(i)), types.Encode(chunk))
	}
	req := &types.ReqChunkRecords{
		Start:    2,
		End:      1,
		IsDetail: false,
		Pid:      nil,
	}
	record, err := chain.GetChunkRecord(req)
	assert.Error(t, err, types.ErrInvalidParam)
	assert.Nil(t, record)
	req.Start = 0
	req.End = 0
	record, err = chain.GetChunkRecord(req)
	assert.NoError(t, err)
	assert.Equal(t, len(record.Infos), 1)
	req.Start = 0
	req.End = 4
	record, err = chain.GetChunkRecord(req)
	assert.NoError(t, err)
	assert.Equal(t, len(record.Infos), 5)
	for i, info := range record.Infos {
		assert.Equal(t, int64(i), info.ChunkNum)
	}
	req.End = 5
	record, err = chain.GetChunkRecord(req)
	assert.Error(t, err, types.ErrNotFound)
}

func TestCaclChunkInfo(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录

	chain := InitEnv()
	cfg := chain.client.GetConfig()
	blockStoreDB := dbm.NewDB("blockchain", "leveldb", dir, 100)
	blockStore := NewBlockStore(chain, blockStoreDB, chain.client)
	assert.NotNil(t, blockStore)
	chain.blockStore = blockStore

	chainCfg := cfg.GetModuleConfig().BlockChain
	chainCfg.ChunkblockNum = 1
	chunkNum, start, end := chain.CalcChunkInfo(0)
	assert.Equal(t, chunkNum, int64(0))
	assert.Equal(t, start, int64(0))
	assert.Equal(t, end, int64(0))

	chunkNum, start, end = chain.CalcChunkInfo(1)
	assert.Equal(t, chunkNum, int64(1))
	assert.Equal(t, start, int64(1))
	assert.Equal(t, end, int64(1))

	chainCfg.ChunkblockNum = 2
	chunkNum, start, end = chain.CalcChunkInfo(0)
	assert.Equal(t, chunkNum, int64(0))
	assert.Equal(t, start, int64(0))
	assert.Equal(t, end, int64(1))
	chunkNum, start, end = chain.CalcChunkInfo(2)
	assert.Equal(t, chunkNum, int64(1))
	assert.Equal(t, start, int64(2))
	assert.Equal(t, end, int64(3))
	chunkNum, start, end = chain.CalcChunkInfo(3)
	assert.Equal(t, chunkNum, int64(1))
	assert.Equal(t, start, int64(2))
	assert.Equal(t, end, int64(3))
	chunkNum, start, end = chain.CalcChunkInfo(4)
	assert.Equal(t, chunkNum, int64(2))
	assert.Equal(t, start, int64(4))
	assert.Equal(t, end, int64(5))
}

func TestGenChunkRecord(t *testing.T) {
	chunk := &types.ChunkInfo{
		ChunkNum:  1,
		ChunkHash: []byte("111111111111111111111"),
		Start:     1,
		End:       10,
	}
	bodys := &types.BlockBodys{
		Items: []*types.BlockBody{
			{Hash: []byte("123")},
			{Hash: []byte("456")},
		},
	}
	kvs := genChunkRecord(chunk, bodys)
	assert.Equal(t, 4, len(kvs))
	assert.Equal(t, kvs[0].Key, calcBlockHashToChunkHash([]byte("123")))
	assert.Equal(t, kvs[0].Value, chunk.ChunkHash)
	assert.Equal(t, kvs[1].Key, calcBlockHashToChunkHash([]byte("456")))
	assert.Equal(t, kvs[1].Value, chunk.ChunkHash)

	assert.Equal(t, kvs[2].Key, calcChunkNumToHash(1))
	assert.Equal(t, kvs[2].Value, types.Encode(chunk))
	assert.Equal(t, kvs[3].Key, calcChunkHashToNum(chunk.ChunkHash))
	assert.Equal(t, kvs[3].Value, types.Encode(chunk))
}

func TestFetchChunkBlock(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录

	chain := InitEnv()
	blockStoreDB := dbm.NewDB("blockchain", "leveldb", dir, 100)
	blockStore := NewBlockStore(chain, blockStoreDB, chain.client)
	assert.NotNil(t, blockStore)
	chain.blockStore = blockStore
	// mock client
	client := &qmocks.Client{}
	chain.client = client
	data := &types.ChunkInfo{}
	client.On("NewMessage", mock.Anything, mock.Anything, mock.Anything).Return(&queue.Message{Data: data})
	client.On("Send", mock.Anything, mock.Anything).Return(nil)
	rspMsg := &queue.Message{Data: &types.BlockBodys{Items: []*types.BlockBody{{}, {}}}}
	client.On("Wait", mock.Anything).Return(rspMsg, nil)
	// set config
	chain.cfg.ChunkblockNum = 5
	start := int64(0)
	end := int64(51)
	chain.InitDownLoadInfo(start, end, []string{"1", "2"})
	// set RecvChunkHash
	for i := start; i <= end; i++ {
		chain.blockStore.Set(calcRecvChunkNumToHash(i), types.Encode(&types.ChunkInfo{ChunkHash: []byte("hash")}))
	}
	// check for updata
	go func() {
		for i := int64(0); i <= end; i++ {
			time.Sleep(time.Microsecond * 500)
			chain.downLoadTask.Done(i)
			fmt.Println("done i", i)
		}
	}()
	chain.ReqDownLoadChunkBlocks()
	time.Sleep(time.Second)
}

func TestFetchChunkRecords(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录

	chain := InitEnv()
	blockStoreDB := dbm.NewDB("blockchain", "leveldb", dir, 100)
	blockStore := NewBlockStore(chain, blockStoreDB, chain.client)
	assert.NotNil(t, blockStore)
	chain.blockStore = blockStore
	// mock client
	client := &qmocks.Client{}
	chain.client = client
	data := &types.ChunkInfo{}
	client.On("NewMessage", mock.Anything, mock.Anything, mock.Anything).Return(&queue.Message{Data: data})
	client.On("Send", mock.Anything, mock.Anything).Return(nil)
	rspMsg := &queue.Message{Data: &types.BlockBodys{Items: []*types.BlockBody{{}, {}}}}
	client.On("Wait", mock.Anything).Return(rspMsg, nil)

	// set config
	chain.cfg.ChunkblockNum = 5
	// 设置最大对端节点高度
	peerInfo := &PeerInfo{
		Name:   "123",
		Height: 9,
	}
	chain.peerList = PeerInfoList{peerInfo}
	chain.bestChainPeerList["123"] = &BestPeerInfo{Peer: peerInfo, IsBestChain: true}

	// case 1 peerMaxBlkHeight < curheight
	chain.blockStore.UpdateHeight2(MaxRollBlockNum + 100)
	chain.ChunkRecordSync()
	// case 2 peerMaxBlkHeight - MaxRollBlockNum > curheight
	// 设置从0开始
	end := MaxRollBlockNum + 6000
	chain.blockStore.UpdateHeight2(-1)
	chain.peerList[0].Height = end
	// check for updata
	go func() {
		count := end / chain.cfg.ChunkblockNum / int64(MaxReqChunkRecord)
		for i := int64(0); i <= count; i++ {
			time.Sleep(time.Microsecond * 200)
			for j := i * int64(MaxReqChunkRecord); j < (i+1)*int64(MaxReqChunkRecord); j++ {
				chain.blockStore.Set(calcRecvChunkNumToHash(j), types.Encode(&types.ChunkInfo{ChunkHash: []byte("hash")}))
			}
			chain.chunkRecordTask.Done(i)
			fmt.Println("done i", i)
		}
	}()
	chain.ChunkRecordSync()
	time.Sleep(time.Second)
}

func saveBlockToDB(chain *BlockChain, start, end int64) {
	batch := chain.blockStore.NewBatch(true)
	for i := start; i <= end; i++ {
		blockdetail := &types.BlockDetail{
			Block: &types.Block{
				Version: 0,
				Height:  i,
			},
		}
		batch.Reset()
		chain.blockStore.SaveBlock(batch, blockdetail, i)
		batch.Write()
		chain.blockStore.UpdateHeight2(i)
	}
}
