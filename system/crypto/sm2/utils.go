// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sm2

import (
	"crypto/elliptic"
	"errors"
	"fmt"
	"math/big"

	"github.com/btcsuite/btcd/btcec"
	"github.com/tjfoc/gmsm/sm2"
)

//Signature 签名
type Signature struct {
	R, S *big.Int
}

func canonicalizeInt(val *big.Int) []byte {
	b := val.Bytes()
	if len(b) == 0 {
		b = []byte{0x00}
	}
	if b[0]&0x80 != 0 {
		paddedBytes := make([]byte, len(b)+1)
		copy(paddedBytes[1:], b)
		b = paddedBytes
	}
	return b
}

//Serialize 序列化
func Serialize(r, s *big.Int) []byte {
	rb := canonicalizeInt(r)
	sb := canonicalizeInt(s)

	length := 6 + len(rb) + len(sb)
	b := make([]byte, length)

	b[0] = 0x30
	b[1] = byte(length - 2)
	b[2] = 0x02
	b[3] = byte(len(rb))
	offset := copy(b[4:], rb) + 4
	b[offset] = 0x02
	b[offset+1] = byte(len(sb))
	copy(b[offset+2:], sb)

	return b
}

//Deserialize 反序列化
func Deserialize(sigStr []byte) (*big.Int, *big.Int, error) {
	sig, err := btcec.ParseDERSignature(sigStr, sm2.P256Sm2())
	if err != nil {
		return nil, nil, err
	}

	return sig.R, sig.S, nil
}

//ToLowS ...
func ToLowS(k *sm2.PublicKey, s *big.Int) *big.Int {
	lowS := IsLowS(s)
	if !lowS && k.Curve != sm2.P256Sm2() {
		s.Sub(k.Params().N, s)

		return s
	}

	return s
}

//IsLowS ...
func IsLowS(s *big.Int) bool {
	return s.Cmp(new(big.Int).Rsh(sm2.P256Sm2().Params().N, 1)) != 1
}

func parsePubKey(pubKeyStr []byte, curve elliptic.Curve) (key *sm2.PublicKey, err error) {
	pubkey := sm2.PublicKey{}
	pubkey.Curve = curve

	if len(pubKeyStr) == 0 {
		return nil, errors.New("pubkey string is empty")
	}

	pubkey.X = new(big.Int).SetBytes(pubKeyStr[1:33])
	pubkey.Y = new(big.Int).SetBytes(pubKeyStr[33:])
	if pubkey.X.Cmp(pubkey.Curve.Params().P) >= 0 {
		return nil, fmt.Errorf("pubkey X parameter is >= to P")
	}
	if pubkey.Y.Cmp(pubkey.Curve.Params().P) >= 0 {
		return nil, fmt.Errorf("pubkey Y parameter is >= to P")
	}
	if !pubkey.Curve.IsOnCurve(pubkey.X, pubkey.Y) {
		return nil, fmt.Errorf("pubkey isn't on secp256k1 curve")
	}
	return &pubkey, nil
}

//SerializePublicKey 公钥序列化
func SerializePublicKey(p *sm2.PublicKey) []byte {
	b := make([]byte, 0, SM2PublicKeyLength)
	b = append(b, 0x4)
	b = paddedAppend(32, b, p.X.Bytes())
	return paddedAppend(32, b, p.Y.Bytes())
}

//SerializePrivateKey 私钥序列化
func SerializePrivateKey(p *sm2.PrivateKey) []byte {
	b := make([]byte, 0, SM2PrivateKeyLength)
	return paddedAppend(SM2PrivateKeyLength, b, p.D.Bytes())
}

func paddedAppend(size uint, dst, src []byte) []byte {
	for i := 0; i < int(size)-len(src); i++ {
		dst = append(dst, 0)
	}
	return append(dst, src...)
}
