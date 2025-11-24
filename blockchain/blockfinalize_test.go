package blockchain

import (
	"os"
	"testing"
	"time"

	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/require"
)

func newTestChain(t *testing.T) (*BlockChain, string) {

	chain := InitEnv()
	chain.client.GetConfig().GetModuleConfig().Consensus.Finalizer = "snowman"
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
	defer chain.Close()
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

}

func TestResetEngine(t *testing.T) {

	chain, dir := newTestChain(t)
	defer os.RemoveAll(dir)
	defer chain.Close()
	f := &finalizer{}
	f.Init(chain)

	hash := []byte("testhash")
	choice := &types.SnowChoice{Height: 1, Hash: hash}
	node := &blockNode{
		parent: chain.bestChain.Tip(),
		height: 14,
		hash:   hash,
	}
	chain.bestChain.SetTip(node)
	chain.client.Sub(consensusTopic)
	go f.resetEngine(1, choice, time.Second)
	recvMsg := func() {
		select {
		case msg := <-chain.client.Recv():
			require.Equal(t, "consensus", msg.Topic)
			require.Equal(t, int64(types.EventSnowmanResetEngine), msg.ID)
			require.Equal(t, int64(types.EventForFinalizer), msg.Ty)
		case <-time.After(time.Second * 5):
			t.Error("resetEngine timeout")
		}
	}
	recvMsg()
	f.reset(choice.Height, choice.Hash)
	recvMsg()
	height, hash1 := f.getLastFinalized()
	require.Equal(t, choice.Height, height)
	require.Equal(t, hash, hash1)

}
