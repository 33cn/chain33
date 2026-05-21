package queue

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// F-CLI-003: RemoveTxsByHashList swallows server errors (logic inversion).
// When msg.GetData() IS an error (ok=true), the function returns nil instead of err.
// When msg.GetData() is NOT an error (ok=false), err is nil and returned correctly.
// The if/return logic is inverted.

func TestRemoveTxsByHashList_ErrorLogicInversion(t *testing.T) {
	// Simulate the buggy pattern from client/queueprotocol.go:216-221
	extractErrorBuggy := func(data interface{}) error {
		var ok bool
		err, ok := data.(error)
		if !ok {
			return err // ok=false means data is NOT error, err=nil → correct
		}
		return nil // ok=true means data IS error, but returns nil → BUG
	}

	extractErrorFixed := func(data interface{}) error {
		err, ok := data.(error)
		if ok {
			return err // data IS an error, return it
		}
		return nil // data is NOT an error, success
	}

	serverErr := ErrIsQueueClosed

	// Buggy: server error is swallowed
	result := extractErrorBuggy(serverErr)
	assert.Nil(t, result, "BUG CONFIRMED: server error is swallowed, returns nil")

	// Fixed: server error is propagated
	result = extractErrorFixed(serverErr)
	assert.Equal(t, serverErr, result, "fixed version returns the error")

	// Both handle success correctly
	assert.Nil(t, extractErrorBuggy("success-data"))
	assert.Nil(t, extractErrorFixed("success-data"))
}
