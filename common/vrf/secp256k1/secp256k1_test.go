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

package secp256k1

import (
	"bytes"
	"crypto/rand"
	"math"
	"testing"
)

func TestH1(t *testing.T) {
	for i := 0; i < 10000; i++ {
		m := make([]byte, 100)
		if _, err := rand.Read(m); err != nil {
			t.Fatalf("Failed generating random message: %v", err)
		}
		x, y := H1(m)
		if x == nil {
			t.Errorf("H1(%v)=%v, want curve point", m, x)
		}
		if got := curve.IsOnCurve(x, y); !got {
			t.Errorf("H1(%v)=[%v, %v], is not on curve", m, x, y)
		}
	}
}

func TestH2(t *testing.T) {
	l := 32
	for i := 0; i < 10000; i++ {
		m := make([]byte, 100)
		if _, err := rand.Read(m); err != nil {
			t.Fatalf("Failed generating random message: %v", err)
		}
		x := H2(m)
		if got := len(x.Bytes()); got < 1 || got > l {
			t.Errorf("len(h2(%v)) = %v, want: 1 <= %v <= %v", m, got, got, l)
		}
	}
}

func TestVRF(t *testing.T) {
	k, pk := GenerateKey()

	m1 := []byte("data1")
	m2 := []byte("data2")
	m3 := []byte("data2")
	index1, proof1 := k.Evaluate(m1)
	index2, proof2 := k.Evaluate(m2)
	index3, proof3 := k.Evaluate(m3)
	for _, tc := range []struct {
		m     []byte
		index [32]byte
		proof []byte
		err   error
	}{
		{m1, index1, proof1, nil},
		{m2, index2, proof2, nil},
		{m3, index3, proof3, nil},
		{m3, index3, proof2, nil},
		{m3, index3, proof1, ErrInvalidVRF},
	} {
		index, err := pk.ProofToHash(tc.m, tc.proof)
		if got, want := err, tc.err; got != want {
			t.Errorf("ProofToHash(%s, %x): %v, want %v", tc.m, tc.proof, got, want)
		}
		if err != nil {
			continue
		}
		if got, want := index, tc.index; got != want {
			t.Errorf("ProofToInex(%s, %x): %x, want %x", tc.m, tc.proof, got, want)
		}
	}
}
func TestRightTruncateProof(t *testing.T) {
	k, pk := GenerateKey()

	data := []byte("data")
	_, proof := k.Evaluate(data)
	proofLen := len(proof)
	for i := 0; i < proofLen; i++ {
		proof = proof[:len(proof)-1]
		if _, err := pk.ProofToHash(data, proof); err == nil {
			t.Errorf("Verify unexpectedly succeeded after truncating %v bytes from the end of proof", i)
		}
	}
}

func TestLeftTruncateProof(t *testing.T) {
	k, pk := GenerateKey()

	data := []byte("data")
	_, proof := k.Evaluate(data)
	proofLen := len(proof)
	for i := 0; i < proofLen; i++ {
		proof = proof[1:]
		if _, err := pk.ProofToHash(data, proof); err == nil {
			t.Errorf("Verify unexpectedly succeeded after truncating %v bytes from the beginning of proof", i)
		}
	}
}

func TestBitFlip(t *testing.T) {
	k, pk := GenerateKey()

	data := []byte("data")
	_, proof := k.Evaluate(data)
	for i := 0; i < len(proof)*8; i++ {
		// Flip bit in position i.
		if _, err := pk.ProofToHash(data, flipBit(proof, i)); err == nil {
			t.Errorf("Verify unexpectedly succeeded after flipping bit %v of vrf", i)
		}
	}
}

func flipBit(a []byte, pos int) []byte {
	index := int(math.Floor(float64(pos) / 8))
	b := a[index]
	b ^= (1 << uint(math.Mod(float64(pos), 8.0)))

	var buf bytes.Buffer
	buf.Write(a[:index])
	buf.Write([]byte{b})
	buf.Write(a[index+1:])
	return buf.Bytes()
}
