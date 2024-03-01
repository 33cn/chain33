package snowman

import (
	"container/list"
	"encoding/hex"
	"sync"
	"sync/atomic"

	lru "github.com/hashicorp/golang-lru"

	"github.com/ava-labs/avalanchego/snow/choices"

	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/system/consensus"
	"github.com/33cn/chain33/system/consensus/snowman/utils"
	"github.com/33cn/chain33/types"
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/database/manager"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow"
	snowcon "github.com/ava-labs/avalanchego/snow/consensus/snowman"
	"github.com/ava-labs/avalanchego/snow/engine/common"
	"github.com/ava-labs/avalanchego/snow/engine/snowman/block"

	"context"
)

var (
	_ common.VM     = (*chain33VM)(nil)
	_ block.ChainVM = (*chain33VM)(nil)
)

// implements the snowman.ChainVM interface
type chain33VM struct {
	blankVM
	api          client.QueueProtocolAPI
	cfg          *types.Chain33Config
	qclient      queue.Client
	pendingBlock map[string]*types.Block

	pendingBlocks  *list.List
	lock           sync.RWMutex
	preferenceID   ids.ID
	acceptedHeight int64
	preferChan     chan ids.ID

	decidedHashes *lru.Cache
}

func (vm *chain33VM) newSnowBlock(blk *types.Block, status choices.Status) *snowBlock {

	sb := &snowBlock{block: blk, vm: vm, status: status}
	sb.id = toSnowID(blk.Hash(vm.cfg))
	return sb
}

// Init init chain33 vm
func (vm *chain33VM) Init(ctx *consensus.Context) {

	vm.api = ctx.Base.GetAPI()
	vm.cfg = vm.api.GetConfig()
	vm.qclient = ctx.Base.GetQueueClient()
	vm.pendingBlock = make(map[string]*types.Block, 8)
	vm.preferChan = make(chan ids.ID, 32)
	c, err := lru.New(1024)
	if err != nil {
		panic("chain33VM Init New lru err" + err.Error())
	}
	vm.decidedHashes = c
	vm.pendingBlocks = list.New()
	choice, err := getLastChoice(vm.qclient)
	if err != nil {
		snowLog.Error("vm Init", "getLastChoice err", err)
		panic(err)
	}
	vm.acceptedHeight = choice.Height
	go vm.handleNotifyNewBlock(ctx.Base.Context)
}

// Initialize implements the snowman.ChainVM interface
func (vm *chain33VM) Initialize(
	_ context.Context,
	chainCtx *snow.Context,
	dbManager manager.Manager,
	genesisBytes []byte,
	upgradeBytes []byte,
	configBytes []byte,
	toEngine chan<- common.Message,
	fxs []*common.Fx,
	appSender common.AppSender,
) error {

	return nil

}

func (vm *chain33VM) handleNotifyNewBlock(ctx context.Context) {

	for {

		select {

		case <-ctx.Done():
			return

		case preferID := <-vm.preferChan:

			snowLog.Debug("handleNotifyNewBlock", "hash", hex.EncodeToString(preferID[:]))
		}

	}
}

// SetState communicates to VM its next state it starts
func (vm *chain33VM) SetState(ctx context.Context, state snow.State) error {

	return nil
}

// Shutdown is called when the node is shutting down.
func (vm *chain33VM) Shutdown(context.Context) error {

	return nil
}

// GetBlock get block
func (vm *chain33VM) GetBlock(_ context.Context, blkID ids.ID) (snowcon.Block, error) {

	details, err := vm.api.GetBlockByHashes(&types.ReqHashes{Hashes: [][]byte{blkID[:]}})
	if err != nil || len(details.GetItems()) < 1 || details.GetItems()[0].GetBlock() == nil {
		snowLog.Error("GetBlock", "hash", blkID.Hex(), "GetBlockByHashes err", err)
		return nil, database.ErrNotFound
	}
	sb := vm.newSnowBlock(details.GetItems()[0].GetBlock(), choices.Processing)
	acceptHeight := atomic.LoadInt64(&vm.acceptedHeight)
	if sb.block.Height <= acceptHeight {
		sb.status = choices.Accepted
	}

	return sb, nil
}

// ParseBlock parse block
func (vm *chain33VM) ParseBlock(_ context.Context, b []byte) (snowcon.Block, error) {

	blk := &types.Block{}
	err := types.Decode(b, blk)
	if err != nil {
		snowLog.Error("ParseBlock", "decode err", err)
		return nil, err
	}
	sb := vm.newSnowBlock(blk, choices.Unknown)
	accepted, ok := vm.decidedHashes.Get(sb.ID())
	if ok {
		sb.status = choices.Rejected
		if accepted.(bool) {
			sb.status = choices.Accepted
		}
	}

	return sb, nil
}

