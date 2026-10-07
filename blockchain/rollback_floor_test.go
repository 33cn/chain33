// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/require"
)

// startupWindow mirrors the window NeedRollback adds: the larger of the two ranges InitCache
// reads above the tip.
func startupWindow(chain *BlockChain) int64 {
	window := types.HighAllowPackHeight + types.LowAllowPackHeight - 1
	if chain.cfg.DefCacheSize > window {
		window = chain.cfg.DefCacheSize
	}
	return window
}

// archiveChunk does what the archiver does in two steps: write the record chunkShardHandle
// would have written, then drop exactly the bodies DeleteBlockBody names.
func archiveChunk(t *testing.T, chain *BlockChain, chunkNum, start, end int64) {
	require.NoError(t, chain.blockStore.Set(calcChunkNumToHash(chunkNum),
		types.Encode(&types.ChunkInfo{Start: start, End: end})))
	kvs := chain.DeleteBlockBody(chunkNum)
	require.NotEmpty(t, kvs)
	chain.blockStore.mustSaveKvset(&types.LocalDBSet{KV: kvs})
}

func newArchivedChain(t *testing.T, tip int64) *BlockChain {
	dir, err := ioutil.TempDir("", "rollbackfloor")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })

	chain := InitEnv()
	blockStore := NewBlockStore(chain, dbm.NewDB("blockchain", "leveldb", dir, 100), chain.client)
	require.NotNil(t, blockStore)
	chain.blockStore = blockStore
	chain.cfg.ChunkblockNum = 1000
	saveBlockToDB(chain, 0, tip)
	return chain
}

// An archived datadir cannot be rolled back past the archive, and the way that shows up is
// the restart, not the deletion loop: once the tip has moved down, InitCache reads the
// bodies above the new tip and panics on the first one that is gone. Build that datadir and
// start the chain at both the lowest accepted target and one block below it.
func TestRollbackTargetMustLeaveTheStartupWindowIntact(t *testing.T) {
	dir, err := ioutil.TempDir("", "rollbackfloor")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	chain := InitEnv()
	blockStore := NewBlockStore(chain, dbm.NewDB("blockchain", "leveldb", dir, 100), chain.client)
	require.NotNil(t, blockStore)
	chain.blockStore = blockStore
	chain.cfg.ChunkblockNum = 1000

	const tip = int64(1999)
	saveBlockToDB(chain, 0, tip)
	// archive the first chunk: write the record the archiver would have written, then drop
	// exactly the bodies it names (DeleteBlockBody returns the keys, its caller writes them)
	require.NoError(t, blockStore.Set(calcChunkNumToHash(0),
		types.Encode(&types.ChunkInfo{Start: 0, End: 999})))
	kvs := chain.DeleteBlockBody(0)
	require.NotEmpty(t, kvs)
	chain.blockStore.mustSaveKvset(&types.LocalDBSet{KV: kvs})
	require.NoError(t, blockStore.SetMaxDeletedChunkNum(0))

	// the archived bodies really are gone from the store
	head, err := chain.blockStore.loadHeaderByIndex(999)
	require.NoError(t, err)
	_, err = getBodyByIndex(blockStore.db, "", calcHeightHashKey(999, head.Hash), nil)
	require.Error(t, err, "precondition: height 999 has no body left")

	// InitCache reads [tip-DefCacheSize, tip] and the taller tx-height window above the tip,
	// so the lowest target that still starts is the floor plus the larger of the two
	window := types.HighAllowPackHeight + types.LowAllowPackHeight - 1
	if chain.cfg.DefCacheSize > window {
		window = chain.cfg.DefCacheSize
	}
	lowest := int64(1000) + window

	require.False(t, chain.NeedRollback(tip, lowest-1),
		"one block lower leaves a body the next start has to read")
	require.True(t, chain.NeedRollback(tip, lowest))

	require.NotPanics(t, func() { chain.InitCache(lowest) }, "the accepted target has to start")
	require.Panics(t, func() { chain.InitCache(lowest - 1) }, "and the refused one must not")
}

