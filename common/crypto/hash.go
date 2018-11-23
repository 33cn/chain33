// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package crypto

import (
	"crypto/sha256"

	"github.com/tjfoc/gmsm/sm3"
	"golang.org/x/crypto/ripemd160"
)

//Sha256 加密算法
func Sha256(bytes []byte) []byte {
	hasher := sha256.New()
	hasher.Write(bytes)
	return hasher.Sum(nil)
}

//Ripemd160 加密算法
func Ripemd160(bytes []byte) []byte {
	hasher := ripemd160.New()
	hasher.Write(bytes)
	return hasher.Sum(nil)
}

//Sm3Hash 加密算法
func Sm3Hash(msg []byte) []byte {
	c := sm3.New()
	c.Write(msg)
	return c.Sum(nil)
}
