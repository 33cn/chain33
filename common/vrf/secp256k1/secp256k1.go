// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Copyright 2016 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package secp256k1 implements a verifiable random function using curve secp256k1.
package secp256k1

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"math/big"

	vrfp "github.com/33cn/chain33/common/vrf"
	"github.com/btcsuite/btcd/btcec/v2"
)

var (
	curve = btcec.S256()
	// ErrInvalidVRF err
	ErrInvalidVRF = errors.New("invalid VRF proof")
)

// PublicKey holds a public VRF key.
type PublicKey struct {
	*ecdsa.PublicKey
}

// PrivateKey holds a private VRF key.
type PrivateKey struct {
	*ecdsa.PrivateKey
}

// GenerateKey generates a fresh keypair for this VRF
func GenerateKey() (vrfp.PrivateKey, vrfp.PublicKey) {
	key, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nil, nil
	}

	return &PrivateKey{PrivateKey: key}, &PublicKey{PublicKey: &key.PublicKey}
}

// H1 hashes m to a curve point
func H1(m []byte) (x, y *big.Int) {
	h := sha512.New()
	var i uint32
	byteLen := (curve.BitSize + 7) >> 3
	for x == nil && i < 100 {
		// TODO: Use a NIST specified DRBG.
		h.Reset()
		if err := binary.Write(h, binary.BigEndian, i); err != nil {
			panic(err)
		}
		if _, err := h.Write(m); err != nil {
			panic(err)
		}
		r := []byte{2} // Set point encoding to "compressed", y=0.
		r = h.Sum(r)
		x, y = Unmarshal(curve, r[:byteLen+1])
		i++
	}
	return
}

var one = big.NewInt(1)

// H2 hashes to an integer [1,N-1]
func H2(m []byte) *big.Int {
	// NIST SP 800-90A § A.5.1: Simple discard method.
	byteLen := (curve.BitSize + 7) >> 3
	h := sha512.New()
	for i := uint32(0); ; i++ {
		// TODO: Use a NIST specified DRBG.
		h.Reset()
		if err := binary.Write(h, binary.BigEndian, i); err != nil {
			panic(err)
		}
		if _, err := h.Write(m); err != nil {
			panic(err)
		}
		b := h.Sum(nil)
		k := new(big.Int).SetBytes(b[:byteLen])
		if k.Cmp(new(big.Int).Sub(curve.N, one)) == -1 {
			return k.Add(k, one)
		}
	}
}

// Evaluate returns the verifiable unpredictable function evaluated at m
func (k PrivateKey) Evaluate(m []byte) (index [32]byte, proof []byte) {
	nilIndex := [32]byte{}
	// Prover chooses r <-- [1,N-1]
	r, _, _, err := elliptic.GenerateKey(curve, rand.Reader)
	if err != nil {
		return nilIndex, nil
	}
	ri := new(big.Int).SetBytes(r)

	// H = H1(m)
	Hx, Hy := H1(m)

	// VRF_k(m) = [k]H
	sHx, sHy := curve.ScalarMult(Hx, Hy, k.D.Bytes())
	vrf := elliptic.Marshal(curve, sHx, sHy) // 65 bytes.

	// G is the base point
	// s = H2(G, H, [k]G, VRF, [r]G, [r]H)
	rGx, rGy := curve.ScalarBaseMult(r)
	rHx, rHy := curve.ScalarMult(Hx, Hy, r)
	var b bytes.Buffer
	if _, err := b.Write(elliptic.Marshal(curve, curve.Gx, curve.Gy)); err != nil {
		panic(err)
	}
	if _, err := b.Write(elliptic.Marshal(curve, Hx, Hy)); err != nil {
		panic(err)
	}
	if _, err := b.Write(elliptic.Marshal(curve, k.PublicKey.X, k.PublicKey.Y)); err != nil {
		panic(err)
	}
	if _, err := b.Write(vrf); err != nil {
		panic(err)
	}
	if _, err := b.Write(elliptic.Marshal(curve, rGx, rGy)); err != nil {
		panic(err)
	}
	if _, err := b.Write(elliptic.Marshal(curve, rHx, rHy)); err != nil {
		panic(err)
	}
	s := H2(b.Bytes())

	// t = r−s*k mod N
	t := new(big.Int).Sub(ri, new(big.Int).Mul(s, k.D))
	t.Mod(t, curve.N)

	// Index = H(vrf)
	index = sha256.Sum256(vrf)

	// Write s, t, and vrf to a proof blob. Also write leading zeros before s and t
	// if needed.
	var buf bytes.Buffer
	if _, err := buf.Write(make([]byte, 32-len(s.Bytes()))); err != nil {
		panic(err)
	}
	if _, err := buf.Write(s.Bytes()); err != nil {
		panic(err)
	}
	if _, err := buf.Write(make([]byte, 32-len(t.Bytes()))); err != nil {
		panic(err)
	}
	if _, err := buf.Write(t.Bytes()); err != nil {
		panic(err)
	}
	if _, err := buf.Write(vrf); err != nil {
		panic(err)
	}

	return index, buf.Bytes()
}

