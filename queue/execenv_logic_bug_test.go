package queue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// F-EXEC-009: executor/execenv.go:360 uses && but should use ||
// Comment says "均不相等时" (when either has changed), but && requires BOTH to differ.
// If only tx changes (same index) or only index changes (same tx), cache won't update.

func TestExecEnvCacheUpdateLogicBug(t *testing.T) {
	type fakeTx struct{ id int }

	tx1 := &fakeTx{1}
	tx2 := &fakeTx{2}

	// Simulate the buggy condition: e.currExecTx != tx && e.currTxIdx != index
	// Current state: currTx=tx1, currIdx=0
	currTx := tx1
	currIdx := 0

	// Case: new tx at SAME index (tx changes, index stays)
	newTx := tx2
	newIdx := 0

	buggyCondition := (currTx != newTx) && (currIdx != newIdx)
	fixedCondition := (currTx != newTx) || (currIdx != newIdx)

	// BUG: buggy version does NOT update cache even though tx changed
	assert.False(t, buggyCondition, "buggy && fails to detect tx change at same index")
	// FIXED: || correctly detects the change
	assert.True(t, fixedCondition, "fixed || detects tx change at same index")

	// Case: same tx at DIFFERENT index (reused tx object, different position)
	currTx = tx1
	currIdx = 0
	newTx2 := tx1 // same tx pointer
	newIdx2 := 5  // different index

	buggyCondition2 := (currTx != newTx2) && (currIdx != newIdx2)
	fixedCondition2 := (currTx != newTx2) || (currIdx != newIdx2)

	// BUG: buggy version does NOT update cache even though index changed
	assert.False(t, buggyCondition2, "buggy && fails to detect index change with same tx")
	// FIXED: || correctly detects the change
	assert.True(t, fixedCondition2, "fixed || detects index change with same tx")
}
