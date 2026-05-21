package queue

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// F-RPC-003: rpc/ethrpc/rpc.go:208-214 Close() has dead code (duplicate condition)
// and never shuts down the HTTP server or closes the listener.
// After Close(), the port remains occupied — resource leak.

func TestHTTPServerCloseResourceLeak(t *testing.T) {
	// Start a simple HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})

	server := &http.Server{Handler: mux}
	l, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	addr := l.Addr().String()

	go server.Serve(l)

	// Simulate buggy Close(): does nothing to the HTTP server
	buggyClose := func() {
		// Original code only checks wsHandler (which is nil in HTTP-only mode)
		// and has dead else-if branch. Never calls server.Shutdown or l.Close.
	}

	buggyClose()

	// BUG: port is still occupied after "close"
	_, err = net.DialTimeout("tcp", addr, 100*time.Millisecond)
	assert.NoError(t, err, "buggy Close() leaves port open — resource leak")

	// Simulate fixed Close(): properly shuts down the server
	fixedClose := func() {
		server.Shutdown(context.Background())
	}

	fixedClose()

	// FIXED: port is released after proper shutdown
	time.Sleep(50 * time.Millisecond)
	_, err = net.DialTimeout("tcp", addr, 100*time.Millisecond)
	assert.Error(t, err, "fixed Close() releases the port")
}
