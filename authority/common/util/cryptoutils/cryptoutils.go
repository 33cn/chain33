/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cryptoutils

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"io"

	"gitlab.33.cn/chain33/chain33/authority/common/providers/core"
	"github.com/pkg/errors"
	log "github.com/inconshreveable/log15"
)

var logger =log.New("module", "autority_cryptoutils")

// GetPublicKeyFromCert will return public key the from cert
func GetPublicKeyFromCert(cert []byte, cs core.CryptoSuite) (core.Key, error) {

	dcert, _ := pem.Decode(cert)
	if dcert == nil {
		return nil, errors.Errorf("Unable to decode cert bytes [%v]", cert)
	}

	x509Cert, err := x509.ParseCertificate(dcert.Bytes)
	if err != nil {
		return nil, errors.Errorf("Unable to parse cert from decoded bytes: %s", err)
	}

	// get the public key in the right format
	key, err := cs.KeyImport(x509Cert, GetX509PublicKeyImportOpts(true))
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to import certificate's public key")
	}

	return key, nil
}

// X509KeyPair will return cer/key pair used for mutual TLS
func X509KeyPair(certPEMBlock []byte, pk core.Key, cs core.CryptoSuite) (tls.Certificate, error) {

	fail := func(err error) (tls.Certificate, error) { return tls.Certificate{}, err }

	var cert tls.Certificate
	for {
		var certDERBlock *pem.Block
		certDERBlock, certPEMBlock = pem.Decode(certPEMBlock)
		if certDERBlock == nil {
			break
		}
		if certDERBlock.Type == "CERTIFICATE" {
			cert.Certificate = append(cert.Certificate, certDERBlock.Bytes)
		} else {
			logger.Debug("Skipping block type: %s", certDERBlock.Type)
		}
	}

	if len(cert.Certificate) == 0 {
		return fail(errors.New("No certs available from bytes"))
	}

	// We are parsing public key for TLS to find its type
	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return fail(err)
	}

	switch x509Cert.PublicKey.(type) {
	case *rsa.PublicKey:
		cert.PrivateKey = &PrivateKey{cs, pk, &rsa.PublicKey{}}
	case *ecdsa.PublicKey:
		cert.PrivateKey = &PrivateKey{cs, pk, &ecdsa.PublicKey{}}
	default:
		return fail(errors.New("tls: unknown public key algorithm"))
	}

	return cert, nil
}

//PrivateKey is signer implementation for golang client TLS
type PrivateKey struct {
	cryptoSuite core.CryptoSuite
	key         core.Key
	publicKey   crypto.PublicKey
}

// Public returns the public key corresponding to private key
func (priv *PrivateKey) Public() crypto.PublicKey {
	return priv.publicKey
}

// Sign signs msg with priv, reading randomness from rand. If opts is a
// *PSSOptions then the PSS algorithm will be used, otherwise PKCS#1 v1.5 will
// be used. This method is intended to support keys where the private part is
// kept in, for example, a hardware module.
func (priv *PrivateKey) Sign(rand io.Reader, msg []byte, opts crypto.SignerOpts) ([]byte, error) {
	if priv.cryptoSuite == nil {
		return nil, errors.New("Crypto suite not set")
	}

	if priv.key == nil {
		return nil, errors.New("Private key not set")
	}

	return priv.cryptoSuite.Sign(priv.key, msg, opts)
}
