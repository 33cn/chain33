/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package mspmgr

import (
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"

	"gitlab.33.cn/chain33/chain33/authority/bccsp"
)

type identity struct {
	// id contains the identifier (MSPID and identity identifier) for this instance
	id *IdentityIdentifier

	// cert contains the x.509 certificate that signs the public key of this instance
	cert *x509.Certificate

	// this is the public key of this instance
	pk bccsp.Key

	// reference to the MSP that "owns" this identity
	msp *bccspmsp
}

func newIdentity(id *IdentityIdentifier, cert *x509.Certificate, pk bccsp.Key, msp *bccspmsp) (MSPIdentity, error) {
	mspLogger.Debug("Creating identity instance for ID %s", certToPEM(cert))

	// Sanitize first the certificate
	cert, err := msp.sanitizeCert(cert)
	if err != nil {
		return nil, err
	}
	return &identity{id: id, cert: cert, pk: pk, msp: msp}, nil
}

// GetIdentifier returns the identifier (MSPID/IDID) for this instance
func (id *identity) GetIdentifier() *IdentityIdentifier {
	return id.id
}

// GetMSPIdentifier returns the MSP identifier for this instance
func (id *identity) GetMSPIdentifier() string {
	return id.id.Mspid
}

// IsValid returns nil if this instance is a valid identity or an error otherwise
func (id *identity) Validate() error {
	return id.msp.Validate(id)
}

// Verify checks against a signature and a message
// to determine whether this identity produced the
// signature; it returns nil if so or an error otherwise
func (id *identity) Verify(msg []byte, sig []byte) error {
	// mspLogger.Infof("Verifying signature")

	// Compute Hash
	hashOpt, err := id.getHashOpt(id.msp.cryptoConfig.SignatureHashFamily)
	if err != nil {
		return fmt.Errorf("Failed getting hash function options [%s]", err)
	}

	digest, err := id.msp.bccsp.Hash(msg, hashOpt)
	if err != nil {
		return fmt.Errorf("Failed computing digest [%s]", err)
	}

		mspLogger.Debug("Verify: digest = %s", hex.Dump(digest))
		mspLogger.Debug("Verify: sig = %s", hex.Dump(sig))

	valid, err := id.msp.bccsp.Verify(id.pk, sig, digest, nil)
	if err != nil {
		return fmt.Errorf("Could not determine the validity of the signature, err %s", err)
	} else if !valid {
		return errors.New("The signature is invalid")
	}

	return nil
}

func (id *identity) getHashOpt(hashFamily string) (bccsp.HashOpts, error) {
	switch hashFamily {
	case bccsp.SHA2:
		return bccsp.GetHashOpt(bccsp.SHA256)
	case bccsp.SHA3:
		return bccsp.GetHashOpt(bccsp.SHA3_256)
	}
	return nil, fmt.Errorf("hash famility not recognized [%s]", hashFamily)
}
