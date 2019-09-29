package blockchain

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/33cn/chain33/common"
	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func InitEnv() *BlockChain {
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	q := queue.New("channel")
	q.SetConfig(cfg)
	chain := New(cfg)
	chain.client = q.Client()
	return chain
}

func TestGetStoreUpgradeMeta(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录

	blockStoreDB := dbm.NewDB("blockchain", "leveldb", dir, 100)

	chain := InitEnv()
	blockStore := NewBlockStore(chain, blockStoreDB, nil)
	require.NotNil(t, blockStore)

	meta, err := blockStore.GetStoreUpgradeMeta()
	require.NoError(t, err)
	require.Equal(t, meta.Version, "0.0.0")

	meta.Version = "1.0.0"
	err = blockStore.SetStoreUpgradeMeta(meta)
	require.NoError(t, err)
	meta, err = blockStore.GetStoreUpgradeMeta()
	require.NoError(t, err)
	require.Equal(t, meta.Version, "1.0.0")
}

func TestSeqSaveAndGet(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录

	blockStoreDB := dbm.NewDB("blockchain", "leveldb", dir, 100)

	chain := InitEnv()
	blockStore := NewBlockStore(chain, blockStoreDB, nil)
	assert.NotNil(t, blockStore)
	blockStore.saveSequence = true
	blockStore.isParaChain = false

	newBatch := blockStore.NewBatch(true)
	seq, err := blockStore.saveBlockSequence(newBatch, []byte("s0"), 0, 1, 0)
	assert.Nil(t, err)
	assert.Equal(t, int64(0), seq)
	err = newBatch.Write()
	assert.Nil(t, err)

	newBatch = blockStore.NewBatch(true)
	seq, err = blockStore.saveBlockSequence(newBatch, []byte("s1"), 1, 1, 0)
	assert.Nil(t, err)
	assert.Equal(t, int64(1), seq)
	err = newBatch.Write()
	assert.Nil(t, err)

	s, err := blockStore.LoadBlockLastSequence()
	assert.Nil(t, err)
	assert.Equal(t, int64(1), s)

	s2, err := blockStore.GetBlockSequence(s)
	assert.Nil(t, err)
	assert.Equal(t, []byte("s1"), s2.Hash)

	s3, err := blockStore.GetSequenceByHash([]byte("s1"))
	assert.Nil(t, err)
	assert.Equal(t, int64(1), s3)
}

func TestParaSeqSaveAndGet(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录

	blockStoreDB := dbm.NewDB("blockchain", "leveldb", dir, 100)

	bchain := InitEnv()
	blockStore := NewBlockStore(bchain, blockStoreDB, nil)
	assert.NotNil(t, blockStore)
	blockStore.saveSequence = true
	blockStore.isParaChain = true

	newBatch := blockStore.NewBatch(true)
	seq, err := blockStore.saveBlockSequence(newBatch, []byte("s0"), 0, 1, 1)
	assert.Nil(t, err)
	assert.Equal(t, int64(0), seq)
	err = newBatch.Write()
	assert.Nil(t, err)

	newBatch = blockStore.NewBatch(true)
	seq, err = blockStore.saveBlockSequence(newBatch, []byte("s1"), 1, 1, 10)
	assert.Nil(t, err)
	assert.Equal(t, int64(1), seq)
	err = newBatch.Write()
	assert.Nil(t, err)

	s, err := blockStore.LoadBlockLastSequence()
	assert.Nil(t, err)
	assert.Equal(t, int64(1), s)

	s2, err := blockStore.GetBlockSequence(s)
	assert.Nil(t, err)
	assert.Equal(t, []byte("s1"), s2.Hash)

	s3, err := blockStore.GetSequenceByHash([]byte("s1"))
	assert.Nil(t, err)
	assert.Equal(t, int64(1), s3)

	s4, err := blockStore.GetMainSequenceByHash([]byte("s1"))
	assert.Nil(t, err)
	assert.Equal(t, int64(10), s4)

	s5, err := blockStore.LoadBlockLastMainSequence()
	assert.Nil(t, err)
	assert.Equal(t, int64(10), s5)

	s6, err := blockStore.GetBlockByMainSequence(1)
	assert.Nil(t, err)
	assert.Equal(t, []byte("s0"), s6.Hash)

	chain := &BlockChain{
		blockStore: blockStore,
	}
	s7, err := chain.ProcGetMainSeqByHash([]byte("s0"))
	assert.Nil(t, err)
	assert.Equal(t, int64(1), s7)

	_, err = chain.ProcGetMainSeqByHash([]byte("s0-not-exist"))
	assert.NotNil(t, err)
}

