package tss

import (
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/require"
)

func TestIsValidRankCombination(t *testing.T) {

	// Let threshold = 3, and participants = 4. Assume that the corresponding rank of each shareholder are 0, 1, 1, 2.
	require.True(t, isValidRankCombination([]uint32{0, 1, 1}, 3))
	require.True(t, isValidRankCombination([]uint32{0, 1, 2}, 3))
	require.True(t, isValidRankCombination([]uint32{0, 1, 1, 2}, 3))
	require.False(t, isValidRankCombination([]uint32{1, 1, 2}, 3))
	require.True(t, isValidRankCombination([]uint32{0, 1}, 2))
	require.False(t, isValidRankCombination([]uint32{0, 1}, 3))
}

// TestSelectSignerSubset pins the signer-subset contract ProcessSign relies on: exactly threshold
// peers, ordered by rank, always including the local node, and only when the Birkhoff rule holds.
func TestSelectSignerSubset(t *testing.T) {
	bks := map[string]*BK{
		"rank0":  {Rank: 0},
		"rank1a": {Rank: 1},
		"rank1b": {Rank: 1},
		"rank1c": {Rank: 1},
	}
	peer := func(name string, self bool) *types.Peer { return &types.Peer{Name: name, Self: self} }

	t.Run("lowest ranks, exactly threshold", func(t *testing.T) {
		got := selectSignerSubset([]*types.Peer{
			peer("rank1c", false), peer("rank0", true), peer("rank1a", false), peer("rank1b", false),
		}, 3, bks)
		require.Len(t, got, 3)
		require.Equal(t, "rank0", got[0])
	})

	t.Run("self is swapped in", func(t *testing.T) {
		// self has the highest rank in the full set, so it is not in the lowest-threshold pick.
		got := selectSignerSubset([]*types.Peer{
			peer("rank1a", false), peer("rank1b", false), peer("rank0", false), peer("rank1c", true),
		}, 3, bks)
		require.Len(t, got, 3)
		require.Contains(t, got, "rank1c")
	})

	t.Run("no valid combination", func(t *testing.T) {
		// Two rank-1 peers and threshold 2: sorted ranks {1,1} violate rank_0 <= 0.
		require.Nil(t, selectSignerSubset([]*types.Peer{peer("rank1a", false), peer("rank1b", true)}, 2, bks))
	})

	t.Run("not enough candidates", func(t *testing.T) {
		require.Nil(t, selectSignerSubset([]*types.Peer{peer("rank0", true)}, 2, bks))
	})
}
