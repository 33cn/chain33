/*
Copyright IBM Corp. 2017 All Rights Reserved.

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
package csp

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/x509"

	"gitlab.33.cn/chain33/chain33/authority/bccsp"
	"gitlab.33.cn/chain33/chain33/authority/bccsp/factory"
	"gitlab.33.cn/chain33/chain33/authority/bccsp/signer"
	"gitlab.33.cn/chain33/chain33/authority/cryptosuite"
)

// GeneratePrivateKey creates a private key and stores it in keystorePath
func GeneratePrivateKey(keystorePath string) (bccsp.Key,
	crypto.Signer, error) {

	var err error
	var priv bccsp.Key
	var s crypto.Signer

	opts := &factory.FactoryOpts{
			HashFamily: "SHA2",
			SecLevel:   256,
	}
	csp, err := factory.GetBCCSPFromOpts(opts)
	var suite cryptosuite.CryptoSuite
	var wrapperkey bccsp.Key

	suite.BCCSP = csp
	if err == nil {
		// generate a key
		priv, err = csp.KeyGen(&bccsp.ECDSAP256KeyGenOpts{})
		if err == nil {
			// create a crypto.Signer
			wrapperkey = cryptosuite.GetKey(priv)
			s, err = signer.New(&suite, wrapperkey)
		}
	}
	return priv, s, err
}

func GetECPublicKey(priv bccsp.Key) (*ecdsa.PublicKey, error) {

	// get the public key
	pubKey, err := priv.PublicKey()
	if err != nil {
		return nil, err
	}
	// marshal to bytes
	pubKeyBytes, err := pubKey.Bytes()
	if err != nil {
		return nil, err
	}
	// unmarshal using pkix
	ecPubKey, err := x509.ParsePKIXPublicKey(pubKeyBytes)
	if err != nil {
		return nil, err
	}
	return ecPubKey.(*ecdsa.PublicKey), nil
}
