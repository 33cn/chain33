// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cggmp

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/33cn/chain33/system/crypto/tss"
	"github.com/33cn/chain33/system/crypto/tss/gg18"
	"github.com/33cn/chain33/types"
	"github.com/getamis/alice/crypto/birkhoffinterpolation"
	"github.com/getamis/alice/crypto/ecpointgrouplaw"
	"github.com/getamis/alice/crypto/elliptic"
	alicedkg "github.com/getamis/alice/crypto/tss/ecdsa/cggmp/dkg"
	alicesign "github.com/getamis/alice/crypto/tss/ecdsa/cggmp/sign"
	"github.com/stretchr/testify/require"
)

// TestProtocolsIndependent asserts the CGGMP protocol tags are unique and disjoint from the
// GG18 tags so the two wrappers can coexist in one process.
func TestProtocolsIndependent(t *testing.T) {
	gg18Tags := []string{gg18.DkgProtocol, gg18.SignProtocol, gg18.ReshareProtocol}
	cggmpTags := []string{DkgProtocol, RefreshProtocol, SignProtocol, PpkProtocol}

	seen := map[string]bool{}
	for _, tag := range append(gg18Tags, cggmpTags...) {
		require.False(t, seen[tag], "duplicate protocol tag %q", tag)
		seen[tag] = true
	}
	require.NotContains(t, gg18Tags, DkgProtocol)
	require.NotContains(t, gg18Tags, SignProtocol)
}

func TestDKGResultConversions(t *testing.T) {
	curve := elliptic.Secp256k1()
	id := "peer-0"

	pub := ecpointgrouplaw.ScalarBaseMult(curve, big.NewInt(42))
	bks := map[string]*birkhoffinterpolation.BkParameter{
		id: birkhoffinterpolation.NewBkParameter(big.NewInt(1), 0),
	}
	ppk := map[string]*ecpointgrouplaw.ECPoint{
		id: ecpointgrouplaw.ScalarBaseMult(curve, big.NewInt(7)),
	}

	res := &alicedkg.Result{
		PublicKey: pub,
		Share:     big.NewInt(7),
		Bks:       bks,
		Rid:       []byte("rid-1"),
	}
	d := newDKGResult(res, ppk)
	require.Equal(t, []byte("rid-1"), d.Rid)
	require.Equal(t, big.NewInt(7).Bytes(), d.Share)
	require.Equal(t, pub.GetX().Bytes(), d.PubX)
	require.Equal(t, pub.GetY().Bytes(), d.PubY)
	require.Len(t, d.PartialPubKeys, 1)

	// Round-trip: reconstruct the alice result restricted to the participant list.
	back, err := d.aliceResult([]string{id})
	require.NoError(t, err)
	require.Equal(t, res.PublicKey.GetX(), back.PublicKey.GetX())
	require.Equal(t, res.PublicKey.GetY(), back.PublicKey.GetY())
	require.Equal(t, res.Share, back.Share)
	require.Equal(t, res.Rid, back.Rid)
	require.Contains(t, back.Bks, id)

	// The group public key must match the alice group key.
	gpk, err := d.GroupPublicKey()
	require.NoError(t, err)
	require.Equal(t, pub.GetX(), gpk.GetX())
	require.Equal(t, pub.GetY(), gpk.GetY())

	// SelfBkString must match the (v1.0.7) BkParameter.String(fieldOrder) format.
	s, err := d.SelfBkString(id, curve.Params().N)
	require.NoError(t, err)
	require.Equal(t, bks[id].String(curve.Params().N), s)
	_, err = d.SelfBkString("missing", curve.Params().N)
	require.Error(t, err)
}

func TestSignatureConversion(t *testing.T) {
	R := big.NewInt(123456789)
	S := big.NewInt(987654321)

	sig, err := ToBtcecSignature(&alicesign.Result{R: R, S: S})
	require.NoError(t, err)
	require.NotNil(t, sig)

	// R above the secp256k1 order overflows the ModNScalar.
	hugeR := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1)) // 2^256-1
	_, err = ToBtcecSignature(&alicesign.Result{R: hugeR, S: S})
	require.Error(t, err)

	_, err = ToBtcecSignature(nil)
	require.Error(t, err)
}

// TestPPKCollectorAttributesByPeerID asserts the partial public key exchange is attributed by
// the authenticated transport peer id and not by the peer-supplied Sender field: a single peer
// that sends several messages with different Sender values must not satisfy the quorum on its
// own (which would let it complete the exchange early with forged contributions).
func TestPPKCollectorAttributesByPeerID(t *testing.T) {
	curve := elliptic.Secp256k1()
	collector := newPPKCollector(2)

	for _, sender := range []string{"peer-a", "peer-b", "peer-c"} {
		addPPKToCollector(collector, "peer-real", &PartialPublicKey{
			Sender: sender,
			X:      ecpointgrouplaw.ScalarBaseMult(curve, big.NewInt(1)).GetX().Bytes(),
			Y:      ecpointgrouplaw.ScalarBaseMult(curve, big.NewInt(1)).GetY().Bytes(),
		})
	}
	// One transport peer, three forged Sender values: still a single contribution.
	require.Len(t, collector.result(), 1)
	select {
	case <-collector.done:
		t.Fatal("collector must not complete on forged Sender values")
	default:
	}
	require.Error(t, collector.wait(10*time.Millisecond))

	// A second, distinct peer completes it.
	addPPKToCollector(collector, "peer-other", &PartialPublicKey{
		Sender: "peer-real", // even claiming another peer's id changes nothing
		X:      ecpointgrouplaw.ScalarBaseMult(curve, big.NewInt(2)).GetX().Bytes(),
		Y:      ecpointgrouplaw.ScalarBaseMult(curve, big.NewInt(2)).GetY().Bytes(),
	})
	require.Len(t, collector.result(), 2)
	require.NoError(t, collector.wait(time.Second))
}