func (vm *chain33VM) addNewBlock(blk *types.Block) bool {

	ah := atomic.LoadInt64(&vm.acceptedHeight)
	if blk.GetHeight() <= ah {
		return false
	}

	vm.lock.Lock()
	defer vm.lock.Unlock()
	vm.pendingBlocks.PushBack(vm.newSnowBlock(blk, choices.Processing))
	snowLog.Debug("addNewBlock", "ah", ah, "bh", blk.GetHeight(), "pendingNum", vm.pendingBlocks.Len())
	return true

	//key := string(blk.ParentHash)
	//exist, ok := vm.pendingBlock[key]
	//if ok {
	//	snowLog.Debug("addNewBlock replace block", "height", blk.Height, "old", hex.EncodeToString(exist.Hash(vm.cfg)),
	//		"new", hex.EncodeToString(blk.Hash(vm.cfg)))
	//}
	//vm.pendingBlock[key] = blk
}

// BuildBlock Attempt to create a new block from data contained in the VM.
//
// If the VM doesn't want to issue a new block, an error should be
// returned.
func (vm *chain33VM) BuildBlock(context.Context) (snowcon.Block, error) {

	vm.lock.Lock()
	defer vm.lock.Unlock()

	if vm.pendingBlocks.Len() <= 0 {
		return nil, utils.ErrBlockNotReady
	}

	sb := vm.pendingBlocks.Remove(vm.pendingBlocks.Front()).(*snowBlock)
	snowLog.Debug("BuildBlock", "pendingNum", vm.pendingBlocks.Len(), "height", sb.Height(), "hash", sb.id.Hex())
	return sb, nil
}

// SetPreference Notify the VM of the currently preferred block.
//
// This should always be a block that has no children known to consensus.
func (vm *chain33VM) SetPreference(ctx context.Context, blkID ids.ID) error {

	vm.lock.Lock()
	vm.preferenceID = blkID
	vm.lock.Unlock()

	snowLog.Debug("SetPreference", "blkHash", blkID.Hex())

	err := vm.qclient.Send(vm.qclient.NewMessage("blockchain",
		types.EventSnowmanPreferBlk, &types.ReqBytes{Data: blkID[:]}), false)

	if err != nil {
		snowLog.Error("SetPreference", "blkHash", blkID.Hex(), "send queue err", err)
		return err
	}

	return nil
}

func getLastChoice(cli queue.Client) (*types.SnowChoice, error) {

	msg := cli.NewMessage("blockchain", types.EventSnowmanLastChoice, &types.ReqNil{})
	err := cli.Send(msg, true)
	if err != nil {
		return nil, err
	}

	reply, err := cli.Wait(msg)
	if err != nil {
		return nil, err
	}
	return reply.GetData().(*types.SnowChoice), nil
}

// LastAccepted returns the ID of the last accepted block.
//
// If no blocks have been accepted by consensus yet, it is assumed there is
// a definitionally accepted block, the Genesis block, that will be
// returned.
func (vm *chain33VM) LastAccepted(context.Context) (ids.ID, error) {

	choice, err := getLastChoice(vm.qclient)
	if err != nil {
		snowLog.Error("LastAccepted", "getLastChoice err", err)
		return ids.Empty, err
	}

	atomic.StoreInt64(&vm.acceptedHeight, choice.Height)
	var id ids.ID
	copy(id[:], choice.Hash)
	return id, nil
}

// VerifyHeightIndex should return:
//   - nil if the height index is available.
//   - ErrIndexIncomplete if the height index is not currently available.
//   - Any other non-standard error that may have occurred when verifying the
//     index.
//
// TODO: Remove after avalanche v1.11.x activates.
func (vm *chain33VM) VerifyHeightIndex(context.Context) error {
	return nil
}

// GetBlockIDAtHeight returns:
// - The ID of the block that was accepted with [height].
// - database.ErrNotFound if the [height] index is unknown.
//
// Note: A returned value of [database.ErrNotFound] typically means that the
//
//	underlying VM was state synced and does not have access to the
//	blockID at [height].
func (vm *chain33VM) GetBlockIDAtHeight(ctx context.Context, height uint64) (ids.ID, error) {

	reply, err := vm.api.GetBlockHash(&types.ReqInt{Height: int64(height)})
	if err != nil {
		snowLog.Error("GetBlock", "height", height, "GetBlockHash err", err)
		return ids.Empty, database.ErrNotFound
	}
	return toSnowID(reply.Hash), nil
}

func (vm *chain33VM) acceptBlock(height int64, blkID ids.ID) error {

	ah := atomic.LoadInt64(&vm.acceptedHeight)
	if height <= ah {
		snowLog.Debug("acceptBlock disorder", "height", height, "accept", ah)
		return nil
	}
	atomic.StoreInt64(&vm.acceptedHeight, height)
	vm.decidedHashes.Add(blkID, true)

	err := vm.qclient.Send(vm.qclient.NewMessage("blockchain",
		types.EventSnowmanAcceptBlk, &types.SnowChoice{Height: height, Hash: blkID[:]}), false)

	return err
}

func (vm *chain33VM) rejectBlock(height int64, blkID ids.ID) {

	vm.decidedHashes.Add(blkID, false)
}

func (vm *chain33VM) removeExpireBlock() {

	for key, blk := range vm.pendingBlock {

		if blk.Height <= vm.acceptedHeight {
			delete(vm.pendingBlock, key)
		}
	}
}
