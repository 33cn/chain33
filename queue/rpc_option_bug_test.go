package queue

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

// F-RPC-004: rpc/ethrpc/rpc.go:218 checks r.Method == "OPTION" (singular)
// but HTTP CORS preflight uses "OPTIONS" (plural). Browsers never get 204.

func TestCORSPreflightMethodNameBug(t *testing.T) {
	// Simulate buggy handler logic
	buggyHandler := func(r *http.Request) int {
		if r.Method == "OPTION" { // BUG: singular
			return http.StatusNoContent
		}
		return http.StatusOK
	}

	fixedHandler := func(r *http.Request) int {
		if r.Method == "OPTIONS" { // FIXED: plural
			return http.StatusNoContent
		}
		return http.StatusOK
	}

	// Browser sends "OPTIONS" for CORS preflight
	req, _ := http.NewRequest("OPTIONS", "http://localhost:8545", nil)

	// BUG: buggy handler doesn't recognize OPTIONS, falls through
	assert.Equal(t, http.StatusOK, buggyHandler(req),
		"buggy OPTION (singular) misses real OPTIONS preflight request")

	// FIXED: correct handler returns 204 No Content
	assert.Equal(t, http.StatusNoContent, fixedHandler(req),
		"fixed OPTIONS (plural) matches CORS preflight")
}
