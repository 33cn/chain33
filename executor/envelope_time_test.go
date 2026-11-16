package executor

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/33cn/chain33/types"
)

// TestEnvelopeBlockTime pins where the envelope window is measured from: the header time
// during block execution, the local clock (zero) when the mempool pre-checks against a
// header that may be minutes old.
func TestEnvelopeBlockTime(t *testing.T) {
	block := &executor{blocktime: 1793848388, ctx: &executorCtx{}}
	require.Equal(t, int64(1793848388), block.envelopeBlockTime())

	pool := &executor{blocktime: 1793848388, ctx: &executorCtx{isMempool: true}}
	require.Equal(t, int64(0), pool.envelopeBlockTime())
}

// TestNewExecTxListCtxCarriesMempoolFlag pins the wiring from the queue message to the
// executor: the mempool flag decides whether the envelope window uses the local clock, and
// dropping it here would silently make the mempool measure against a stale header time.
func TestNewExecTxListCtxCarriesMempoolFlag(t *testing.T) {
	ctx := newExecTxListCtx(&types.ExecTxList{
		StateHash:  []byte{1},
		Height:     12,
		BlockTime:  1793848388,
		Difficulty: 3,
		MainHash:   []byte{2},
		MainHeight: 11,
		ParentHash: []byte{3},
		IsMempool:  true,
	})
	require.True(t, ctx.isMempool)
	require.Equal(t, int64(1793848388), ctx.blocktime)

	require.False(t, newExecTxListCtx(&types.ExecTxList{}).isMempool)
}
