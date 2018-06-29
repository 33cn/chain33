package csp

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/x509"

	"gitlab.33.cn/chain33/chain33/authority/tools/cryptogen/factory"
	"gitlab.33.cn/chain33/chain33/authority/tools/cryptogen/factory/csp"
	"gitlab.33.cn/chain33/chain33/authority/tools/cryptogen/factory/signer"
)

func GeneratePrivateKey(keystorePath string) (csp.Key,
	crypto.Signer, error) {

	var err error
	var priv csp.Key
	var s crypto.Signer

	opts := &factory.FactoryOpts{
		KeyStorePath: keystorePath,
	}

	lcscp, err := factory.GetCSPFromOpts(opts)
	if err == nil {
		priv, err = lcscp.KeyGen(csp.ECDSAP256KeyGen)
		if err == nil {
			s, err = signer.New(lcscp, priv)
		}
	}
	return priv, s, err
}

func GetECPublicKey(priv csp.Key) (*ecdsa.PublicKey, error) {
	pubKey, err := priv.PublicKey()
	if err != nil {
		return nil, err
	}

	pubKeyBytes, err := pubKey.Bytes()
	if err != nil {
		return nil, err
	}

	ecPubKey, err := x509.ParsePKIXPublicKey(pubKeyBytes)
	if err != nil {
		return nil, err
	}
	return ecPubKey.(*ecdsa.PublicKey), nil
}
