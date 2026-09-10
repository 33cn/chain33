// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cggmp

import (
	"testing"
	"time"

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
