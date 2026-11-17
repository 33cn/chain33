// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cggmp

import (
	"crypto/sha256"
	"math/big"
	"testing"

	"github.com/getamis/alice/crypto/birkhoffinterpolation"
	"github.com/getamis/alice/crypto/ecpointgrouplaw"
	"github.com/getamis/alice/crypto/elliptic"
	alicetss "github.com/getamis/alice/crypto/tss"
	alicedkg "github.com/getamis/alice/crypto/tss/ecdsa/cggmp/dkg"
	alicerefresh "github.com/getamis/alice/crypto/tss/ecdsa/cggmp/refresh"
	alicesign "github.com/getamis/alice/crypto/tss/ecdsa/cggmp/sign"
	alicetypes "github.com/getamis/alice/types"
	"github.com/stretchr/testify/require"
)

// testPrevRefresh builds a previous-refresh result whose ppk material is the (valid) DKG
// generation. The validation tests below only need material that reconstructs the group key;
// the refresh round itself is covered by TestCGGMPRefreshTwiceInProcess.
func testPrevRefresh(d *DKGResult, peers []string) *RefreshResult {
	r := &RefreshResult{
		Share:          append([]byte(nil), d.Share...),
		PartialPubKeys: make(map[string]*ECPoint, len(peers)),
		Ssid:           []byte("previous-refresh-ssid"),
	}
	for _, id := range peers {
		r.PartialPubKeys[id] = d.PartialPubKeys[id]
	}
	return r
}

// TestProcessRefreshAgainRejectsMismatchedInput covers the fail-closed entry checks: a previous
// refresh result that belongs to another participant set, misses material, or cannot derive the
// next zero-knowledge domain must be rejected instead of being silently narrowed or reused.
// Every case returns before a peer manager is created, so no transport is needed.
func TestProcessRefreshAgainRejectsMismatchedInput(t *testing.T) {
	newDKG := func() (*DKGResult, []string) {
		d, peers := testTwoPartyDKGResult()
		d.Threshold = 2
		d.Peers = append([]string(nil), peers...)
		return d, peers
	}

	dkgRes, peers := newDKG()
	require.NoError(t, testPrevRefresh(dkgRes, peers).validateRefreshInput(dkgRes, peers, 2))

	t.Run("nil previous result", func(t *testing.T) {
		d, p := newDKG()
		_, err := ProcessRefreshAgain(p, 2, d, nil, "refresh-again-session")
		require.ErrorIs(t, err, errNilRefreshResult)
	})

	t.Run("previous result from a wider participant set", func(t *testing.T) {
		d, p := newDKG()
		prev := testPrevRefresh(d, p)
		prev.PartialPubKeys["peer-extra"] = prev.PartialPubKeys["peer-rank0"]
		_, err := ProcessRefreshAgain(p, 2, d, prev, "refresh-again-session")
		require.ErrorIs(t, err, errRefreshResultMismatch)
		require.Contains(t, err.Error(), "peer-extra")
	})

	t.Run("previous result missing a participant", func(t *testing.T) {
		d, p := newDKG()
		prev := testPrevRefresh(d, p)
		delete(prev.PartialPubKeys, "peer-rank1")
		_, err := ProcessRefreshAgain(p, 2, d, prev, "refresh-again-session")
		require.ErrorIs(t, err, errMissingPeerPPK)
		require.Contains(t, err.Error(), "peer-rank1")
	})

	t.Run("previous result without ssid", func(t *testing.T) {
		d, p := newDKG()
		prev := testPrevRefresh(d, p)
		prev.Ssid = nil
		_, err := ProcessRefreshAgain(p, 2, d, prev, "refresh-again-session")
		require.ErrorIs(t, err, errMissingRefreshSsid)
	})

	t.Run("previous result with inconsistent ppk", func(t *testing.T) {
		d, p := newDKG()
		prev := testPrevRefresh(d, p)
		prev.PartialPubKeys["peer-rank1"] = fromECPoint(
			ecpointgrouplaw.ScalarBaseMult(elliptic.Secp256k1(), big.NewInt(99)))
		_, err := ProcessRefreshAgain(p, 2, d, prev, "refresh-again-session")
		require.ErrorIs(t, err, errPPKInconsistent)
	})

	t.Run("dkg threshold mismatch", func(t *testing.T) {
		d, p := newDKG()
		d.Threshold = 3
		_, err := ProcessRefreshAgain(p, 2, d, testPrevRefresh(d, p), "refresh-again-session")
		require.ErrorIs(t, err, errDKGResultMismatch)
	})
}

// TestRefreshSelfShareCheck pins the generation-consistency check run before the round starts:
// the local share and the local partial public key must encode the same point.
func TestRefreshSelfShareCheck(t *testing.T) {
	curve := elliptic.Secp256k1()
	share := big.NewInt(7)
	ppk := ecpointgrouplaw.ScalarBaseMult(curve, share)

	require.NoError(t, checkSelfShare(curve, share, ppk, "self"))

	err := checkSelfShare(curve, big.NewInt(8), ppk, "self")
	require.ErrorIs(t, err, errRefreshShareMismatch)
	require.Contains(t, err.Error(), "self")

	require.ErrorIs(t, checkSelfShare(curve, share, nil, "self"), errMissingPeerPPK)
}