// TestPendingPPKBounded asserts the pre-collector ppk buffer stays bounded: at most one entry
// per authenticated peer, at most maxPendingPPKPerSession participants per session, and at most
// maxPendingPPKSessions sessions — a peer cannot grow it with duplicates or random session ids.
func TestPendingPPKBounded(t *testing.T) {
	first := func() int {
		ppkMu.Lock()
		defer ppkMu.Unlock()
		return len(pendingPPK)
	}
	require.Equal(t, 0, first())
	defer func() {
		ppkMu.Lock()
		pendingPPK = make(map[string][]pendingPPKMsg)
		ppkMu.Unlock()
	}()
	snapshot := func(sessionID string) int {
		ppkMu.Lock()
		defer ppkMu.Unlock()
		return len(pendingPPK[tss.ComposeProtocol(PpkProtocol, sessionID)])
	}
	send := func(sessionID, peerID, sender string) {
		handlePpkMsg(&tss.MessageWrapper{
			Protocol: PpkProtocol, SessionID: sessionID, PeerID: peerID,
			Msg: types.Encode(&PartialPublicKey{Sender: sender, X: []byte{1}, Y: []byte{1}}),
		})
	}

	// One peer sending several points (with different Sender values each time) to a session
	// with no collector yet: attributed by peer id, so it collapses onto a single entry.
	for _, sender := range []string{"peer-a", "peer-b", "peer-c"} {
		send("ppk-flood-session", "peer-real", sender)
	}
	require.Equal(t, 1, snapshot("ppk-flood-session"))

	// Distinct participants are capped per session.
	for i := 0; i < maxPendingPPKPerSession+10; i++ {
		peer := fmt.Sprintf("peer-%d", i)
		send("ppk-flood-session", peer, peer)
	}
	require.Equal(t, maxPendingPPKPerSession, snapshot("ppk-flood-session"))

	// Many sessions, one message each: the session count stays capped.
	for i := 0; i < maxPendingPPKSessions+20; i++ {
		peer := fmt.Sprintf("peer-%d", i)
		send(fmt.Sprintf("ppk-flood-%d", i), peer, peer)
	}
	require.LessOrEqual(t, first(), maxPendingPPKSessions, "pending ppk sessions must stay bounded")
}

// testTwoPartyDKGResult builds a DKG result for two participants that is internally consistent:
// f(x) = a + b*x is the shared degree-1 polynomial, the peer at x=1 holds a rank-0 share
// (f(1) = a+b) and the one at x=2 a rank-1 share (f'(2) = b), so the Birkhoff reconstruction of
// the two partial public keys must give the group public key g^a. This mirrors the rank-0/rank-1
// participant sets the p2p integration test uses.
func testTwoPartyDKGResult() (*DKGResult, []string) {
	curve := elliptic.Secp256k1()
	const a, b = 11, 5
	peers := []string{"peer-rank0", "peer-rank1"}
	pub := ecpointgrouplaw.ScalarBaseMult(curve, big.NewInt(a))
	return &DKGResult{
		PubX:  pub.GetX().Bytes(),
		PubY:  pub.GetY().Bytes(),
		Share: big.NewInt(a + b).Bytes(),
		Bks: map[string]*tss.BK{
			"peer-rank0": {Rank: 0, X: big.NewInt(1).Bytes()},
			"peer-rank1": {Rank: 1, X: big.NewInt(2).Bytes()},
		},
		Rid: []byte("rid-birkhoff"),
		PartialPubKeys: map[string]*ECPoint{
			"peer-rank0": fromECPoint(ecpointgrouplaw.ScalarBaseMult(curve, big.NewInt(a+b))),
			"peer-rank1": fromECPoint(ecpointgrouplaw.ScalarBaseMult(curve, big.NewInt(b))),
		},
	}, peers
}

