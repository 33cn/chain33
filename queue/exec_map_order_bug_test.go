package queue

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

// F-EXEC-001: executor/executor.go:416,516 iterates globalPlugins map
// without sorting keys. Map iteration in Go is non-deterministic, so
// KV pairs appended from plugins arrive in random order — consensus-breaking
// in a blockchain where all nodes must produce identical state.

func TestMapIterationNonDeterminism(t *testing.T) {
	plugins := map[string]int{
		"fee":       1,
		"stat":      2,
		"mvcc":      3,
		"txindex":   4,
		"addrindex": 5,
		"addrfee":   6,
	}

	// Run buggy iteration (direct map range) multiple times
	// and check if order is always the same
	buggyResults := make([][]string, 100)
	for i := 0; i < 100; i++ {
		var order []string
		for name := range plugins {
			order = append(order, name)
		}
		buggyResults[i] = order
	}

	// Check if any two runs produced different orders
	foundDifference := false
	for i := 1; i < 100; i++ {
		if len(buggyResults[0]) != len(buggyResults[i]) {
			foundDifference = true
			break
		}
		for j := range buggyResults[0] {
			if buggyResults[0][j] != buggyResults[i][j] {
				foundDifference = true
				break
			}
		}
		if foundDifference {
			break
		}
	}
	// Note: Go randomizes map iteration, so this should find differences
	// In rare cases it might not, but with 6 keys and 100 iterations it's near-certain
	t.Logf("Map iteration produced different orders: %v", foundDifference)

	// Fixed version: sort keys first — always deterministic
	fixedResults := make([][]string, 100)
	for i := 0; i < 100; i++ {
		names := make([]string, 0, len(plugins))
		for name := range plugins {
			names = append(names, name)
		}
		sort.Strings(names)
		fixedResults[i] = names
	}

	// All fixed runs must produce identical order
	for i := 1; i < 100; i++ {
		assert.Equal(t, fixedResults[0], fixedResults[i],
			"sorted iteration must be deterministic across all runs")
	}

	// Verify the sorted order is what we expect
	expected := []string{"addrfee", "addrindex", "fee", "mvcc", "stat", "txindex"}
	assert.Equal(t, expected, fixedResults[0])
}
