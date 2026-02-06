package tss

import (
	"testing"

	"github.com/stretchr/testify/require"
)





func TestIsValidRankCombination(t *testing.T) {

	// Let threshold = 3, and participants = 4. Assume that the corresponding rank of each shareholder are 0, 1, 1, 2. 
	require.True(t, isValidRankCombination([]uint32{0, 1, 1}, 3))
	require.True(t, isValidRankCombination([]uint32{0, 1, 2}, 3))
	require.True(t, isValidRankCombination([]uint32{0, 1,1,2}, 3))
	require.False(t, isValidRankCombination([]uint32{1,1,2}, 3))
	require.True(t, isValidRankCombination([]uint32{0, 1}, 2))
	require.False(t, isValidRankCombination([]uint32{0, 1}, 3))
}
