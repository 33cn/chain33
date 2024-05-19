package snowman

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/33cn/chain33/client/mocks"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/mock"
)

func TestSnowman(t *testing.T) {

	api := new(mocks.QueueProtocolAPI)
	q, ctx := newTestCtx()

	gblock := &types.Block{Height: 1}
	sc := &types.SnowChoice{Height: 1, Hash: gblock.Hash(q.GetConfig())}
	api.On("GetBlockByHashes", mock.Anything).Return(&types.BlockDetails{Items: []*types.BlockDetail{{Block: gblock}}}, nil)
	api.On("GetFinalizedBlock").Return(sc, nil)
	api.On("GetConfig").Return(q.GetConfig())

	defer q.Close()
	defer ctx.Base.Cancel()
	ctx.Base.SetAPI(api)
	cli := q.Client()
	go func() {

		cli.Sub("p2p")
		for msg := range cli.Recv() {
			if msg.Ty == types.EventPeerInfo {
				msg.Reply(cli.NewMessage("", 0, &types.PeerList{}))
			} else {
				msg.Reply(cli.NewMessage("", 0, &types.Reply{}))
			}
		}
	}()

	sm := &snowman{}
	sm.Initialize(ctx)

	for timeout := 0; sm.initDone.Load() == false; timeout++ {
		time.Sleep(time.Second)
		timeout++
		if timeout >= 3 {
			t.Errorf("wait snowman init timeout")
			return
		}
	}

	blk := &types.Block{ParentHash: sc.Hash, Height: 2}
	blkHash := blk.Hash(q.GetConfig())
	sm.AddBlock(blk)
	peer := "peerid"
	reqID := uint32(0)

	sm.SubMsg(queue.NewMessage(types.EventSnowmanChits, "", 0,
		&types.SnowChits{PeerName: peer, PreferredBlkHash: blkHash, AcceptedBlkHash: blkHash}))
	sm.SubMsg(queue.NewMessage(types.EventSnowmanPutBlock, "", 0,
		&types.SnowPutBlock{RequestID: reqID + 1, PeerName: peer, BlockHash: blkHash, BlockData: types.Encode(blk)}))
	sm.SubMsg(queue.NewMessage(types.EventSnowmanPullQuery, "", 0,
		&types.SnowPullQuery{RequestID: reqID + 2, PeerName: peer, BlockHash: blkHash}))
	sm.SubMsg(queue.NewMessage(types.EventSnowmanPushQuery, "", 0,
		&types.SnowPushQuery{RequestID: reqID + 3, PeerName: peer, BlockData: types.Encode(blk)}))
	sm.SubMsg(queue.NewMessage(types.EventSnowmanQueryFailed, "", 0,
		&types.SnowFailedQuery{RequestID: reqID + 4, PeerName: peer}))
	sm.SubMsg(queue.NewMessage(types.EventSnowmanGetFailed, "", 0,
		&types.SnowFailedQuery{RequestID: reqID + 5, PeerName: peer}))

	sc.Height = 2
	sc.Hash = blk.Hash(q.GetConfig())
	sm.SubMsg(queue.NewMessage(types.EventSnowmanResetEngine, "", 0, nil))
	for timeout := 0; atomic.LoadInt64(&sm.vm.acceptedHeight) < 2; timeout++ {
		time.Sleep(time.Second)
		timeout++
		if timeout >= 3 {
			t.Errorf("test timeout acceptHeight=%d", atomic.LoadInt64(&sm.vm.acceptedHeight))
			return
		}
	}
}
