package utils

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/x509"

	"errors"
	"fmt"

	"github.com/tjfoc/gmsm/sm2"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/cert/authority/tools/cryptogen/factory/csp"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/cert/authority/tools/cryptogen/factory/signer"
)

func getCSPFromOpts(KeyStorePath string) (csp.CSP, error) {
	if KeyStorePath == "" {
		return nil, errors.New("Invalid config. It must not be nil.")
	}

	fks, err := csp.NewFileBasedKeyStore(nil, KeyStorePath, false)
	if err != nil {
		return nil, fmt.Errorf("Failed to initialize software key store: %s", err)
	}

	return csp.New(fks)
}

func GeneratePrivateKey(keystorePath string, opt int) (csp.Key, crypto.Signer, error) {
	var err error
	var priv csp.Key
	var s crypto.Signer

	lcscp, err := getCSPFromOpts(keystorePath)
	if err != nil {
		return nil, nil, err
	}

	priv, err = lcscp.KeyGen(opt)
	if err == nil {
		s, err = signer.New(lcscp, priv)
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

func GetSM2PublicKey(priv csp.Key) (*sm2.PublicKey, error) {
	pubKey, err := priv.PublicKey()
	if err != nil {
		return nil, err
	}

	pubKeyBytes, err := pubKey.Bytes()
	if err != nil {
		return nil, err
	}

	sm2PubKey, err := sm2.ParseSm2PublicKey(pubKeyBytes)
	if err != nil {
		return nil, err
	}
	return sm2PubKey, nil
}
