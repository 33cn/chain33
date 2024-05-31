package snowman

import (
	"bytes"
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
	api     client.QueueProtocolAPI
	cfg     *types.Chain33Config
	qclient queue.Client

	pendingBlocks  *list.List
	lock           sync.RWMutex
	acceptedHeight int64
	decidedHashes  *lru.Cache
	cacheBlks      *lru.Cache
}

func (vm *chain33VM) newSnowBlock(blk *types.Block, status choices.Status) *snowBlock {

	sb := &snowBlock{block: blk, vm: vm, status: status}
	sb.id = toSnowID(blk.Hash(vm.cfg))
	sb.parent = toSnowID(blk.GetParentHash())
	return sb
}

// Init init chain33 vm
func (vm *chain33VM) Init(ctx *consensus.Context) {

	vm.api = ctx.Base.GetAPI()
	vm.cfg = vm.api.GetConfig()
	vm.qclient = ctx.Base.GetQueueClient()

	c, err := lru.New(1024)
	if err != nil {
		panic("chain33VM Init New lru err" + err.Error())
	}
	vm.decidedHashes = c
	vm.cacheBlks, err = lru.New(128)
	if err != nil {
		panic("chain33VM Init New reqBlks lru err" + err.Error())
	}
	vm.pendingBlocks = list.New()
	choice, err := vm.api.GetFinalizedBlock()
	if err != nil {
		snowLog.Error("vm Init", "getLastChoice err", err)
		panic(err)
	}
	vm.acceptedHeight = choice.Height
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

func (vm *chain33VM) reset() {
	vm.decidedHashes.Purge()
	vm.pendingBlocks.Init()
	atomic.StoreInt64(&vm.acceptedHeight, 0)
}

// SetState communicates to VM its next state it starts
func (vm *chain33VM) SetState(ctx context.Context, state snow.State) error {

	return nil
}

// Shutdown is called when the node is shutting down.
func (vm *chain33VM) Shutdown(context.Context) error {

	return nil
}

func (vm *chain33VM) checkDecided(sb *snowBlock) {

	accepted, ok := vm.decidedHashes.Get(sb.ID())
	if ok {
		sb.status = choices.Rejected
		if accepted.(bool) {
			sb.status = choices.Accepted
		}
	}
}

// GetBlock get block
func (vm *chain33VM) GetBlock(_ context.Context, blkID ids.ID) (snowcon.Block, error) {

	val, ok := vm.cacheBlks.Get(blkID)
	if ok {
		sb := val.(*snowBlock)
		snowLog.Debug("GetBlock cache", "hash", blkID.Hex(), "height", sb.Height(), "status", sb.Status().String())
		sb.status = choices.Processing
		vm.checkDecided(sb)
		return sb, nil
	}
	details, err := vm.api.GetBlockByHashes(&types.ReqHashes{Hashes: [][]byte{blkID[:]}})
	if err != nil || len(details.GetItems()) < 1 || details.GetItems()[0].GetBlock() == nil {
		snowLog.Debug("vmGetBlock failed", "hash", blkID.Hex(), "err", err)
		return nil, database.ErrNotFound
	}
	sb := vm.newSnowBlock(details.GetItems()[0].GetBlock(), choices.Processing)
	vm.checkDecided(sb)
	return sb, nil
}

// 检测当前节点对应区块偏好
func (vm *chain33VM) checkPreference(sb *snowBlock) {

	if sb.Status().Decided() {
		return
	}
	header, err := vm.api.GetLastHeader()
	if err != nil {
		snowLog.Error("checkPreference", "height", sb.Height(), "hash", sb.ID().Hex(), "GetLastHeader err", err)
		return
	}
	// 高度128以内不处理
	if header.GetHeight() < sb.block.Height+128 {
		return
	}

	hash, err := vm.api.GetBlockHash(&types.ReqInt{Height: sb.block.GetHeight()})
	if err != nil || len(hash.GetHash()) <= 0 {
		snowLog.Error("checkPreference", "height", sb.Height(), "hash", sb.ID().Hex(), "GetBlockHash err", err)
		return
	}
	// hash不一致, 为分叉区块
	if !bytes.Equal(hash.GetHash(), sb.block.Hash(vm.cfg)) {
		sb.status = choices.Rejected
		vm.decidedHashes.Add(sb.ID(), false)
		snowLog.Debug("checkPreference reject", "height", sb.Height(), "hash", sb.ID().Hex(), "mainHash", hex.EncodeToString(hash.GetHash()))
	}
}

// ParseBlock parse block fetch from peer
func (vm *chain33VM) ParseBlock(_ context.Context, b []byte) (snowcon.Block, error) {

	blk := &types.Block{}
	err := types.Decode(b, blk)
	if err != nil {
		snowLog.Error("vmParseBlock", "decode err", err)
		return nil, err
	}
	sb := vm.newSnowBlock(blk, choices.Processing)
	vm.checkDecided(sb)
	vm.checkPreference(sb)
	// add to cache
	vm.cacheBlks.Add(sb.ID(), sb)
	return sb, nil
}

const maxPendingNum = 128

func (vm *chain33VM) addNewBlock(blk *types.Block) bool {

	ah := atomic.LoadInt64(&vm.acceptedHeight)
	if blk.GetHeight() <= ah {
		return false
	}

	vm.lock.Lock()
	defer vm.lock.Unlock()
	if vm.pendingBlocks.Len() > maxPendingNum {
		return false
	}
	vm.pendingBlocks.PushBack(vm.newSnowBlock(blk, choices.Processing))
	snowLog.Debug("vm addNewBlock", "height", blk.GetHeight(), "hash", hex.EncodeToString(blk.Hash(vm.cfg)),
		"parent", hex.EncodeToString(blk.ParentHash), "acceptedHeight", ah, "pendingNum", vm.pendingBlocks.Len())
	return true
}

// BuildBlock Attempt to create a new block from data contained in the VM.
//
// If the VM doesn't want to issue a new block, an error should be
// returned.
func (vm *chain33VM) BuildBlock(_ context.Context) (snowcon.Block, error) {

	ah := atomic.LoadInt64(&vm.acceptedHeight)
	vm.lock.Lock()
	defer vm.lock.Unlock()

	for vm.pendingBlocks.Len() > 0 {

		sb := vm.pendingBlocks.Remove(vm.pendingBlocks.Front()).(*snowBlock)
		if sb.Height() <= uint64(ah) {
			continue
		}
		snowLog.Debug("vmBuildBlock", "pendingNum", vm.pendingBlocks.Len(), "height", sb.Height(),
			"hash", sb.id.Hex(), "parent", sb.Parent().Hex())
		return sb, nil
	}

	return nil, utils.ErrBlockNotReady
}

// SetPreference Notify the VM of the currently preferred block.
//
// This should always be a block that has no children known to consensus.
func (vm *chain33VM) SetPreference(ctx context.Context, blkID ids.ID) error {

	snowLog.Debug("vmSetPreference", "blkHash", blkID.Hex())
	//err := vm.qclient.Send(vm.qclient.NewMessage("blockchain",
	//	types.EventSnowmanPreferBlk, &types.ReqBytes{Data: blkID[:]}), false)
	//
	//if err != nil {
	//	snowLog.Error("vmSetPreference", "blkHash", blkID.Hex(), "send queue err", err)
	//	return err
	//}

	return nil
}

// LastAccepted returns the ID of the last accepted block.
//
// If no blocks have been accepted by consensus yet, it is assumed there is
// a definitionally accepted block, the Genesis block, that will be
// returned.
func (vm *chain33VM) LastAccepted(_ context.Context) (ids.ID, error) {

	choice, err := vm.api.GetFinalizedBlock()
	if err != nil {
		snowLog.Error("vmLastAccepted", "getLastChoice err", err)
		return ids.Empty, err
	}

	atomic.StoreInt64(&vm.acceptedHeight, choice.Height)
	var id ids.ID
	copy(id[:], choice.Hash)
	vm.decidedHashes.Add(id, true)
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
		snowLog.Error("vmGetBlockIDAtHeight", "height", height, "GetBlockHash err", err)
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
