package snowman

import (
	"context"
	"fmt"
	"time"

	"github.com/33cn/chain33/types"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/choices"
)

// wrap chain33 block for implementing the snowman.Block interface
type snowBlock struct {
	id     ids.ID
	parent ids.ID
	block  *types.Block
	status choices.Status
	vm     *chain33VM
}

// ID implements the snowman.Block interface
func (b *snowBlock) ID() ids.ID { return b.id }

// Accept implements the snowman.Block interface
func (b *snowBlock) Accept(ctx context.Context) error {

	b.status = choices.Accepted
	snowLog.Debug("snowBlock accept", "hash", b.id.Hex(), "height", b.Height(), "parent", b.Parent().Hex())
	err := b.vm.acceptBlock(b.block.Height, b.id)
	if err != nil {
		snowLog.Error("Accepting block error", "hash", b.id.Hex(), "height", b.Height())
	}
	return nil
}

// Reject implements the snowman.Block interface
// This element will not be accepted by any correct node in the network.
func (b *snowBlock) Reject(ctx context.Context) error {
	b.status = choices.Rejected
	snowLog.Debug("snowBlock reject", "hash", b.ID().Hex(), "height", b.Height(), "parent", b.Parent().Hex())
	b.vm.rejectBlock(b.block.Height, b.ID())
	return nil
}

// SetStatus implements the InternalBlock interface allowing ChainState
// to set the status on an existing block
//func (b *snowBlock) SetStatus(status choices.Status) { b.status = status }

// Status implements the snowman.Block interface
func (b *snowBlock) Status() choices.Status {
	return b.status
}

// Parent implements the snowman.Block interface
func (b *snowBlock) Parent() ids.ID {
	return b.parent
}

// Height implements the snowman.Block interface
func (b *snowBlock) Height() uint64 {
	return uint64(b.block.Height)
}

// Timestamp implements the snowman.Block interface
func (b *snowBlock) Timestamp() time.Time {
	return time.Unix(b.block.BlockTime, 0)
}

// Verify implements the snowman.Block interface
func (b *snowBlock) Verify(ctx context.Context) error {

	snowLog.Debug(fmt.Sprintf("Verify block %s at height %d", b.ID().Hex(), b.Height()))
	// TODO verify block
	return nil
}

// Bytes implements the snowman.Block interface
func (b *snowBlock) Bytes() []byte {
	return types.Encode(b.block)
}

func (b *snowBlock) String() string { return fmt.Sprintf("chain33 block, hash = %s", b.ID().Hex()) }

func toSnowID(blkHash []byte) (id ids.ID) {
	copy(id[:], blkHash)
	return
}
