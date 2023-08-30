package snowman

import (
	"fmt"
	"time"

	"github.com/33cn/chain33/types"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/choices"
	"github.com/ava-labs/avalanchego/snow/consensus/snowman"
)

// wrap chain33 block for implementing the snowman.Block interface
type snowBlock struct {
	id     ids.ID
	block  *types.Block
	status choices.Status
}

func newSnowBlock(blk *types.Block, cfg *types.Chain33Config) snowman.Block {

	sb := &snowBlock{block: blk}
	copy(sb.id[:], blk.Hash(cfg))
	return sb
}

// ID implements the snowman.Block interface
func (b *snowBlock) ID() ids.ID { return b.id }

// Accept implements the snowman.Block interface
func (b *snowBlock) Accept() error {

	b.status = choices.Accepted
	log.Debug(fmt.Sprintf("Accepting block %s at height %d", b.ID().Hex(), b.Height()))
	// TODO accept block
	return nil
}

// Reject implements the snowman.Block interface
func (b *snowBlock) Reject() error {
	b.status = choices.Rejected
	log.Debug(fmt.Sprintf("Rejecting block %s at height %d", b.ID().Hex(), b.Height()))
	// TODO reject block
	return nil
}

// SetStatus implements the InternalBlock interface allowing ChainState
// to set the status on an existing block
func (b *snowBlock) SetStatus(status choices.Status) { b.status = status }

// Status implements the snowman.Block interface
func (b *snowBlock) Status() choices.Status {
	return b.status
}

// Parent implements the snowman.Block interface
func (b *snowBlock) Parent() (id ids.ID) {

	copy(id[:], b.block.ParentHash)
	return
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
func (b *snowBlock) Verify() error {

	log.Debug(fmt.Sprintf("Verify block %s at height %d", b.ID().Hex(), b.Height()))
	// TODO verify block
	return nil
}

// Bytes implements the snowman.Block interface
func (b *snowBlock) Bytes() []byte {
	return types.Encode(b.block)
}

func (b *snowBlock) String() string { return fmt.Sprintf("chain33 block, hash = %s", b.ID().Hex()) }
