package blockchain

import (
	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func newTestChain(t *testing.T) (*BlockChain, string) {

	chain := InitEnv()
	dir, err := os.MkdirTemp("", "finalize")
	require.Nil(t, err)
	blockStoreDB := dbm.NewDB("blockchain", "leveldb", dir, 64)
	chain.blockStore = NewBlockStore(chain, blockStoreDB, nil)
	node := newPreGenBlockNode()
	node.parent = nil
	chain.bestChain = newChainView(node)
	return chain, dir
}

func TestFinalizer(t *testing.T) {

	chain, dir := newTestChain(t)
	defer os.RemoveAll(dir)

	f := &finalizer{}
	f.Init(chain)

	hash := []byte("testhash")
	choice := &types.SnowChoice{Height: 1, Hash: hash}
	msg := queue.NewMessage(0, "test", 0, choice)

	f.snowmanPreferBlock(msg)
	f.snowmanAcceptBlock(msg)
	height, hash1 := f.getLastFinalized()
	require.Equal(t, 0, int(height))
	require.Equal(t, 0, len(hash1))
	node := &blockNode{
		parent: chain.bestChain.Tip(),
		height: 1,
		hash:   hash,
	}
	chain.bestChain.SetTip(node)
	f.snowmanAcceptBlock(msg)
	height, hash1 = f.getLastFinalized()
	require.Equal(t, 1, int(height))
	require.Equal(t, hash, hash1)

	f.snowmanLastChoice(msg)
	msg1, err := chain.client.Wait(msg)
	require.Nil(t, err)
	require.Equal(t, msg1.Data, choice)
	f.Init(chain)
}
