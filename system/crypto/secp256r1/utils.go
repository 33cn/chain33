// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package secp256r1

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/asn1"
	"errors"
	"fmt"
	"math/big"
)

const (
	pubkeyCompressed   byte = 0x2 // y_bit + x coord
	pubkeyUncompressed byte = 0x4 // x coord + y coord
)

// MarshalECDSASignature marshal ECDSA signature
func MarshalECDSASignature(r, s *big.Int) ([]byte, error) {
	return asn1.Marshal(signatureECDSA{r, s})
}

// UnmarshalECDSASignature unmarshal ECDSA signature
func UnmarshalECDSASignature(raw []byte) (*big.Int, *big.Int, error) {
	sig := new(signatureECDSA)
	_, err := asn1.Unmarshal(raw, sig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed unmashalling signature [%s]", err)
	}

	if sig.R == nil || sig.S == nil {
		return nil, nil, errors.New("invalid signature, R/S must be different from nil")
	}

	if sig.R.Sign() != 1 || sig.S.Sign() != 1 {
		return nil, nil, errors.New("invalid signature, R/S must be larger than zero")
	}

	return sig.R, sig.S, nil
}

// ToLowS convert to low int
func ToLowS(k *ecdsa.PublicKey, s *big.Int) *big.Int {
	lowS := IsLowS(s)
	if !lowS {
		s.Sub(k.Params().N, s)

		return s
	}

	return s
}

// IsLowS check is low int
func IsLowS(s *big.Int) bool {
	return s.Cmp(new(big.Int).Rsh(elliptic.P256().Params().N, 1)) != 1
}

func isOdd(a *big.Int) bool {
	return a.Bit(0) == 1
}

// SerializePublicKeyCompressed serialize compressed publicKey
func SerializePublicKeyCompressed(p *ecdsa.PublicKey) []byte {
	b := make([]byte, 0, publicKeyECDSALengthCompressed)
	format := pubkeyCompressed
	if isOdd(p.Y) {
		format |= 0x1
	}
	b = append(b, format)
	return paddedAppend(32, b, p.X.Bytes())
}

// y² = x³ - 3x + b
func polynomial(B, P, x *big.Int) *big.Int {
	x3 := new(big.Int).Mul(x, x)
	x3.Mul(x3, x)

	threeX := new(big.Int).Lsh(x, 1)
	threeX.Add(threeX, x)

	x3.Sub(x3, threeX)
	x3.Add(x3, B)
	x3.Mod(x3, P)

	return x3
}

func parsePubKeyCompressed(data []byte) (*ecdsa.PublicKey, error) {
	curve := elliptic.P256()
	byteLen := (curve.Params().BitSize + 7) / 8
	if len(data) != 1+byteLen {
		return nil, errors.New("parsePubKeyCompressed byteLen error")
	}
	if data[0] != 2 && data[0] != 3 { // compressed form
		return nil, errors.New("parsePubKeyCompressed compressed form error")
	}
	p := curve.Params().P
	x := new(big.Int).SetBytes(data[1:])
	if x.Cmp(p) >= 0 {
		return nil, errors.New("parsePubKeyCompressed x data error")
	}

	y := polynomial(curve.Params().B, curve.Params().P, x)
	y = y.ModSqrt(y, p)
	if y == nil {
		return nil, errors.New("parsePubKeyCompressed y data error")
	}
	if byte(y.Bit(0)) != data[0]&1 {
		y.Neg(y).Mod(y, p)
	}
	if !curve.IsOnCurve(x, y) {
		return nil, errors.New("parsePubKeyCompressed IsOnCurve error")
	}

	pubkey := ecdsa.PublicKey{}
	pubkey.Curve = curve
	pubkey.X = x
	pubkey.Y = y
	return &pubkey, nil
}

// SerializePrivateKey serialize private key
func SerializePrivateKey(p *ecdsa.PrivateKey) []byte {
	b := make([]byte, 0, privateKeyECDSALength)
	return paddedAppend(privateKeyECDSALength, b, p.D.Bytes())
}

func paddedAppend(size uint, dst, src []byte) []byte {
	for i := 0; i < int(size)-len(src); i++ {
		dst = append(dst, 0)
	}
	return append(dst, src...)
}
