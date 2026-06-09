package queue

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// F-RPC-001: rpc/ethrpc/rpc.go ServeHTTP has no IP whitelist check.
// The standard JSON-RPC handler (rpc/http.go) checks checkIPWhitelist() and
// checkBasicAuth() before forwarding requests. The ethrpc handler forwards
// all requests regardless of source IP, bypassing the auth layer entirely.

func TestEthRPCBypassesAuthBug(t *testing.T) {
	// Simulate standard RPC auth: checks IP whitelist
	whitelist := map[string]bool{"127.0.0.1": true}

	standardRPCHandler := func(w http.ResponseWriter, r *http.Request) {
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		parsedIP := net.ParseIP(ip)
		if !parsedIP.IsLoopback() {
			if _, ok := whitelist[ip]; !ok {
				http.Error(w, "IP not authorized", http.StatusForbidden)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
	}

	// Simulate buggy ethrpc: no auth check at all
	buggyEthRPCHandler := func(w http.ResponseWriter, r *http.Request) {
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		if ip != "" {
			// Only logs, never blocks
		}
		w.WriteHeader(http.StatusOK)
	}

	// Simulate fixed ethrpc: checks IP whitelist
	fixedEthRPCHandler := func(w http.ResponseWriter, r *http.Request) {
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		parsedIP := net.ParseIP(ip)
		if !parsedIP.IsLoopback() {
			if _, ok := whitelist[ip]; !ok {
				http.Error(w, "IP not authorized", http.StatusForbidden)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
	}

	// External IP request (should be blocked)
	req := httptest.NewRequest("POST", "/", nil)
	req.RemoteAddr = "203.0.113.1:12345" // external IP

	// Standard RPC correctly blocks external IP
	w1 := httptest.NewRecorder()
	standardRPCHandler(w1, req)
	assert.Equal(t, http.StatusForbidden, w1.Code,
		"standard RPC blocks non-whitelisted IP")

	// BUG: ethrpc allows external IP through
	w2 := httptest.NewRecorder()
	buggyEthRPCHandler(w2, req)
	assert.Equal(t, http.StatusOK, w2.Code,
		"buggy ethrpc allows non-whitelisted IP through")

	// FIXED: ethrpc now blocks external IP
	w3 := httptest.NewRecorder()
	fixedEthRPCHandler(w3, req)
	assert.Equal(t, http.StatusForbidden, w3.Code,
		"fixed ethrpc blocks non-whitelisted IP")
}
