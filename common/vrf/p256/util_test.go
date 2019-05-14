// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package p256

import (
	"testing"

	"github.com/33cn/chain33/common/crypto"
	"github.com/33cn/chain33/common/vrf"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

func Test_GenerateKey(t *testing.T) {
	k, pk := GenerateKey()
	testVRF(t, k, pk)
}

func Test_GenVrfKey(t *testing.T) {
	c, err := crypto.New(types.GetSignName("", types.SECP256K1))
	assert.NoError(t, err)
	priv, err := c.GenKey()
	assert.NoError(t, err)

	vpriv, _, pubKey := GenVrfKey(priv)
	vpub, err := ParseVrfPubKey(pubKey)
	assert.NoError(t, err)
	testVRF(t, vpriv, vpub)
}

func testVRF(t *testing.T, priv vrf.PrivateKey, pub vrf.PublicKey) {
	m1 := []byte("data1")
	m2 := []byte("data2")
	m3 := []byte("data2")
	hash1, proof1 := priv.Evaluate(m1)
	hash2, proof2 := priv.Evaluate(m2)
	hash3, proof3 := priv.Evaluate(m3)
	for _, tc := range []struct {
		m     []byte
		hash  [32]byte
		proof []byte
		err   error
	}{
		{m1, hash1, proof1, nil},
		{m2, hash2, proof2, nil},
		{m3, hash3, proof3, nil},
		{m3, hash3, proof2, nil},
		{m3, hash3, proof1, ErrInvalidVRF},
	} {
		hash, err := pub.ProofToHash(tc.m, tc.proof)
		if got, want := err, tc.err; got != want {
			t.Errorf("ProofToHash(%s, %x): %v, want %v", tc.m, tc.proof, got, want)
		}
		if err != nil {
			continue
		}
		if got, want := hash, tc.hash; got != want {
			t.Errorf("ProofToHash(%s, %x): %x, want %x", tc.m, tc.proof, got, want)
		}
	}
}
