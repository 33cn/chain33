package executor

import (
	"testing"

	"github.com/stretchr/testify/require"
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