// ProofToHash asserts that proof is correct for m and outputs index.
func (pk *PublicKey) ProofToHash(m, proof []byte) (index [32]byte, err error) {
	nilIndex := [32]byte{}
	// verifier checks that s == H2(m, [t]G + [s]([k]G), [t]H1(m) + [s]VRF_k(m))
	if got, want := len(proof), 64+65; got != want {
		return nilIndex, ErrInvalidVRF
	}

	// Parse proof into s, t, and vrf.
	s := proof[0:32]
	t := proof[32:64]
	vrf := proof[64 : 64+65]

	uHx, uHy := elliptic.Unmarshal(curve, vrf)
	if uHx == nil {
		return nilIndex, ErrInvalidVRF
	}

	// [t]G + [s]([k]G) = [t+ks]G
	tGx, tGy := curve.ScalarBaseMult(t)
	ksGx, ksGy := curve.ScalarMult(pk.X, pk.Y, s)
	tksGx, tksGy := curve.Add(tGx, tGy, ksGx, ksGy)

	// H = H1(m)
	// [t]H + [s]VRF = [t+ks]H
	Hx, Hy := H1(m)
	tHx, tHy := curve.ScalarMult(Hx, Hy, t)
	sHx, sHy := curve.ScalarMult(uHx, uHy, s)
	tksHx, tksHy := curve.Add(tHx, tHy, sHx, sHy)

	//   H2(G, H, [k]G, VRF, [t]G + [s]([k]G), [t]H + [s]VRF)
	// = H2(G, H, [k]G, VRF, [t+ks]G, [t+ks]H)
	// = H2(G, H, [k]G, VRF, [r]G, [r]H)
	var b bytes.Buffer
	if _, err := b.Write(elliptic.Marshal(curve, curve.Gx, curve.Gy)); err != nil {
		panic(err)
	}
	if _, err := b.Write(elliptic.Marshal(curve, Hx, Hy)); err != nil {
		panic(err)
	}
	if _, err := b.Write(elliptic.Marshal(curve, pk.X, pk.Y)); err != nil {
		panic(err)
	}
	if _, err := b.Write(vrf); err != nil {
		panic(err)
	}
	if _, err := b.Write(elliptic.Marshal(curve, tksGx, tksGy)); err != nil {
		panic(err)
	}
	if _, err := b.Write(elliptic.Marshal(curve, tksHx, tksHy)); err != nil {
		panic(err)
	}
	h2 := H2(b.Bytes())

	// Left pad h2 with zeros if needed. This will ensure that h2 is padded
	// the same way s is.
	var buf bytes.Buffer
	if _, err := buf.Write(make([]byte, 32-len(h2.Bytes()))); err != nil {
		panic(err)
	}
	if _, err := buf.Write(h2.Bytes()); err != nil {
		panic(err)
	}

	if !hmac.Equal(s, buf.Bytes()) {
		return nilIndex, ErrInvalidVRF
	}
	return sha256.Sum256(vrf), nil
}

// Public returns the corresponding public key as bytes.
func (k PrivateKey) Public() crypto.PublicKey {
	return &k.PublicKey
}
