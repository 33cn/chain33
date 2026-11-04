// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cggmp

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/getamis/alice/crypto/ecpointgrouplaw"
	"github.com/getamis/alice/crypto/elliptic"
	alicetss "github.com/getamis/alice/crypto/tss"
	alicedkg "github.com/getamis/alice/crypto/tss/ecdsa/cggmp/dkg"
	alicerefresh "github.com/getamis/alice/crypto/tss/ecdsa/cggmp/refresh"
	alicesign "github.com/getamis/alice/crypto/tss/ecdsa/cggmp/sign"
	alicetypes "github.com/getamis/alice/types"
	"github.com/stretchr/testify/require"
)

const (
	e2eN                   = 3
	e2eThreshold    uint32 = 3
	e2ePhaseTimeout        = 120 * time.Second
)

type phaseListener struct {
	ch chan error
}

func newPhaseListener() *phaseListener {
	return &phaseListener{ch: make(chan error, 1)}
}

func (l *phaseListener) OnStateChanged(oldState, newState alicetypes.MainState) {
	switch newState {
	case alicetypes.StateDone:
		select {
		case l.ch <- nil:
		default:
		}
	case alicetypes.StateFailed:
		select {
		case l.ch <- fmt.Errorf("state failed %s -> %s", oldState.String(), newState.String()):
		default:
		}
	}
}

func (l *phaseListener) wait(t *testing.T) {
	t.Helper()
	select {
	case err := <-l.ch:
		require.NoError(t, err)
	case <-time.After(e2ePhaseTimeout):
		t.Fatal("phase timed out")
	}
}

// jsonRoundTrip marshals and unmarshals v, proving the persisted type fully survives
// (DB) serialization and that everything a later phase needs is captured.
func jsonRoundTrip[T any](t *testing.T, v *T) *T {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	var back T
	require.NoError(t, json.Unmarshal(b, &back))
	return &back
}

func dkgResultPublicKey(t *testing.T, d *DKGResult) *btcec.PublicKey {
	t.Helper()
	var fieldX, fieldY btcec.FieldVal
	require.False(t, fieldX.SetByteSlice(d.PubX), "invalid pubX")
	require.False(t, fieldY.SetByteSlice(d.PubY), "invalid pubY")
	return btcec.NewPublicKey(&fieldX, &fieldY)
}

// partialPublicKeyFor returns this node's partial public key g^{share}.
func partialPublicKeyFor(curve elliptic.Curve, share *big.Int) *ECPoint {
	return fromECPoint(ecpointgrouplaw.ScalarBaseMult(curve, share))
}

