// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tss

import (
	"fmt"
	"math/big"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
)

// BigIntToModNScalar converts a big-endian big.Int into a fixed 32-byte btcec.ModNScalar.
// It is shared by the gg18 and cggmp wrappers when turning an alice (R,S) result into a
// btcec signature.
func BigIntToModNScalar(val *big.Int) (*btcec.ModNScalar, error) {
	var scalar btcec.ModNScalar

	b := val.Bytes()
	if len(b) > 32 {
		return nil, fmt.Errorf("value exceeds 32 bytes")
	}
	padded := make([]byte, 32)
	copy(padded[32-len(b):], b)
	// SetByteSlice returns true when the value overflows the modulus.
	if scalar.SetByteSlice(padded) {
		return nil, fmt.Errorf("modulus overflow")
	}
	return &scalar, nil
}

// BuildBtcecSignature converts an (R,S) pair (as returned by the alice gg18/cggmp signers)
// into a btcec ecdsa.Signature. Both gg18.AliceToBtcecSignature and cggmp.ToBtcecSignature
// delegate to this helper so the scalar conversion is defined once.
func BuildBtcecSignature(R, S *big.Int) (*ecdsa.Signature, error) {
	rScalar, err := BigIntToModNScalar(R)
	if err != nil {
		return nil, fmt.Errorf("convert R failed: %w", err)
	}
	sScalar, err := BigIntToModNScalar(S)
	if err != nil {
		return nil, fmt.Errorf("convert S failed: %w", err)
	}
	return ecdsa.NewSignature(rScalar, sScalar), nil
}
