package ecdsa

import (
	"math/big"
	"encoding/asn1"
	"crypto/ecdsa"
	"crypto/elliptic"
	"fmt"
	"errors"
)

func marshalECDSASignature(r, s *big.Int) ([]byte, error) {
	return asn1.Marshal(ECDSASignature{r, s})
}

func unmarshalECDSASignature(raw []byte) (*big.Int, *big.Int, error) {
	// Unmarshal
	sig := new(ECDSASignature)
	_, err := asn1.Unmarshal(raw, sig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed unmashalling signature [%s]", err)
	}

	// Validate sig
	if sig.R == nil {
		return nil, nil, errors.New("invalid signature, R must be different from nil")
	}
	if sig.S == nil {
		return nil, nil, errors.New("invalid signature, S must be different from nil")
	}

	if sig.R.Sign() != 1 {
		return nil, nil, errors.New("invalid signature, R must be larger than zero")
	}
	if sig.S.Sign() != 1 {
		return nil, nil, errors.New("invalid signature, S must be larger than zero")
	}

	return sig.R, sig.S, nil
}

func toLowS(k *ecdsa.PublicKey, s *big.Int) (*big.Int, bool, error) {
	lowS := isLowS(k, s)
	if !lowS {
		// Set s to N - s that will be then in the lower part of signature space
		// less or equal to half order
		s.Sub(k.Params().N, s)

		return s, true, nil
	}

	return s, false, nil
}

// IsLow checks that s is a low-S
func isLowS(k *ecdsa.PublicKey, s *big.Int) (bool) {
	return s.Cmp(new(big.Int).Rsh(elliptic.P256().Params().N, 1)) != 1
}

func isOdd(a *big.Int) bool {
	return a.Bit(0) == 1
}

func parsePubKey(pubKeyStr []byte, curve elliptic.Curve) (key *ecdsa.PublicKey, err error) {
	pubkey := ecdsa.PublicKey{}
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

func serializePublicKey(p *ecdsa.PublicKey) []byte {
	b := make([]byte, 0, ECDSA_PUBLICKEY_LENGTH)
	b = append(b, 0x4)
	b = paddedAppend(32, b, p.X.Bytes())
	return paddedAppend(32, b, p.Y.Bytes())
}

func serializePrivateKey(p *ecdsa.PrivateKey) []byte {
	b := make([]byte, 0, ECDSA_RPIVATEKEY_LENGTH)
	return paddedAppend(ECDSA_RPIVATEKEY_LENGTH, b, p.D.Bytes())
}

func paddedAppend(size uint, dst, src []byte) []byte {
	for i := 0; i < int(size)-len(src); i++ {
		dst = append(dst, 0)
	}
	return append(dst, src...)
}

