// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cggmp

import (
	"math/big"
	"testing"

	"github.com/33cn/chain33/system/crypto/tss"
	"github.com/33cn/chain33/system/crypto/tss/gg18"
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
