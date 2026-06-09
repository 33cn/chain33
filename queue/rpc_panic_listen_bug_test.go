package queue

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

// F-RPC-005: rpc/ethrpc/rpc.go:188 calls panic(err) when net.Listen fails.
// A production server should return the error, not crash the process.

func TestListenFailurePanicBug(t *testing.T) {
	// Occupy a port so the second listen will fail
	l, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	defer l.Close()
	occupiedAddr := l.Addr().String()

	// Simulate buggy Start() behavior: panic on listen failure
	buggyStart := func() (int, error) {
		_, err := net.Listen("tcp", occupiedAddr)
		if err != nil {
			panic(err) // BUG: crashes process
		}
		return 0, nil
	}

	// Simulate fixed Start() behavior: return error
	fixedStart := func() (int, error) {
		_, err := net.Listen("tcp", occupiedAddr)
		if err != nil {
			return 0, err // FIXED: propagate error
		}
		return 0, nil
	}

	// BUG: buggy version panics
	assert.Panics(t, func() { buggyStart() },
		"buggy Start() panics when port is occupied")

	// FIXED: fixed version returns error gracefully
	port, err := fixedStart()
	assert.Error(t, err, "fixed Start() returns error when port is occupied")
	assert.Equal(t, 0, port)
}
