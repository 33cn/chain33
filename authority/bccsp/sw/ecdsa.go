/*
Copyright IBM Corp. 2016 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
/*
Notice: This file has been modified for Hyperledger Fabric SDK Go usage.
Please review third_party pinning scripts and patches for more details.
*/
package sw

import (
	"crypto/ecdsa"
	"crypto/rand"
	"errors"
	"fmt"

	"crypto/elliptic"
	"encoding/asn1"
	"math/big"

	"gitlab.33.cn/chain33/chain33/authority/bccsp"
	"gitlab.33.cn/chain33/chain33/authority/bccsp/utils"
)

type ECDSASignature struct {
	R, S *big.Int
}

var (
	// curveHalfOrders contains the precomputed curve group orders halved.
	// It is used to ensure that signature' S value is lower or equal to the
	// curve group order halved. We accept only low-S signatures.
	// They are precomputed for efficiency reasons.
	curveHalfOrders map[elliptic.Curve]*big.Int = map[elliptic.Curve]*big.Int{
		elliptic.P224(): new(big.Int).Rsh(elliptic.P224().Params().N, 1),
		elliptic.P256(): new(big.Int).Rsh(elliptic.P256().Params().N, 1),
		elliptic.P384(): new(big.Int).Rsh(elliptic.P384().Params().N, 1),
		elliptic.P521(): new(big.Int).Rsh(elliptic.P521().Params().N, 1),
	}
)

func marshalECDSASignature(r, s *big.Int) ([]byte, error) {
	return asn1.Marshal(ECDSASignature{r, s})
}

func unmarshalECDSASignature(raw []byte) (*big.Int, *big.Int, error) {
	// Unmarshal
	sig := new(ECDSASignature)
	_, err := asn1.Unmarshal(raw, sig)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed unmashalling signature [%s]", err)
	}

	// Validate sig
	if sig.R == nil {
		return nil, nil, errors.New("Invalid signature. R must be different from nil.")
	}
	if sig.S == nil {
		return nil, nil, errors.New("Invalid signature. S must be different from nil.")
	}

	if sig.R.Sign() != 1 {
		return nil, nil, errors.New("Invalid signature. R must be larger than zero")
	}
	if sig.S.Sign() != 1 {
		return nil, nil, errors.New("Invalid signature. S must be larger than zero")
	}

	return sig.R, sig.S, nil
}

func signECDSA(k *ecdsa.PrivateKey, digest []byte, opts bccsp.SignerOpts) (signature []byte, err error) {
	r, s, err := ecdsa.Sign(rand.Reader, k, digest)
	if err != nil {
		return nil, err
	}

	s, _, err = utils.ToLowS(&k.PublicKey, s)
	if err != nil {
		return nil, err
	}

	return utils.MarshalECDSASignature(r, s)
}

func verifyECDSA(k *ecdsa.PublicKey, signature, digest []byte, opts bccsp.SignerOpts) (valid bool, err error) {
	r, s, err := utils.UnmarshalECDSASignature(signature)
	if err != nil {
		return false, fmt.Errorf("Failed unmashalling signature [%s]", err)
	}

	lowS, err := utils.IsLowS(k, s)
	if err != nil {
		return false, err
	}

	if !lowS {
		return false, fmt.Errorf("Invalid S. Must be smaller than half the order [%s][%s].", s, utils.GetCurveHalfOrdersAt(k.Curve))
	}

	return ecdsa.Verify(k, digest, r, s), nil
}

func SignatureToLowS(k *ecdsa.PublicKey, signature []byte) ([]byte, error) {
	r, s, err := unmarshalECDSASignature(signature)
	if err != nil {
		return nil, err
	}

	s, modified, err := toLowS(k, s)
	if err != nil {
		return nil, err
	}

	if modified {
		return marshalECDSASignature(r, s)
	}

	return signature, nil
}

// IsLow checks that s is a low-S
func isLowS(k *ecdsa.PublicKey, s *big.Int) (bool, error) {
	halfOrder, ok := curveHalfOrders[k.Curve]
	if !ok {
		return false, fmt.Errorf("Curve not recognized [%s]", k.Curve)
	}

	return s.Cmp(halfOrder) != 1, nil

}

func toLowS(k *ecdsa.PublicKey, s *big.Int) (*big.Int, bool, error) {
	lowS, err := isLowS(k, s)
	if err != nil {
		return nil, false, err
	}

	if !lowS {
		// Set s to N - s that will be then in the lower part of signature space
		// less or equal to half order
		s.Sub(k.Params().N, s)

		return s, true, nil
	}

	return s, false, nil
}

type ecdsaSigner struct{}

func (s *ecdsaSigner) Sign(k bccsp.Key, digest []byte, opts bccsp.SignerOpts) (signature []byte, err error) {
	return signECDSA(k.(*ecdsaPrivateKey).privKey, digest, opts)
}

type ecdsaPrivateKeyVerifier struct{}

func (v *ecdsaPrivateKeyVerifier) Verify(k bccsp.Key, signature, digest []byte, opts bccsp.SignerOpts) (valid bool, err error) {
	return verifyECDSA(&(k.(*ecdsaPrivateKey).privKey.PublicKey), signature, digest, opts)
}

type ecdsaPublicKeyKeyVerifier struct{}

func (v *ecdsaPublicKeyKeyVerifier) Verify(k bccsp.Key, signature, digest []byte, opts bccsp.SignerOpts) (valid bool, err error) {
	return verifyECDSA(k.(*ecdsaPublicKey).pubKey, signature, digest, opts)
}
