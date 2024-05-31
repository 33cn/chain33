package snowman

import (
	"testing"
	"time"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/system/consensus"
	"github.com/33cn/chain33/system/consensus/snowman/utils"
	"github.com/33cn/chain33/types"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/choices"
	"github.com/stretchr/testify/require"
)

func newTestCtx() (queue.Queue, *consensus.Context) {

	q := queue.New("test")
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	q.SetConfig(cfg)
	base := consensus.NewBaseClient(cfg.GetModuleConfig().Consensus)
	base.InitClient(q.Client(), func() {})
	ctx := &consensus.Context{Base: base}
	return q, ctx
}

func mockHandleChainMsg(cli queue.Client) {

	cli.Sub("blockchain")

	for msg := range cli.Recv() {

		if msg.Ty == types.EventSnowmanLastChoice {
			msg.Reply(cli.NewMessage("", 0, &types.SnowChoice{Height: 1, Hash: []byte("test")}))
		} else if msg.Ty == types.EventGetBlockByHashes {
			msg.Reply(cli.NewMessage("", 0, &types.BlockDetails{Items: []*types.BlockDetail{{Block: &types.Block{Height: 1}}}}))
		} else if msg.Ty == types.EventGetBlockHash {
			msg.Reply(cli.NewMessage("", 0, &types.ReplyHash{Hash: []byte("test")}))
		} else if msg.Ty == types.EventGetLastHeader {
			msg.Reply(cli.NewMessage("", 0, &types.Header{Height: 130}))
		}
	}
}

func TestChain33VM(t *testing.T) {

	_, ctx := newTestCtx()
	cli := ctx.Base.GetQueueClient()
	defer cli.Close()
	go mockHandleChainMsg(cli)

	vm := &chain33VM{}

	// test init
	vm.Init(ctx)
	_, err := vm.LastAccepted(nil)
	require.Nil(t, err)
	require.Equal(t, 1, int(vm.acceptedHeight))

	// test get/parse block
	sb, err := vm.GetBlock(nil, ids.Empty)
	require.Nil(t, err)
	require.Equal(t, 1, int(sb.Height()))
	require.Equal(t, choices.Processing, sb.Status())
	blk := sb.(*snowBlock).block
	sb1, err := vm.ParseBlock(nil, sb.Bytes())
	require.Nil(t, err)
	require.Equal(t, choices.Rejected, sb1.Status())
	vm.decidedHashes.Add(sb.ID(), true)
	sb1, _ = vm.ParseBlock(nil, sb.Bytes())
	require.Equal(t, choices.Accepted, sb1.Status())
	require.Nil(t, err)
	// test and and build new block
	require.False(t, vm.addNewBlock(blk))
	_, err = vm.BuildBlock(nil)
	require.Equal(t, utils.ErrBlockNotReady, err)
	blk.Height = vm.acceptedHeight + 1
	require.True(t, vm.addNewBlock(blk))
	sb1, err = vm.BuildBlock(nil)
	require.Equal(t, blk.Height, int64(sb1.Height()))
	require.Nil(t, err)
	// test GetBlockIDAtHeight
	id, _ := vm.GetBlockIDAtHeight(nil, 0)
	require.Equal(t, "test", string(id[:4]))

	// test accept or reject block
	require.Nil(t, sb1.Accept(nil))
	val, _ := vm.decidedHashes.Get(sb1.ID())
	require.Equal(t, blk.Height, vm.acceptedHeight)
	require.True(t, val.(bool))
	require.Nil(t, sb1.Reject(nil))
	val, _ = vm.decidedHashes.Get(sb1.ID())
	require.False(t, val.(bool))

	// test reset vm
	blk.Height = vm.acceptedHeight + 1
	require.True(t, vm.addNewBlock(blk))
	vm.reset()
	require.Equal(t, 0, vm.decidedHashes.Len())
	require.Equal(t, 0, int(vm.acceptedHeight))
	require.Equal(t, 0, vm.pendingBlocks.Len())
}

func TestVmUnimplementMethod(t *testing.T) {

	vm := &chain33VM{}
	require.Nil(t, vm.Initialize(nil, nil, nil, nil, nil, nil, nil, nil, nil))
	require.Nil(t, vm.SetState(nil, 0))
	require.Nil(t, vm.SetPreference(nil, ids.Empty))
	require.Nil(t, vm.Shutdown(nil))
	require.Nil(t, vm.VerifyHeightIndex(nil))

	require.Nil(t, vm.CrossChainAppRequestFailed(nil, ids.Empty, 0))
	require.Nil(t, vm.CrossChainAppRequest(nil, ids.Empty, 0, types.Now(), nil))
	require.Nil(t, vm.CrossChainAppResponse(nil, ids.Empty, 0, nil))
	require.Nil(t, vm.AppRequest(nil, ids.EmptyNodeID, 0, time.Now(), nil))
	require.Nil(t, vm.AppRequestFailed(nil, ids.EmptyNodeID, 0))
	require.Nil(t, vm.AppResponse(nil, ids.EmptyNodeID, 0, nil))
	require.Nil(t, vm.AppGossip(nil, ids.EmptyNodeID, nil))
	require.Nil(t, vm.GossipTx(nil))
	_, err := vm.HealthCheck(nil)
	require.Nil(t, err)
	require.Nil(t, vm.Connected(nil, ids.EmptyNodeID, nil))
	require.Nil(t, vm.Disconnected(nil, ids.EmptyNodeID))
	_, err = vm.CreateHandlers(nil)
	require.Nil(t, err)
	_, err = vm.CreateStaticHandlers(nil)
	require.Nil(t, err)
	_, err = vm.Version(nil)
	require.Nil(t, err)
}
