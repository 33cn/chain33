// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cggmp

import (
	"math/big"
	"testing"
	"time"

	"github.com/getamis/alice/crypto/ecpointgrouplaw"
	"github.com/getamis/alice/crypto/elliptic"
	"github.com/stretchr/testify/require"
)

// TestPhaseTimeouts asserts the per-phase default timeouts (refresh, which generates a 2048-bit
// Paillier key per node, gets a much larger default than the other phases) and that an explicit
// WithTimeout still overrides them all.
func TestPhaseTimeouts(t *testing.T) {
	cfg := newConfig()
	require.Equal(t, defaultTimeout, cfg.phaseTimeout(DkgProtocol))
	require.Equal(t, defaultTimeout, cfg.phaseTimeout(PpkProtocol))
	require.Equal(t, defaultTimeout, cfg.phaseTimeout(SignProtocol))
	require.Equal(t, refreshDefaultTimeout, cfg.phaseTimeout(RefreshProtocol))
	require.Equal(t, 5*time.Minute, refreshDefaultTimeout)
	require.Greater(t, refreshDefaultTimeout, defaultTimeout)

	override := newConfig(WithTimeout(7 * time.Second))
	require.Equal(t, 7*time.Second, override.phaseTimeout(DkgProtocol))
	require.Equal(t, 7*time.Second, override.phaseTimeout(RefreshProtocol))

	// A non-positive WithTimeout keeps its "no timeout" meaning (tss.NewListener has no deadline).
	noTimeout := newConfig(WithTimeout(0))
	require.Equal(t, time.Duration(0), noTimeout.phaseTimeout(DkgProtocol))
	require.Equal(t, time.Duration(0), noTimeout.phaseTimeout(RefreshProtocol))
}

// TestProcessRejectsInvalidRankBeforePeerDiscovery pins the entry check: alice's EnsureRank
// rejects rank >= threshold, and the wrapper must reject it before creating a peer manager, which
// would otherwise block until the phase timeout. No peers are connected in this test, so a
// blocking implementation would hang instead of returning.
func TestProcessRejectsInvalidRankBeforePeerDiscovery(t *testing.T) {
	peers := []string{"peer-a", "peer-b", "peer-c"}
	// alice's EnsureRank rejects rank+1 >= threshold, so threshold 3 allows only ranks 0 and 1:
	// rank 2 is the boundary case that a "rank >= threshold" check would let through.
	for _, rank := range []uint32{2, 3} {
		_, err := ProcessDKG(peers, 3, rank, "cggmp-rank-check", WithTimeout(50*time.Millisecond))
		require.ErrorIs(t, err, errInvalidRank, "rank %d", rank)
	}
}

// TestBeginSessionRunRejectsConcurrent asserts one session id cannot be driven by two Process*
// calls at once: the second would reset the first one's ppk collector.
func TestBeginSessionRunRejectsConcurrent(t *testing.T) {
	const sessionID = "cggmp-session-single-flight"
	require.NoError(t, beginSessionRun(DkgProtocol, sessionID))
	require.ErrorIs(t, beginSessionRun(DkgProtocol, sessionID), errSessionRunning)

	// Teardown releases the claim so a retry under the same id works again.
	endSessionRun(DkgProtocol, sessionID)
	require.NoError(t, beginSessionRun(DkgProtocol, sessionID))
	endSessionRun(DkgProtocol, sessionID)
}

// TestPPKCollectorWaitNonPositiveTimeout is the regression test for the "non-positive timeout means
// no timeout" contract: time.After(0) fires immediately, so a collector that used it would fail the
// ppk phase the moment the caller passed WithTimeout(0). The wait must stay blocked until the last
// participant's point arrives.
func TestPPKCollectorWaitNonPositiveTimeout(t *testing.T) {
	c := newPPKCollector([]byte("rid"), []string{"peer-b"})

	done := make(chan error, 1)
	go func() { done <- c.wait(0) }()

	select {
	case err := <-done:
		t.Fatalf("wait(0) returned before the last point arrived: %v", err)
	case <-time.After(50 * time.Millisecond):
	}

	c.add("peer-b", []byte("rid"), ecpointgrouplaw.ScalarBaseMult(elliptic.Secp256k1(), big.NewInt(7)))
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("wait did not return after the last point arrived")
	}
}