// The deletion routine stops DelRollbackChunkNum chunks short of the newest archived chunk
// and keeps running while the node is up, so the chunk the cursor names is not the last one
// the archiver will delete: the backlog from the previous run drains into the range the
// rollback is about to depend on. Here the cursor is still on chunk 0, chunk 1 is already
// dropped with a different chunk size on record, and the archiver is twelve chunks ahead --
// the floor has to come from chunk 1, not from the cursor.
func TestRollbackFloorFollowsTheArchiverNotTheCursor(t *testing.T) {
	const tip = int64(2999)
	chain := newArchivedChain(t, tip)
	archiveChunk(t, chain, 0, 0, 999)
	archiveChunk(t, chain, 1, 1000, 1499)
	require.NoError(t, chain.blockStore.SetMaxDeletedChunkNum(0)) // the last run stopped here
	require.NoError(t, chain.updateMaxSerialChunkNum(12))         // ten chunks of backlog left

	floor, archived, err := chain.archivedBodyFloor()
	require.NoError(t, err)
	require.True(t, archived)
	require.Equal(t, int64(1500), floor, "the floor is the last chunk the archiver will delete")

	lowest := floor + startupWindow(chain)
	require.False(t, chain.NeedRollback(tip, lowest-1),
		"a target whose bodies the backlog is about to delete has to be refused")
	require.True(t, chain.NeedRollback(tip, lowest))
	require.NotPanics(t, func() { chain.InitCache(lowest) })
	require.Panics(t, func() { chain.InitCache(lowest - 1) })
}

// A record that cannot be read is not "nothing was archived": the configured chunk size can
// be lower than the size the archiver used, so falling back to it would clear a target whose
// bodies are gone. Refuse instead.
func TestRollbackFloorRefusesAnUnreadableRecord(t *testing.T) {
	chain := newArchivedChain(t, 1999)
	require.NoError(t, chain.blockStore.Set(calcChunkNumToHash(0), []byte("not a chunk record")))
	require.NoError(t, chain.blockStore.SetMaxDeletedChunkNum(0))

	floor, archived, err := chain.archivedBodyFloor()
	require.Error(t, err, "a failed read must reach the caller, not the fallback")
	require.False(t, archived)
	require.Zero(t, floor)
}

// A cursor with no record behind it is a datadir from before the records were written: the
// configured chunk size is then the only bound available.
func TestRollbackFloorFallsBackToTheConfiguredChunkSize(t *testing.T) {
	chain := newArchivedChain(t, 1999)
	require.NoError(t, chain.blockStore.SetMaxDeletedChunkNum(0))

	floor, archived, err := chain.archivedBodyFloor()
	require.NoError(t, err)
	require.True(t, archived)
	require.Equal(t, int64(1000), floor)
}

// bityuan runs with TxHeight disabled, where InitCache returns before it reads a body, so the
// only bodies a rollback needs are the ones it walks over on the way down -- the window does
// not apply and the floor is the target bound itself.
func TestRollbackFloorWithoutTxHeight(t *testing.T) {
	dir, err := ioutil.TempDir("", "rollbackfloor")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	cfg := types.NewChain33Config(strings.Replace(types.GetDefaultCfgstring(), "TxHeight=true", "TxHeight=false", 1))
	q := queue.New("channel")
	q.SetConfig(cfg)
	chain := New(cfg)
	chain.client = q.Client()
	require.False(t, chain.client.GetConfig().IsEnable("TxHeight"), "precondition")
	blockStore := NewBlockStore(chain, dbm.NewDB("blockchain", "leveldb", dir, 100), chain.client)
	chain.blockStore = blockStore
	chain.cfg.ChunkblockNum = 1000

	const tip = int64(1999)
	saveBlockToDB(chain, 0, tip)
	archiveChunk(t, chain, 0, 0, 999)
	require.NoError(t, chain.blockStore.SetMaxDeletedChunkNum(0))

	floor, archived, err := chain.archivedBodyFloor()
	require.NoError(t, err)
	require.True(t, archived)
	require.Equal(t, int64(1000), floor)
	require.False(t, chain.NeedRollback(tip, 999), "999 has no body left")
	require.True(t, chain.NeedRollback(tip, 1000))
}
