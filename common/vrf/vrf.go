// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package vrf defines the interface to a verifiable random function.
package vrf

import (
	"crypto"
)

// A VRF is a pseudorandom function f_k from a secret key k, such that that
// knowledge of k not only enables one to evaluate f_k at for any message m,
// but also to provide an NP-proof that the value f_k(m) is indeed correct
// without compromising the unpredictability of f_k for any m' != m.

// PrivateKey supports evaluating the VRF function.
type PrivateKey interface {
	// Evaluate returns the output of H(f_k(m)) and its proof.
	Evaluate(m []byte) (index [32]byte, proof []byte)
	// Public returns the corresponding public key.
	Public() crypto.PublicKey
}

// PublicKey supports verifying output from the VRF function.
type PublicKey interface {
	// ProofToHash verifies the NP-proof supplied by Proof and outputs Index.
	ProofToHash(m, proof []byte) (index [32]byte, err error)
}
