// Copyright (c) 2013-2014 The btcsuite developers
// Copyright (c) 2015 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package base58

import (
	"errors"

	"github.com/dchest/blake256"
)

// ErrChecksum indicates that the checksum of a check-encoded string does not verify against
// the checksum.
var ErrChecksum = errors.New("checksum error")

// ErrInvalidFormat indicates that the check-encoded string has an invalid format.
var ErrInvalidFormat = errors.New("invalid format: version and/or checksum bytes missing")

// checksum: first four bytes of double-BLAKE256.
func checksum(input []byte) (cksum [4]byte) {
	h := blake256.New()
	h.Write(input)
	intermediateHash := h.Sum(nil)
	h.Reset()
	h.Write(intermediateHash)
	finalHash := h.Sum(nil)
	copy(cksum[:], finalHash[:])
	return
}

// CheckEncode prepends two version bytes and appends a four byte checksum.
func CheckEncode(input []byte, version [2]byte) string {
	b := make([]byte, 0, 2+len(input)+4)
	b = append(b, version[:]...)
	b = append(b, input[:]...)
	cksum := checksum(b)
	b = append(b, cksum[:]...)
	return Encode(b)
}

// CheckDecode decodes a string that was encoded with CheckEncode and verifies
// the checksum.
func CheckDecode(input string) (result []byte, version [2]byte, err error) {
	decoded := Decode(input)
	if len(decoded) < 6 {
		return nil, [2]byte{0, 0}, ErrInvalidFormat
	}
	version = [2]byte{decoded[0], decoded[1]}
	var cksum [4]byte
	copy(cksum[:], decoded[len(decoded)-4:])
	if checksum(decoded[:len(decoded)-4]) != cksum {
		return nil, [2]byte{0, 0}, ErrChecksum
	}
	payload := decoded[2 : len(decoded)-4]
	result = append(result, payload...)
	return
}
