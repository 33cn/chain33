package snowman

import (
	"encoding/hex"
	"sync"

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

type preferBlock struct {
	height int64
	hash   []byte
}

// implements the snowman.ChainVM interface
type chain33VM struct {
	blankVM
	api          client.QueueProtocolAPI
	cfg          *types.Chain33Config
	qclient      queue.Client
	pendingBlock map[string]*types.Block
	lock         sync.RWMutex
	preferenceID ids.ID
	acceptHeight int64
}

func (vm *chain33VM) newSnowBlock(blk *types.Block) snowcon.Block {

	sb := &snowBlock{block: blk, vm: vm}
	copy(sb.id[:], blk.Hash(vm.cfg))
	return sb
}

// Init init chain33 vm
func (vm *chain33VM) Init(ctx *consensus.Context) {

	vm.api = ctx.Base.GetAPI()
	vm.cfg = vm.api.GetConfig()
	vm.qclient = ctx.Base.GetQueueClient()
	vm.pendingBlock = make(map[string]*types.Block, 8)
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
	if err != nil || details.GetItems()[0].GetBlock() == nil {
		snowLog.Error("GetBlock", "GetBlockByHashes err", err)
		return nil, database.ErrNotFound
	}

	return vm.newSnowBlock(details.GetItems()[0].GetBlock()), nil
}

// ParseBlock parse block
func (vm *chain33VM) ParseBlock(_ context.Context, b []byte) (snowcon.Block, error) {

	blk := &types.Block{}
	err := types.Decode(b, blk)
	if err != nil {
		snowLog.Error("ParseBlock", "decode err", err)
		return nil, err
	}

	return vm.newSnowBlock(blk), nil
}

func (vm *chain33VM) addNewBlock(blk *types.Block) {

	vm.lock.Lock()
	defer vm.lock.Unlock()
	key := string(blk.ParentHash)
	exist, ok := vm.pendingBlock[key]
	if !ok {
		snowLog.Debug("addNewBlock replace block", "old", hex.EncodeToString(exist.Hash(vm.cfg)),
			"new", hex.EncodeToString(blk.Hash(vm.cfg)))
	}
	vm.pendingBlock[key] = blk
}

// BuildBlock Attempt to create a new block from data contained in the VM.
//
// If the VM doesn't want to issue a new block, an error should be
// returned.
func (vm *chain33VM) BuildBlock(context.Context) (snowcon.Block, error) {

	vm.lock.RLock()
	defer vm.lock.RUnlock()

	blk, ok := vm.pendingBlock[string(vm.preferenceID[:])]

	if !ok {
		return nil, utils.ErrBlockNotReady
	}

	return vm.newSnowBlock(blk), nil
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

// LastAccepted returns the ID of the last accepted block.
//
// If no blocks have been accepted by consensus yet, it is assumed there is
// a definitionally accepted block, the Genesis block, that will be
// returned.
func (vm *chain33VM) LastAccepted(context.Context) (ids.ID, error) {


	msg := vm.qclient.NewMessage("blockchain", types.EventSnowmanLastAcceptHeight, &types.ReqNil{})
	err := vm.qclient.Send(msg, true)
	if err != nil {
		snowLog.Error("LastAccepted", "send msg err", err)
		return ids.Empty, err
	}

	reply, err := vm.qclient.Wait(msg)
	if err != nil {
		snowLog.Error("LastAccepted", "wait msg err", err)
		return ids.Empty, err
	}
	hash := reply.GetData().(*types.ReqBytes)
	var id ids.ID
	copy(id[:], hash.Data)
	return id, nil
}

// VerifyHeightIndex should return:
//   - nil if the height index is available.
//   - ErrIndexIncomplete if the height index is not currently available.
//   - Any other non-standard error that may have occurred when verifying the
//     index.
//
// TODO: Remove after v1.11.x activates.
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
	var id ids.ID
	copy(id[:], reply.Hash)

	return id, nil
}

func (vm *chain33VM) acceptBlock(height int64, blkID ids.ID) error {

	vm.lock.Lock()
	defer vm.lock.Unlock()
	vm.acceptHeight = height
	vm.removeExpireBlock()

	err := vm.qclient.Send(vm.qclient.NewMessage("blockchain",
		types.EventSnowmanAcceptBlk, &types.ReqBytes{Data: blkID[:]}), false)

	return err
}

func (vm *chain33VM) removeExpireBlock() {

	for key, blk := range vm.pendingBlock {

		if blk.Height <= vm.acceptHeight {
			delete(vm.pendingBlock, key)
		}
	}
}
