// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package btcutil

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
)

// BlindSignature Based on algorithm described in An Efficient Blind Signature Scheme
// Based on the Elliptic Curve Discrete Logarithm Problem by
// Nikooghadam and Zakerolhosseini
type BlindSignature struct {
	M, S *big.Int // called m and s in the paper
	F    *ecdsa.PublicKey
}

// BlindVerify 不清楚干嘛的
func BlindVerify(Q *ecdsa.PublicKey, sig *BlindSignature) bool {
	crv := Secp256k1().Params()

	// onlooker verifies signature (§4.5)
	sG := ScalarBaseMult(sig.S)
	rm := new(big.Int).Mul(new(big.Int).Mod(sig.F.X, crv.N), sig.M)
	rm.Mod(rm, crv.N)
	rmQ := ScalarMult(rm, Q)
	rmQplusF := Add(rmQ, sig.F)

	fmt.Println("")
	fmt.Printf("sG      = %x\n", sG.X)
	fmt.Printf("rmQ + F = %x\n", rmQplusF.X)
	return KeysEqual(sG, rmQplusF)
}
