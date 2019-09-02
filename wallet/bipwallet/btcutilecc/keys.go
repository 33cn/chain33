// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package btcutil

import (
	"crypto/ecdsa"
	"io"
)

// GenerateKey generates a public and private key pair
func GenerateKey(rand io.Reader) (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(Secp256k1(), rand)
}

// KeysEqual check key equal
func KeysEqual(a, b *ecdsa.PublicKey) bool {
	return a.X.Cmp(b.X) == 0 && a.Y.Cmp(b.Y) == 0
}