// TestComputeRefreshAgainSSID pins the per-round domain separation: chaining the previous ssid
// yields a distinct value for every refresh, deterministic for all participants of one round, and
// different from the first-refresh ssid.
func TestComputeRefreshAgainSSID(t *testing.T) {
	rid := []byte("dkg-rid")
	first := computeSSID("refresh-session", rid)
	second := computeRefreshAgainSSID("refresh-session", first, rid)
	third := computeRefreshAgainSSID("refresh-session", second, rid)

	require.NotEqual(t, first, second)
	require.NotEqual(t, second, third)
	require.Equal(t, second, computeRefreshAgainSSID("refresh-session", first, rid),
		"every participant must derive the same ssid")
}

// runInProcessDKG drives alice's DKG core for n in-process parties (the same construction
// e2e_test.go uses) and returns one persisted DKGResult per node, each carrying the full ppk map.
func runInProcessDKG(t *testing.T, n int, threshold uint32) []*DKGResult {
	t.Helper()
	curve := elliptic.Secp256k1()
	ids := make([]string, n)
	for i := 0; i < n; i++ {
		ids[i] = alicetss.GetTestID(i)
	}
	coreMap := make(map[string]alicetypes.MessageMain, n)
	cores := make([]*alicedkg.DKG, n)
	listeners := make([]*phaseListener, n)
	for i := 0; i < n; i++ {
		pm := alicetss.NewTestPeerManager(i, n)
		pm.Set(coreMap)
		l := newPhaseListener()
		core, err := alicedkg.NewDKG(curve, pm, []byte("refresh-again-dkg-session"), threshold, 0, l)
		require.NoError(t, err)
		coreMap[ids[i]] = core
		cores[i] = core
		listeners[i] = l
	}
	for _, c := range cores {
		c.Start()
	}
	for _, l := range listeners {
		l.wait(t)
	}

	allPPK := make(map[string]*ECPoint, n)
	results := make([]*DKGResult, n)
	for i := 0; i < n; i++ {
		res, err := cores[i].GetResult()
		require.NoError(t, err)
		cores[i].Stop()
		allPPK[ids[i]] = partialPublicKeyFor(curve, res.Share)
		results[i] = newDKGResult(res, nil, threshold, ids)
	}
	for i := 0; i < n; i++ {
		results[i].PartialPubKeys = make(map[string]*ECPoint, n)
		for j := 0; j < n; j++ {
			results[i].PartialPubKeys[ids[j]] = allPPK[ids[j]]
		}
		results[i] = jsonRoundTrip(t, results[i])
	}
	return results
}

// runInProcessRefresh drives one alice refresh round for n in-process parties and returns the
// persisted per-node results. shares/ppks are the per-node inputs; bks and pubKey are the
// immutable key parameters.
func runInProcessRefresh(t *testing.T, ids []string, threshold uint32, shares []*big.Int,
	ppks []map[string]*ecpointgrouplaw.ECPoint, bks map[string]*birkhoffinterpolation.BkParameter,
	pubKey *ecpointgrouplaw.ECPoint, ssid []byte) []*RefreshResult {
	t.Helper()
	coreMap := make(map[string]alicetypes.MessageMain, len(ids))
	cores := make([]*alicerefresh.Refresh, len(ids))
	listeners := make([]*phaseListener, len(ids))
	for i := range ids {
		pm := alicetss.NewTestPeerManager(i, len(ids))
		pm.Set(coreMap)
		l := newPhaseListener()
		core, err := alicerefresh.NewRefresh(shares[i], pubKey, pm, threshold,
			ppks[i], bks, 2048, ssid, l)
		require.NoError(t, err)
		coreMap[ids[i]] = core
		cores[i] = core
		listeners[i] = l
	}
	for _, c := range cores {
		c.Start()
	}
	for _, l := range listeners {
		l.wait(t)
	}
	results := make([]*RefreshResult, len(ids))
	for i := range ids {
		res, err := cores[i].GetResult()
		require.NoError(t, err)
		cores[i].Stop()
		results[i] = jsonRoundTrip(t, newRefreshResult(res, ssid))
	}
	return results
}

