// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package btcutil

import "crypto/ecdsa"
import "math/big"

// ECDH Calculate a shared secret using elliptic curve Diffie-Hellman
func ECDH(priv *ecdsa.PrivateKey, pub *ecdsa.PublicKey) *big.Int {
	x, _ := Secp256k1().ScalarMult(pub.X, pub.Y, priv.D.Bytes())
	return x
}
