package dht

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	sec "github.com/libp2p/go-libp2p/core/sec"
	"github.com/libp2p/go-libp2p/p2p/net/swarm"
	"github.com/stretchr/testify/require"
)

// TestLanDialInFlight covers the duplicate-announcement guard: a second begin for the same peer
// must be refused while the dial is in flight, and a new dial may start after the first finished.
func TestLanDialInFlight(t *testing.T) {
	var f lanDialInFlight
	id := peer.ID("peer-a")

	require.True(t, f.begin(id))
	require.False(t, f.begin(id), "a repeated mDNS announcement must not start another dial")

	// Other peers are unaffected.
	require.True(t, f.begin(peer.ID("peer-b")))
	f.end(peer.ID("peer-b"))

	f.end(id)
	require.True(t, f.begin(id), "a dial may start again once the previous one finished")
	f.end(id)
}

// TestLanDialInFlightConcurrent asserts LoadOrStore semantics under concurrent announcements:
// exactly one caller wins the dial for a peer.
func TestLanDialInFlightConcurrent(t *testing.T) {
	var f lanDialInFlight
	id := peer.ID("peer-a")
	const goroutines = 64

	var started atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if f.begin(id) {
				started.Add(1)
			}
		}()
	}
	wg.Wait()
	require.Equal(t, int32(1), started.Load())
}

// TestIsPermanentDialError pins the error classification that decides whether connectLANPeer
// gives up immediately or keeps retrying. Only errors that cannot heal for the same address set
// may be classified as permanent; transport failures and timeouts must stay retryable.
func TestIsPermanentDialError(t *testing.T) {
	mismatch := sec.ErrPeerIDMismatch{Expected: peer.ID("peer-a"), Actual: peer.ID("peer-b")}
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"dial to self", swarm.ErrDialToSelf, true},
		{"no addresses", swarm.ErrNoAddresses, true},
		{"no remote addresses", network.ErrNoRemoteAddrs, true},
		{"invalid multiaddr", peer.ErrInvalidAddr, true},
		{"peer id mismatch", mismatch, true},
		{"wrapped peer id mismatch", fmt.Errorf("handshake failed: %w", mismatch), true},
		{"all dials failed", fmt.Errorf("all dials failed: %w", swarm.ErrAllDialsFailed), false},
		{"transport error", &swarm.TransportError{Cause: errors.New("connection refused")}, false},
		{"context canceled", context.Canceled, false},
		{
			// The regression this classification exists for: one mDNS announcement lists loopback
			// next to the routable address, so the dial to self must not hide the retryable TLS
			// simultaneous-open race recorded for the other address.
			"self + retryable address",
			&swarm.DialError{
				Cause: swarm.ErrAllDialsFailed,
				DialErrors: []swarm.TransportError{
					{Cause: swarm.ErrDialToSelf},
					{Cause: errors.New("failed to negotiate security protocol: tls: unexpected message")},
				},
			},
			false,
		},
		{
			"every address permanent",
			&swarm.DialError{
				Cause: swarm.ErrAllDialsFailed,
				DialErrors: []swarm.TransportError{
					{Cause: swarm.ErrDialToSelf},
					{Cause: peer.ErrInvalidAddr},
				},
			},
			true,
		},
		{
			// libp2p uses this cause when the address filter removed everything undialable; the
			// address-level verdict still says "dialing ourselves can never work".
			"no good addresses, all self",
			&swarm.DialError{
				Cause:      swarm.ErrNoGoodAddresses,
				DialErrors: []swarm.TransportError{{Cause: swarm.ErrDialToSelf}},
			},
			true,
		},
		{
			"peer id mismatch on every address",
			&swarm.DialError{
				Cause:      swarm.ErrAllDialsFailed,
				DialErrors: []swarm.TransportError{{Cause: mismatch}},
			},
			true,
		},
		{
			// The recorded list is capped; the dropped entry may be the retryable one, so a
			// truncated list cannot prove "every address is permanent".
			"truncated address list",
			&swarm.DialError{
				Cause:      swarm.ErrAllDialsFailed,
				Skipped:    3,
				DialErrors: []swarm.TransportError{{Cause: swarm.ErrDialToSelf}},
			},
			false,
		},
		{
			"no address dialed, permanent cause",
			&swarm.DialError{Cause: swarm.ErrNoAddresses},
			true,
		},
		{
			"no address dialed, retryable cause",
			&swarm.DialError{Cause: errors.New("connection gater disallowed the connection")},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, isPermanentDialError(tt.err))
		})
	}
}