func TestSeqCreateAndDelete(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录

	blockStoreDB := dbm.NewDB("blockchain", "leveldb", dir, 100)

	chain := InitEnv()
	blockStore := NewBlockStore(chain, blockStoreDB, nil)
	assert.NotNil(t, blockStore)
	blockStore.saveSequence = false
	blockStore.isParaChain = true

	batch := blockStore.NewBatch(true)
	for i := 0; i <= 100; i++ {
		var header types.Header
		h0 := calcHeightToBlockHeaderKey(int64(i))
		header.Hash = []byte(fmt.Sprintf("%d", i))
		types.Encode(&header)
		batch.Set(h0, types.Encode(&header))
	}
	blockStore.height = 100
	batch.Write()

	blockStore.saveSequence = true
	blockStore.CreateSequences(10)
	seq, err := blockStore.LoadBlockLastSequence()
	assert.Nil(t, err)
	assert.Equal(t, int64(100), seq)

	seq, err = blockStore.GetSequenceByHash([]byte("1"))
	assert.Nil(t, err)
	assert.Equal(t, int64(1), seq)

	seq, err = blockStore.GetSequenceByHash([]byte("0"))
	assert.Nil(t, err)
	assert.Equal(t, int64(0), seq)
}

func TestHasTx(t *testing.T) {
	dir, err := ioutil.TempDir("", "example")
	assert.Nil(t, err)
	defer os.RemoveAll(dir) // clean up
	os.RemoveAll(dir)       //删除已存在目录

	blockStoreDB := dbm.NewDB("blockchain", "leveldb", dir, 100)

	chain := InitEnv()
	cfg := chain.client.GetConfig()
	blockStore := NewBlockStore(chain, blockStoreDB, chain.client)
	assert.NotNil(t, blockStore)
	blockStore.saveSequence = false
	blockStore.isParaChain = false
	cfg.S("quickIndex", true)

	//txstring1 和txstring2的短hash是一样的，但是全hash是不一样的
	txstring1 := "0xaf095d11326ebb97d142fdb0e0138ef28524470c121b4811bdd05857b2d06764"
	txstring2 := "0xaf095d11326ebb97d142fdb0e0138ef28524470c121b4811bdd05857b2d06765"
	txstring3 := "0x8fac317e02ee25b1bbc5bd5a8570962b482928b014d14817b3c7a4e6aeddb3c6"
	txstring4 := "0x6522279c4fae53965e7bfbd35651dcd68813a50c65bf7af20b02c9bfe3d2ce8b"

	txhash1, err := common.FromHex(txstring1)
	assert.Nil(t, err)
	txhash2, err := common.FromHex(txstring2)
	assert.Nil(t, err)
	txhash3, err := common.FromHex(txstring3)
	assert.Nil(t, err)
	txhash4, err := common.FromHex(txstring4)
	assert.Nil(t, err)

	batch := blockStore.NewBatch(true)

	var txresult types.TxResult
	txresult.Height = 1
	txresult.Index = int32(1)
	batch.Set(cfg.CalcTxKey(txhash1), types.Encode(&txresult))
	batch.Set(types.CalcTxShortKey(txhash1), []byte("1"))

	txresult.Height = 3
	txresult.Index = int32(3)
	batch.Set(cfg.CalcTxKey(txhash3), types.Encode(&txresult))
	batch.Set(types.CalcTxShortKey(txhash3), []byte("1"))

	batch.Write()

	has, _ := blockStore.HasTx(txhash1)
	assert.Equal(t, has, true)

	has, _ = blockStore.HasTx(txhash2)
	assert.Equal(t, has, false)

	has, _ = blockStore.HasTx(txhash3)
	assert.Equal(t, has, true)

	has, _ = blockStore.HasTx(txhash4)
	assert.Equal(t, has, false)
}
