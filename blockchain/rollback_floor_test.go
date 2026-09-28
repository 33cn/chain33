// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"io/ioutil"
	"os"
	"testing"

	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/require"
)

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
