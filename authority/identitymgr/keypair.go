/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package identitymgr

import (
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"github.com/pkg/errors"
)

type CryptoKeyPair struct {
	Key  CryptoConfig
	Cert CryptoConfig
}

type CryptoConfig struct {
	// the following two fields are interchangeable.
	// If Path is available, then it will be used to load the cert
	// if Pem is available, then it has the raw data of the cert it will be used as-is
	// Certificate root certificate path
	Path string
	// Certificate actual content
	Pem string
}

// Bytes returns the tls certificate as a byte array by loading it either from the embedded Pem or Path
func (cfg CryptoConfig) Bytes() ([]byte, error) {
	var bytes []byte
	var err error

	if cfg.Pem != "" {
		bytes = []byte(cfg.Pem)
	} else if cfg.Path != "" {
		bytes, err = ioutil.ReadFile(cfg.Path)

		if err != nil {
			return nil, errors.Wrapf(err, "failed to load pem bytes from path %s", cfg.Path)
		}
	}

	return bytes, nil
}

// CryptoCert returns the tls certificate as a *x509.Certificate by loading it either from the embedded Pem or Path
func (cfg CryptoConfig) CryptoCert() (*x509.Certificate, error) {
	bytes, err := cfg.Bytes()

	if err != nil {
		return nil, err
	}

	return loadCert(bytes)
}

// loadCAKey
func loadCert(rawData []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(rawData)

	if block != nil {
		pub, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, errors.Wrap(err, "certificate parsing failed")
		}

		return pub, nil
	}

	// return an error with an error code for clients to test against status.EmptyCert code
	return nil, errors.New("pem data missing")
}
