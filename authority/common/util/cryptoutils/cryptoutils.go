/*
Copyright SecureKey Technologies Inc. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cryptoutils

import (
	"crypto/x509"
	"encoding/pem"

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