// TestCGGMP3PartyInProcess drives the full alice CGGMP flow DKG -> partial public key
// exchange -> refresh -> sign with three in-process parties, round-tripping every phase
// result through the persisted (JSON) chain33 representation. This validates that the
// persisted DKGResult/RefreshResult carry everything ProcessRefresh/ProcessSign need and
// that CGGMP on alice v1.0.7 produces a signature verifiable against the DKG group key.
func TestCGGMP3PartyInProcess(t *testing.T) {
	if testing.Short() {
		t.Skip("skip cggmp crypto e2e in short mode")
	}
	curve := elliptic.Secp256k1()
	ids := make([]string, e2eN)
	for i := 0; i < e2eN; i++ {
		ids[i] = alicetss.GetTestID(i)
	}

	// ---- Phase 1: DKG + partial public key exchange ----
	dkgResults := make([]*DKGResult, e2eN)
	{
		coreMap := make(map[string]alicetypes.MessageMain, e2eN)
		cores := make([]*alicedkg.DKG, e2eN)
		listeners := make([]*phaseListener, e2eN)
		for i := 0; i < e2eN; i++ {
			pm := alicetss.NewTestPeerManager(i, e2eN)
			pm.Set(coreMap)
			l := newPhaseListener()
			core, err := alicedkg.NewDKG(curve, pm, []byte("cggmp-dkg-session"), e2eThreshold, 0, l)
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
		allPPK := make(map[string]*ECPoint, e2eN)
		for i := 0; i < e2eN; i++ {
			res, err := cores[i].GetResult()
			require.NoError(t, err)
			cores[i].Stop()
			// self partial public key = g^{share_i}
			allPPK[ids[i]] = partialPublicKeyFor(curve, res.Share)
			dkgResults[i] = newDKGResult(res, nil)
		}
		// All parties hold the same group public key / bks / rid but their own share.
		for i := 0; i < e2eN; i++ {
			dkgResults[i].PartialPubKeys = make(map[string]*ECPoint, e2eN)
			for j := 0; j < e2eN; j++ {
				dkgResults[i].PartialPubKeys[ids[j]] = allPPK[ids[j]]
			}
			dkgResults[i] = jsonRoundTrip(t, dkgResults[i])
		}
		require.Equal(t, dkgResults[0].Rid, dkgResults[1].Rid)
		require.Equal(t, dkgResults[0].Rid, dkgResults[2].Rid)
	}

	// ---- Phase 2: refresh (key refresh + Paillier/Pedersen provisioning) ----
	refreshResults := make([]*RefreshResult, e2eN)
	{
		coreMap := make(map[string]alicetypes.MessageMain, e2eN)
		cores := make([]*alicerefresh.Refresh, e2eN)
		listeners := make([]*phaseListener, e2eN)
		for i := 0; i < e2eN; i++ {
			aRes, err := dkgResults[i].aliceResult(ids)
			require.NoError(t, err)
			ppks, err := dkgResults[i].toAlicePartialPubKeys(ids)
			require.NoError(t, err)
			ssid := computeSSID("cggmp-refresh-session", dkgResults[i].Rid)
			pm := alicetss.NewTestPeerManager(i, e2eN)
			pm.Set(coreMap)
			l := newPhaseListener()
			core, err := alicerefresh.NewRefresh(aRes.Share, aRes.PublicKey, pm, e2eThreshold, ppks, aRes.Bks, 2048, ssid, l)
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
		for i := 0; i < e2eN; i++ {
			res, err := cores[i].GetResult()
			require.NoError(t, err)
			cores[i].Stop()
			rr, err := newRefreshResult(res)
			require.NoError(t, err)
			refreshResults[i] = jsonRoundTrip(t, rr)
		}
	}

	// ---- Phase 3: sign (4-round) + verify ----
	msg := sha256.Sum256([]byte("cggmp-in-process-e2e"))
	{
		coreMap := make(map[string]alicetypes.MessageMain, e2eN)
		cores := make([]*alicesign.Sign, e2eN)
		listeners := make([]*phaseListener, e2eN)
		for i := 0; i < e2eN; i++ {
			aRes, err := dkgResults[i].aliceResult(ids)
			require.NoError(t, err)
			share := new(big.Int).SetBytes(refreshResults[i].Share)
			ppks, err := refreshResults[i].toAlicePartialPubKeys(ids)
			require.NoError(t, err)
			ped := refreshResults[i].toAlicePedersen(ids)
			paillierKey, err := refreshResults[i].ownPaillier()
			require.NoError(t, err)
			ssid := computeSSID("cggmp-sign-session", dkgResults[i].Rid)
			pm := alicetss.NewTestPeerManager(i, e2eN)
			pm.Set(coreMap)
			l := newPhaseListener()
			core, err := alicesign.NewSign(e2eThreshold, ssid, share, aRes.PublicKey, ppks, paillierKey, ped, aRes.Bks, msg[:], pm, l)
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
		for i := 0; i < e2eN; i++ {
			res, err := cores[i].GetResult()
			require.NoError(t, err)
			cores[i].Stop()
			sig, err := ToBtcecSignature(res)
			require.NoError(t, err)
			pub := dkgResultPublicKey(t, dkgResults[0])
			require.True(t, sig.Verify(msg[:], pub), "node %d signature must verify", i)
		}
	}
}
