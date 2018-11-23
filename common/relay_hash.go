// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package common

import (
	"crypto/sha256"
	"encoding/hex"
)

//Revers String returns the Hash as the hexadecimal string of the byte-reversed hash.
func (h Hash) Revers() Hash {
	for i := 0; i < hashLength/2; i++ {
		h[i], h[hashLength-1-i] = h[hashLength-1-i], h[i]
	}
	return h
}

//ReversString String returns the Hash as the hexadecimal string of the byte-reversed hash.
func (h Hash) ReversString() string {
	for i := 0; i < hashLength/2; i++ {
		h[i], h[hashLength-1-i] = h[hashLength-1-i], h[i]
	}
	return hex.EncodeToString(h[:])
}

// HashB calculates hash(b) and returns the resulting bytes.
func HashB(b []byte) []byte {
	hash := sha256.Sum256(b)
	return hash[:]
}

// HashH calculates hash(b) and returns the resulting bytes as a Hash.
func HashH(b []byte) Hash {
	return Hash(sha256.Sum256(b))
}

// DoubleHashB calculates hash(hash(b)) and returns the resulting bytes.
func DoubleHashB(b []byte) []byte {
	first := sha256.Sum256(b)
	second := sha256.Sum256(first[:])
	return second[:]
}

// DoubleHashH calculates hash(hash(b)) and returns the resulting bytes as a
// Hash.
func DoubleHashH(b []byte) Hash {
	first := sha256.Sum256(b)
	return Hash(sha256.Sum256(first[:]))
}
