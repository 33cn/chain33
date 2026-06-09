package mavl

import (
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

func TestVerifyKVPairProof_NilProofPanic(t *testing.T) {
	// F-MAVL-001: When ReadProof fails, proofnode is nil but code
	// continues to call proofnode.Verify() causing nil dereference panic.
	// Passing invalid/empty proof bytes triggers ReadProof error.

	roothash := []byte("fake-root-hash-32-bytes-long!!!!")
	kv := &types.KeyValue{Key: []byte("testkey"), Value: []byte("testval")}
	invalidProof := []byte("invalid-proof-data")

	// This should NOT panic, but currently does due to nil dereference
	assert.NotPanics(t, func() {
		result := VerifyKVPairProof(nil, roothash, kv, invalidProof)
		assert.False(t, result)
	}, "BUG: VerifyKVPairProof panics on invalid proof (nil dereference)")
}
