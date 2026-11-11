package dht

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
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