// TestValidatePartialPubKeys covers the ppk-stage Birkhoff reconstruction: consistent material
// passes, a bad g^{share} is rejected (naming the phase and the peers involved) as soon as the
// ppk exchange completes, and a missing entry names the offending peer.
func TestValidatePartialPubKeys(t *testing.T) {
	dkgRes, peers := testTwoPartyDKGResult()
	require.NoError(t, dkgRes.validatePartialPubKeys(peers, 2))

	// Inject a bad point for one peer: the reconstruction must fail instead of being carried
	// into the refresh phase.
	curve := elliptic.Secp256k1()
	dkgRes.PartialPubKeys["peer-rank1"] = fromECPoint(ecpointgrouplaw.ScalarBaseMult(curve, big.NewInt(99)))
	err := dkgRes.validatePartialPubKeys(peers, 2)
	require.ErrorIs(t, err, errPPKInconsistent)
	require.Contains(t, err.Error(), "ppk")
	require.Contains(t, err.Error(), "peer-rank1")

	// A missing partial public key / bk is reported with the exact peer.
	dkgRes, peers = testTwoPartyDKGResult()
	delete(dkgRes.PartialPubKeys, "peer-rank1")
	err = dkgRes.validatePartialPubKeys(peers, 2)
	require.ErrorIs(t, err, errMissingPeerPPK)
	require.Contains(t, err.Error(), "peer-rank1")

	dkgRes, peers = testTwoPartyDKGResult()
	delete(dkgRes.Bks, "peer-rank0")
	err = dkgRes.validatePartialPubKeys(peers, 2)
	require.ErrorIs(t, err, errMissingPeerBk)
	require.Contains(t, err.Error(), "peer-rank0")

	// Fewer participants than the threshold is rejected too.
	dkgRes, peers = testTwoPartyDKGResult()
	require.Error(t, dkgRes.validatePartialPubKeys(peers, 3))
}

// TestProcessRefreshRejectsBadPPK asserts ProcessRefresh fails on the ppk-stage error instead of
// starting the refresh with corrupted material (both paths return before any network use).
func TestProcessRefreshRejectsBadPPK(t *testing.T) {
	dkgRes, peers := testTwoPartyDKGResult()
	curve := elliptic.Secp256k1()
	dkgRes.PartialPubKeys["peer-rank1"] = fromECPoint(ecpointgrouplaw.ScalarBaseMult(curve, big.NewInt(99)))

	_, err := ProcessRefresh(peers, 2, dkgRes, "refresh-session")
	require.ErrorIs(t, err, errPPKInconsistent)
}

// TestProcessSignValidatesSignerMaterial asserts ProcessSign rejects incomplete signer material
// with a clear error rather than handing the maps to alice, which indexes them without checking
// (a missing entry used to panic inside the alice sign core) and rejects a signer count that
// does not match the threshold.
func TestProcessSignValidatesSignerMaterial(t *testing.T) {
	dkgRes, peers := testTwoPartyDKGResult()
	msg := []byte("cggmp-sign-material-test")
	newRefresh := func() *RefreshResult {
		return &RefreshResult{
			Share: big.NewInt(11).Bytes(),
			PartialPubKeys: map[string]*ECPoint{
				"peer-rank0": dkgRes.PartialPubKeys["peer-rank0"],
				"peer-rank1": dkgRes.PartialPubKeys["peer-rank1"],
			},
			PedParams: map[string]*PedersenParam{
				"peer-rank0": {N: []byte{1}, S: []byte{1}, T: []byte{1}},
				"peer-rank1": {N: []byte{1}, S: []byte{1}, T: []byte{1}},
			},
		}
	}

	// Sanity: complete material passes the entry check (the sign run itself needs the p2p
	// transport and is covered by the integration test).
	require.NoError(t, newRefresh().validateSignMaterial(dkgRes, peers, 2))

	cases := []struct {
		name    string
		mutate  func(r *RefreshResult, d *DKGResult)
		wantErr error
		peer    string
	}{
		{"missing bk", func(_ *RefreshResult, d *DKGResult) { delete(d.Bks, "peer-rank1") }, errMissingPeerBk, "peer-rank1"},
		{"missing partial pubkey", func(r *RefreshResult, _ *DKGResult) {
			delete(r.PartialPubKeys, "peer-rank0")
		}, errMissingPeerPPK, "peer-rank0"},
		{"missing pedersen", func(r *RefreshResult, _ *DKGResult) {
			delete(r.PedParams, "peer-rank1")
		}, errMissingPeerPed, "peer-rank1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, p := testTwoPartyDKGResult()
			r := newRefresh()
			tc.mutate(r, d)
			_, err := ProcessSign(p, 2, msg, d, r, "sign-session")
			require.ErrorIs(t, err, tc.wantErr)
			require.Contains(t, err.Error(), tc.peer)
		})
	}

	// threshold must equal the signer count.
	_, err := ProcessSign(peers, 3, msg, dkgRes, newRefresh(), "sign-session")
	require.ErrorIs(t, err, errThresholdPeers)
}

// TestSharedSignatureBuilder keeps the scalar->btcec conversion single-sourced in the tss
// package (used by both gg18.AliceToBtcecSignature and cggmp.ToBtcecSignature).
func TestSharedSignatureBuilder(t *testing.T) {
	sig, err := tss.BuildBtcecSignature(big.NewInt(1), big.NewInt(2))
	require.NoError(t, err)
	require.NotNil(t, sig)

	tooLarge := new(big.Int).Lsh(big.NewInt(1), 300)
	_, err = tss.BuildBtcecSignature(tooLarge, big.NewInt(2))
	require.Error(t, err)
}
