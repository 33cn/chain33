// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bip32

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"

	"io"

	"github.com/33cn/chain33/wallet/bipwallet/basen"
	"github.com/33cn/chain33/wallet/bipwallet/btcutilecc"
	"golang.org/x/crypto/ripemd160"
)

var (
	curve                 = btcutil.Secp256k1()
	curveParams           = curve.Params()
	bitcoinBase58Encoding = basen.NewEncoding("123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz")
)

//
// Hashes
//

func hashSha256(data []byte) []byte {
	hasher := sha256.New()
	_, err := hasher.Write(data)
	if err != nil {
		return nil
	}
	return hasher.Sum(nil)
}

func hashDoubleSha256(data []byte) []byte {
	return hashSha256(hashSha256(data))
}

func hashRipeMD160(data []byte) []byte {
	hasher := ripemd160.New()
	_, err := io.WriteString(hasher, string(data))
	if err != nil {
		return nil
	}
	return hasher.Sum(nil)
}

func hash160(data []byte) []byte {
	return hashRipeMD160(hashSha256(data))
}

//
// Encoding
//

func checksum(data []byte) []byte {
	return hashDoubleSha256(data)[:4]
}

func addChecksumToBytes(data []byte) []byte {
	checksum := checksum(data)
	return append(data, checksum...)
}

func base58Encode(data []byte) []byte {
	return []byte(bitcoinBase58Encoding.EncodeToString(data))
}

// Keys
func publicKeyForPrivateKey(key []byte) []byte {
	return compressPublicKey(curve.ScalarBaseMult(key))
}

func addPublicKeys(key1 []byte, key2 []byte) []byte {
	x1, y1 := expandPublicKey(key1)
	x2, y2 := expandPublicKey(key2)
	return compressPublicKey(curve.Add(x1, y1, x2, y2))
}

func addPrivateKeys(key1 []byte, key2 []byte) []byte {
	var key1Int big.Int
	var key2Int big.Int
	key1Int.SetBytes(key1)
	key2Int.SetBytes(key2)

	key1Int.Add(&key1Int, &key2Int)
	key1Int.Mod(&key1Int, curve.Params().N)

	b := key1Int.Bytes()
	if len(b) < 32 {
		extra := make([]byte, 32-len(b))
		b = append(extra, b...)
	}
	return b
}

func compressPublicKey(x *big.Int, y *big.Int) []byte {
	var key bytes.Buffer

	// Write header; 0x2 for even y value; 0x3 for odd
	err := key.WriteByte(byte(0x2) + byte(y.Bit(0)))
	if err != nil {
		return nil
	}

	// Write X coord; Pad the key so x is aligned with the LSB. Pad size is key length - header size (1) - xBytes size
	xBytes := x.Bytes()
	for i := 0; i < (PublicKeyCompressedLength - 1 - len(xBytes)); i++ {
		err := key.WriteByte(0x0)
		if err != nil {
			return nil
		}
	}
	_, err = key.Write(xBytes)
	if err != nil {
		return nil
	}
	return key.Bytes()
}

func expandPublicKey(key []byte) (*big.Int, *big.Int) {
	params := curveParams
	exp := big.NewInt(1)
	exp.Add(params.P, exp)
	exp.Div(exp, big.NewInt(4))
	x := big.NewInt(0).SetBytes(key[1:33])
	y := big.NewInt(0).SetBytes(key[:1])
	beta := big.NewInt(0)
	// #nosec
	beta.Exp(x, big.NewInt(3), nil)
	beta.Add(beta, big.NewInt(7))
	// #nosec
	beta.Exp(beta, exp, params.P)
	if y.Add(beta, y).Mod(y, big.NewInt(2)).Int64() == 0 {
		y = beta
	} else {
		y = beta.Sub(params.P, beta)
	}
	return x, y
}

func validatePrivateKey(key []byte) error {
	if fmt.Sprintf("%x", key) == "0000000000000000000000000000000000000000000000000000000000000000" || //if the key is zero
		bytes.Compare(key, curveParams.N.Bytes()) >= 0 || //or is outside of the curve
		len(key) != 32 { //or is too short
		return errors.New("Invalid seed")
	}

	return nil
}

func validateChildPublicKey(key []byte) error {
	x, y := expandPublicKey(key)

	if x.Sign() == 0 || y.Sign() == 0 {
		return errors.New("Invalid public key")
	}

	return nil
}

//
// Numerical
//
func uint32Bytes(i uint32) []byte {
	bytes := make([]byte, 4)
	binary.BigEndian.PutUint32(bytes, i)
	return bytes
}