// TestCGGMPRefreshTwiceInProcess proves that the same key can be refreshed twice in a row on
// alice v1.0.7: the second round is seeded with the first refresh's share and partial public keys
// (refreshInputs), produces a new generation, and that generation signs with a signature that
// verifies against the DKG group key.
//
// The DKG result's own share / partial public keys are corrupted before the second round, so the
// test only passes if the previous refresh result is really the input source.
func TestCGGMPRefreshTwiceInProcess(t *testing.T) {
	if testing.Short() {
		t.Skip("skip cggmp crypto e2e in short mode")
	}
	curve := elliptic.Secp256k1()
	const n = 3
	const threshold uint32 = 3
	ids := make([]string, n)
	for i := 0; i < n; i++ {
		ids[i] = alicetss.GetTestID(i)
	}
	dkgResults := runInProcessDKG(t, n, threshold)
	require.Equal(t, dkgResults[0].Rid, dkgResults[1].Rid)

	// ---- First refresh, seeded by the DKG material ----
	shares := make([]*big.Int, n)
	ppks := make([]map[string]*ecpointgrouplaw.ECPoint, n)
	for i := 0; i < n; i++ {
		aRes, err := dkgResults[i].aliceResult(ids)
		require.NoError(t, err)
		shares[i] = aRes.Share
		ppks[i], err = dkgResults[i].toAlicePartialPubKeys(ids)
		require.NoError(t, err)
	}
	ssid1 := computeSSID("refresh-again-round-1", dkgResults[0].Rid)
	refresh1 := runInProcessRefresh(t, ids, threshold, shares, ppks, dkgResults[0].toAliceBks(ids),
		mustGroupPublicKey(t, dkgResults[0]), ssid1)
	for i := 0; i < n; i++ {
		require.Equal(t, ssid1, refresh1[i].Ssid)
	}

	// ---- Second refresh, seeded by the first refresh ----
	// Corrupt the DKG generation: a further refresh must use the previous refresh's share and ppks,
	// not the ones the previous round already rotated away from.
	for i := 0; i < n; i++ {
		dkgResults[i].Share = big.NewInt(1).Bytes()
		for id := range dkgResults[i].PartialPubKeys {
			dkgResults[i].PartialPubKeys[id] = fromECPoint(
				ecpointgrouplaw.ScalarBaseMult(curve, big.NewInt(99)))
		}
	}
	shares2 := make([]*big.Int, n)
	ppks2 := make([]map[string]*ecpointgrouplaw.ECPoint, n)
	for i := 0; i < n; i++ {
		share, prevPPKs, err := refresh1[i].refreshInputs(dkgResults[i], ids, threshold)
		require.NoError(t, err)
		shares2[i] = share
		ppks2[i] = prevPPKs
	}
	ssid2 := computeRefreshAgainSSID("refresh-again-round-2", refresh1[0].Ssid, dkgResults[0].Rid)
	require.NotEqual(t, ssid1, ssid2)
	refresh2 := runInProcessRefresh(t, ids, threshold, shares2, ppks2, dkgResults[0].toAliceBks(ids),
		mustGroupPublicKey(t, dkgResults[0]), ssid2)
	for i := 0; i < n; i++ {
		require.Equal(t, ssid2, refresh2[i].Ssid)
		require.NotEqual(t, refresh1[i].Share, refresh2[i].Share, "the share must actually rotate")
	}

	// ---- The second generation must sign ----
	msg := sha256.Sum256([]byte("cggmp-refresh-twice"))
	coreMap := make(map[string]alicetypes.MessageMain, n)
	cores := make([]*alicesign.Sign, n)
	listeners := make([]*phaseListener, n)
	for i := 0; i < n; i++ {
		aRes, err := dkgResults[i].aliceResult(ids)
		require.NoError(t, err)
		share := new(big.Int).SetBytes(refresh2[i].Share)
		ppks, err := refresh2[i].toAlicePartialPubKeys(ids)
		require.NoError(t, err)
		ped := refresh2[i].toAlicePedersen(ids)
		paillierKey, err := refresh2[i].ownPaillier()
		require.NoError(t, err)
		ssid := computeSSID("refresh-again-sign", dkgResults[i].Rid)
		pm := alicetss.NewTestPeerManager(i, n)
		pm.Set(coreMap)
		l := newPhaseListener()
		core, err := alicesign.NewSign(threshold, ssid, share, aRes.PublicKey, ppks, paillierKey,
			ped, aRes.Bks, msg[:], pm, l)
		require.NoError(t, err)
		coreMap[ids[i]] = core
		cores[i] = core
		listeners[i] = l
	}
	for _, c := range cores {
		c.Start()
	}
	for _, l := range listeners {
		l.wait(t)
	}
	for i := 0; i < n; i++ {
		res, err := cores[i].GetResult()
		require.NoError(t, err)
		cores[i].Stop()
		sig, err := ToBtcecSignature(res)
		require.NoError(t, err)
		require.True(t, sig.Verify(msg[:], dkgResultPublicKey(t, dkgResults[0])),
			"node %d signature from the second refresh generation must verify", i)
	}
}

// mustGroupPublicKey returns the DKG group public key, failing the test on invalid coordinates.
func mustGroupPublicKey(t *testing.T, d *DKGResult) *ecpointgrouplaw.ECPoint {
	t.Helper()
	pub, err := d.GroupPublicKey()
	require.NoError(t, err)
	return pub
}
