/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identitymgr

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"gitlab.33.cn/chain33/chain33/authority/bccsp"
	"gitlab.33.cn/chain33/chain33/authority/bccsp/utils"
	"gitlab.33.cn/chain33/chain33/authority/common/core"

	"fmt"

	"github.com/pkg/errors"
)

// GetPublicKeyFromCert will return public key the from cert
func GetPublicKeyFromCert(cert []byte, cs core.CryptoSuite) (bccsp.Key, error) {

	dcert, _ := pem.Decode(cert)
	if dcert == nil {
		return nil, errors.Errorf("Unable to decode cert bytes [%v]", cert)
	}

	x509Cert, err := x509.ParseCertificate(dcert.Bytes)
	if err != nil {
		return nil, errors.Errorf("Unable to parse cert from decoded bytes: %s", err)
	}

	// get the public key in the right format
	key, err := cs.KeyImport(x509Cert, GetX509PublicKeyImportOpts())
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to import certificate's public key")
	}

	return key, nil
}

// ImportBCCSPKeyFromPEMBytes attempts to create a private BCCSP key from a pem byte slice
func ImportBCCSPKeyFromPEMBytes(keyBuff []byte, myCSP core.CryptoSuite) (bccsp.Key, error) {
	keyFile := "pem bytes"

	key, err := PEMtoPrivateKey(keyBuff, nil)
	if err != nil {
		return nil, errors.WithMessage(err, fmt.Sprintf("Failed parsing private key from %s", keyFile))
	}
	switch key.(type) {
	case *ecdsa.PrivateKey:
		priv, err := PrivateKeyToDER(key.(*ecdsa.PrivateKey))
		if err != nil {
			return nil, errors.WithMessage(err, fmt.Sprintf("Failed to convert ECDSA private key for '%s'", keyFile))
		}
		sk, err := myCSP.KeyImport(priv, GetECDSAPrivateKeyImportOpts())
		if err != nil {
			return nil, errors.WithMessage(err, fmt.Sprintf("Failed to import ECDSA private key for '%s'", keyFile))
		}
		return sk, nil
	case *rsa.PrivateKey:
		return nil, errors.Errorf("Failed to import RSA key from %s; RSA private key import is not supported", keyFile)
	default:
		return nil, errors.Errorf("Failed to import key from %s: invalid secret key type", keyFile)
	}
}

// PEMtoPrivateKey is a bridge for bccsp utils.PEMtoPrivateKey()
func PEMtoPrivateKey(raw []byte, pwd []byte) (interface{}, error) {
	return utils.PEMtoPrivateKey(raw, pwd)
}

// PrivateKeyToDER marshals is bridge for utils.PrivateKeyToDER
func PrivateKeyToDER(privateKey *ecdsa.PrivateKey) ([]byte, error) {
	return utils.PrivateKeyToDER(privateKey)
}

//GetX509PublicKeyImportOpts options for importing public keys from an x509 certificate
func GetX509PublicKeyImportOpts() bccsp.KeyImportOpts {
	return &bccsp.X509PublicKeyImportOpts{}
}

//GetECDSAPrivateKeyImportOpts options for ECDSA secret key importation in DER format
// or PKCS#8 format.
func GetECDSAPrivateKeyImportOpts() bccsp.KeyImportOpts {
	return &bccsp.ECDSAPrivateKeyImportOpts{}
}
